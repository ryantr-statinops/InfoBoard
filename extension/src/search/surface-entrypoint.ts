import { normalizeTab, type SearchTabItem } from './types.js';
import { searchTabsWithMeta, getActiveFilterToken, type ActiveFilterState, type FilterDefinition } from './scorer.js';
import { MRU_STORAGE_KEY, removeTab } from '../background/mru-manager.js';

export function getFilterChipDescription(key: string): string {
  switch (key) {
    case 'audio': return getI18nMessage('filterChipAudio', 'Playing audio');
    case 'pinned': return getI18nMessage('filterChipPinned', 'Pinned tabs');
    case 'group': return getI18nMessage('filterChipGroup', 'Tab groups');
    case 'win': return getI18nMessage('filterChipWin', 'Current window');
    default: return '';
  }
}

export function renderFilterChips(doc: Document, state: ActiveFilterState, onSelect: (def: FilterDefinition) => void): void {
  const container = doc.getElementById('filter-chips');
  if (!container) return;

  if (!state.isActive || state.matchedFilters.length === 0) {
    container.hidden = true;
    container.innerHTML = '';
    return;
  }

  container.innerHTML = '';
  state.matchedFilters.forEach((def, idx) => {
    const chip = doc.createElement('button');
    chip.className = 'filter-chip';
    chip.setAttribute('type', 'button');
    chip.setAttribute('data-filter-key', def.key);

    const tokenSpan = doc.createElement('span');
    tokenSpan.className = 'filter-chip-token';
    tokenSpan.textContent = `@${def.full}`;

    const descSpan = doc.createElement('span');
    descSpan.className = 'filter-chip-desc';
    descSpan.textContent = getFilterChipDescription(def.key);

    chip.appendChild(tokenSpan);
    chip.appendChild(descSpan);

    if (idx === 0) {
      const hintKbd = doc.createElement('kbd');
      hintKbd.className = 'filter-chip-hint';
      hintKbd.textContent = 'Tab';
      chip.appendChild(hintKbd);
    }

    chip.addEventListener('click', (e: Event) => {
      e.preventDefault();
      onSelect(def);
    });

    container.appendChild(chip);
  });

  container.hidden = false;
}

export function getI18nMessage(key: string, fallback: string): string {
  try {
    if (typeof chrome !== 'undefined' && chrome.i18n?.getMessage) {
      const msg = chrome.i18n.getMessage(key);
      if (msg) return msg;
    }
  } catch {
    // fallback
  }
  return fallback;
}

export function applyI18n(doc: Document): void {
  try {
    const textEls = doc.querySelectorAll?.('[data-i18n]');
    if (textEls && typeof textEls.forEach === 'function') {
      textEls.forEach((el) => {
        const key = el.getAttribute?.('data-i18n');
        if (key) {
          const fallback = el.textContent || '';
          el.textContent = getI18nMessage(key, fallback);
        }
      });
    }

    const placeholderEls = doc.querySelectorAll?.('[data-i18n-placeholder]');
    if (placeholderEls && typeof placeholderEls.forEach === 'function') {
      placeholderEls.forEach((el) => {
        const key = el.getAttribute?.('data-i18n-placeholder');
        if (key) {
          const inputEl = el as HTMLInputElement;
          const fallback = inputEl.placeholder || '';
          inputEl.placeholder = getI18nMessage(key, fallback);
        }
      });
    }
  } catch {
    // fallback gracefully
  }
}

export let allTabs: SearchTabItem[] = [];
export let currentResults: SearchTabItem[] = [];
export let selectedIndex = 0;
export let currentWindowId: number | undefined = undefined;

export async function loadTabsAndReconcileMRU(): Promise<SearchTabItem[]> {
  const [sessionData, rawTabs, currentWin] = await Promise.all([
    chrome.storage.session.get(MRU_STORAGE_KEY).catch(() => ({} as Record<string, any>)),
    chrome.tabs.query({}).catch(() => [] as chrome.tabs.Tab[]),
    chrome.windows?.getCurrent ? chrome.windows.getCurrent().catch(() => null) : Promise.resolve(null)
  ]);

  if (currentWin && typeof currentWin.id === 'number') {
    currentWindowId = currentWin.id;
  }

  const rawMru = (sessionData as Record<string, any>)[MRU_STORAGE_KEY];
  const mruIds: number[] = Array.isArray(rawMru) ? rawMru : [];
  const mruMap = new Map<number, number>();
  mruIds.forEach((id, idx) => mruMap.set(id, idx));

  if (mruMap.size === 0) {
    const sorted = [...rawTabs].sort((a, b) => (b.lastAccessed ?? 0) - (a.lastAccessed ?? 0));
    sorted.forEach((tab, idx) => {
      if (typeof tab.id === 'number') mruMap.set(tab.id, idx);
    });
  }

  const items: SearchTabItem[] = [];
  for (const tab of rawTabs) {
    const item = normalizeTab(tab, mruMap);
    if (item) items.push(item);
  }

  items.sort((a, b) => a.mruRank - b.mruRank);
  allTabs = items;
  currentResults = [...items];
  return items;
}

let isClosing = false;

export async function dismissWithAnimation(doc?: Document): Promise<void> {
  if (isClosing) return;
  isClosing = true;

  const currentDoc = doc ?? (typeof document !== 'undefined' ? document : undefined);
  const container = currentDoc?.querySelector?.('.search-container') as HTMLElement | null | undefined;
  if (container) {
    if (container.classList) {
      container.classList.add('is-closing');
    } else {
      container.className += ' is-closing';
    }
  }

  // 500ms delay allows the reverse waterfall exit animation to play smoothly
  const delayMs = (globalThis as Record<string, any>).process?.env?.NODE_TEST_CONTEXT ? 0 : 500;
  await new Promise(resolve => setTimeout(resolve, delayMs));

  if (typeof window !== 'undefined' && typeof window.close === 'function') {
    isClosing = false;
    window.close();
  }
}

export async function activateTab(tab: SearchTabItem, doc?: Document): Promise<void> {
  try {
    await chrome.tabs.update(tab.id, { active: true });
    await chrome.windows.update(tab.windowId, { focused: true });
  } catch (err) {
    console.error('Failed to activate tab:', err);
  } finally {
    await dismissWithAnimation(doc);
  }
}

export async function closeTab(tab: SearchTabItem, doc?: Document): Promise<void> {
  const currentDoc = doc ?? (typeof document !== 'undefined' ? document : undefined);

  try {
    if (typeof chrome !== 'undefined' && chrome.tabs?.remove) {
      await chrome.tabs.remove(tab.id);
    }
  } catch (err) {
    console.error('Failed to close tab:', err);
  }

  try {
    await removeTab(tab.id);
  } catch {
    // Graceful fallback if storage fails
  }

  allTabs = allTabs.filter(t => t.id !== tab.id);
  currentResults = currentResults.filter(t => t.id !== tab.id);

  if (selectedIndex >= currentResults.length) {
    selectedIndex = Math.max(0, currentResults.length - 1);
  }

  if (currentDoc) {
    renderTabList(currentDoc);
  }
}

export function updateSelection(newIndex: number, doc: Document = document): void {
  if (currentResults.length === 0) return;
  const listEl = doc.getElementById('tab-results');
  if (!listEl) return;

  const prevEl = listEl.children[selectedIndex] as HTMLElement | undefined;
  if (prevEl) prevEl.setAttribute('aria-selected', 'false');

  selectedIndex = (newIndex + currentResults.length) % currentResults.length;

  const nextEl = listEl.children[selectedIndex] as HTMLElement | undefined;
  if (nextEl) {
    nextEl.setAttribute('aria-selected', 'true');
    nextEl.scrollIntoView({ block: 'nearest' });
  }
}

export function renderTabList(doc: Document = document): void {
  const listEl = doc.getElementById('tab-results');
  const emptyEl = doc.getElementById('empty-state');
  if (!listEl) return;

  listEl.innerHTML = '';

  if (currentResults.length === 0) {
    if (emptyEl) emptyEl.hidden = false;
    return;
  }
  if (emptyEl) emptyEl.hidden = true;

  currentResults.forEach((tab, idx) => {
    const li = doc.createElement('li');
    li.className = 'tab-item';
    li.setAttribute('role', 'option');
    li.id = `tab-opt-${tab.id}`;
    li.setAttribute('aria-selected', idx === selectedIndex ? 'true' : 'false');
    li.addEventListener('click', () => { void activateTab(tab); });

    // Favicon
    if (tab.favIconUrl && !tab.favIconUrl.startsWith('chrome://')) {
      const img = doc.createElement('img');
      img.className = 'tab-favicon';
      img.src = tab.favIconUrl;
      img.alt = '';
      img.onerror = () => {
        img.replaceWith(createFaviconPlaceholder(doc, tab.title || tab.domain));
      };
      li.appendChild(img);
    } else {
      li.appendChild(createFaviconPlaceholder(doc, tab.title || tab.domain));
    }

    // Tab Info
    const info = doc.createElement('div');
    info.className = 'tab-info';

    const titleEl = doc.createElement('span');
    titleEl.className = 'tab-title';
    titleEl.textContent = tab.title;
    info.appendChild(titleEl);

    const meta = doc.createElement('div');
    meta.className = 'tab-meta';

    if (tab.domain) {
      const domainEl = doc.createElement('span');
      domainEl.className = 'tab-domain';
      domainEl.textContent = tab.domain;
      meta.appendChild(domainEl);
    }

    if (tab.active) {
      const activeBadge = doc.createElement('span');
      activeBadge.className = 'tab-badge tab-badge-active';
      activeBadge.textContent = getI18nMessage('badgeCurrent', 'Current');
      meta.appendChild(activeBadge);
    }

    if (tab.pinned) {
      const pinBadge = doc.createElement('span');
      pinBadge.className = 'tab-badge';
      pinBadge.textContent = getI18nMessage('badgePinned', 'Pinned');
      meta.appendChild(pinBadge);
    }

    if (tab.audible) {
      const audioBadge = doc.createElement('span');
      audioBadge.className = tab.muted ? 'tab-badge tab-badge-muted' : 'tab-badge tab-badge-audio';
      audioBadge.textContent = getI18nMessage(tab.muted ? 'badgeMuted' : 'badgeAudio', tab.muted ? 'Muted' : 'Audio');
      meta.appendChild(audioBadge);
    }

    info.appendChild(meta);
    li.appendChild(info);

    // Close button
    const closeBtn = doc.createElement('button');
    closeBtn.className = 'tab-close-btn';
    closeBtn.setAttribute('type', 'button');
    closeBtn.setAttribute('aria-label', 'Close tab');
    closeBtn.title = 'Close tab (Alt+D or Del)';
    closeBtn.innerHTML = '<svg viewBox="0 0 14 14" width="11" height="11" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><line x1="3" y1="3" x2="11" y2="11"></line><line x1="11" y1="3" x2="3" y2="11"></line></svg>';
    closeBtn.addEventListener('click', (e: Event) => {
      e.stopPropagation();
      void closeTab(tab, doc);
    });
    li.appendChild(closeBtn);

    listEl.appendChild(li);
  });

  // Ensure selected item is visible
  const selectedEl = listEl.children[selectedIndex] as HTMLElement | undefined;
  if (selectedEl) {
    selectedEl.scrollIntoView({ block: 'nearest' });
  }
}

function createFaviconPlaceholder(doc: Document, text: string): HTMLElement {
  const placeholder = doc.createElement('span');
  placeholder.className = 'tab-favicon-placeholder';
  placeholder.textContent = (text.trim()[0] || '?').toUpperCase();
  return placeholder;
}

export function updateSearchNotice(doc: Document, usedTypoFallback: boolean): void {
  const notice = doc.getElementById('search-notice');
  if (!notice) return;
  if (usedTypoFallback) {
    notice.textContent = getI18nMessage('typoNotice', 'Showing closest matches (typo tolerance)');
    notice.hidden = false;
  } else {
    notice.hidden = true;
  }
}

export async function initializeSearchSurface(doc: Document = document): Promise<void> {
  const input = doc.querySelector<HTMLInputElement>('[data-search-input]');
  if (!input) return;

  applyI18n(doc);

  allTabs = await loadTabsAndReconcileMRU();
  currentResults = [...allTabs];

  // Pre-select previous tab (index 1) for instant Alt+Tab toggle
  selectedIndex = currentResults.length > 1 ? 1 : 0;
  updateSearchNotice(doc, false);
  renderTabList(doc);

  input.addEventListener('input', () => {
    const query = input.value.trim();
    if (query.length === 0) {
      currentResults = [...allTabs];
      selectedIndex = currentResults.length > 1 ? 1 : 0;
      updateSearchNotice(doc, false);
    } else {
      const { results, usedTypoFallback } = searchTabsWithMeta(allTabs, query, currentWindowId);
      currentResults = results;
      selectedIndex = 0;
      updateSearchNotice(doc, usedTypoFallback);
    }
    renderTabList(doc);
  });

  const isCloseShortcut = (e: KeyboardEvent): boolean => {
    if (e.key === 'Escape') return true;
    if (e.altKey && (e.key === 'w' || e.key === 'W' || e.code === 'KeyW')) return true;
    if (e.metaKey && e.shiftKey && (e.key === 'k' || e.key === 'K' || e.code === 'KeyK')) return true;
    return false;
  };

  input.addEventListener('keydown', (e: KeyboardEvent) => {
    if (isCloseShortcut(e)) {
      e.preventDefault();
      void dismissWithAnimation(doc);
      return;
    }

    if (currentResults.length === 0) return;

    if ((e.altKey && (e.key === 'd' || e.key === 'D' || e.code === 'KeyD')) || (e.key === 'Delete' && (input as HTMLInputElement).value === '')) {
      e.preventDefault();
      const selected = currentResults[selectedIndex];
      if (selected) {
        void closeTab(selected, doc);
      }
      return;
    }

    if (e.key === 'Enter') {
      e.preventDefault();
      const selected = currentResults[selectedIndex];
      if (selected) {
        void activateTab(selected, doc);
      }
    } else if (e.key === 'ArrowDown' || (e.ctrlKey && e.key === 'n')) {
      e.preventDefault();
      updateSelection(selectedIndex + 1, doc);
    } else if (e.key === 'ArrowUp' || (e.ctrlKey && e.key === 'p')) {
      e.preventDefault();
      updateSelection(selectedIndex - 1, doc);
    } else if (e.key === 'Home') {
      e.preventDefault();
      updateSelection(0, doc);
    } else if (e.key === 'End') {
      e.preventDefault();
      updateSelection(currentResults.length - 1, doc);
    }
  });

  input.focus();
}

if (typeof document !== 'undefined') {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => { void initializeSearchSurface(); }, { once: true });
  } else {
    void initializeSearchSurface();
  }
}
