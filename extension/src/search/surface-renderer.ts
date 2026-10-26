import type { QueryRenderRow } from '../runtime/query-client.js';
import { statusMessage, type SurfaceState } from './surface-state.js';

export interface SurfaceElements {
  input: HTMLInputElement;
  status: HTMLElement;
  list: HTMLElement;
  empty: HTMLElement;
  close: HTMLButtonElement | null;
}

export function findSurfaceElements(doc: Document): SurfaceElements | undefined {
  const input = doc.querySelector<HTMLInputElement>('[data-search-input]');
  const status = doc.querySelector<HTMLElement>('[data-surface-status]');
  const list = doc.querySelector<HTMLElement>('[data-result-list]');
  const empty = doc.querySelector<HTMLElement>('[data-empty-state]');
  if (!input || !status || !list || !empty) return undefined;
  return {
    input,
    status,
    list,
    empty,
    close: doc.querySelector<HTMLButtonElement>('[data-close-search]'),
  };
}

export function renderSurface(doc: Document, state: SurfaceState): void {
  const elements = findSurfaceElements(doc);
  if (!elements) return;
  const { input, status, list, empty } = elements;

  const root = doc.documentElement;
  root.dataset.surfaceState = state.status;
  root.dataset.searchReady = state.status === 'opening' ? 'false' : 'true';
  root.dataset.browserStateConfirmed = String(state.browserStateConfirmed);
  root.dataset.resultCount = String(state.rows.length);
  root.dataset.selectedIndex = String(state.selectedIndex);
  if (state.projectionRevision === null) delete root.dataset.projectionRevision;
  else root.dataset.projectionRevision = String(state.projectionRevision);
  if (state.persistenceState === undefined) delete root.dataset.persistenceState;
  else root.dataset.persistenceState = state.persistenceState;
  if (state.shortcutFocusMs === undefined) delete root.dataset.shortcutFocusMs;
  else root.dataset.shortcutFocusMs = String(state.shortcutFocusMs);

  input.setAttribute('aria-busy', String(state.status === 'querying'));
  input.setAttribute('aria-expanded', String(state.rows.length > 0));
  const selected = state.selectedIndex >= 0 && state.selectedIndex < state.rows.length
    ? resultElementId(state.selectedIndex)
    : '';
  if (selected === '') input.removeAttribute('aria-activedescendant');
  else input.setAttribute('aria-activedescendant', selected);

  list.replaceChildren();
  const fragment = doc.createDocumentFragment();
  state.rows.forEach((row, index) => fragment.append(renderRow(doc, row, index, state.selectedIndex === index)));
  list.append(fragment);
  list.hidden = state.rows.length === 0;
  list.setAttribute('aria-busy', String(state.status === 'querying'));

  empty.textContent = emptyMessage(state);
  empty.hidden = empty.textContent === '';
  status.textContent = statusMessage(state);
  status.dataset.surfaceStatus = state.status;
}

function renderRow(doc: Document, row: QueryRenderRow, index: number, selected: boolean): HTMLLIElement {
  const item = doc.createElement('li');
  item.id = resultElementId(index);
  item.className = 'result-row';
  item.setAttribute('role', 'option');
  item.setAttribute('aria-selected', String(selected));
  item.setAttribute('aria-label', rowLabel(row));
  item.dataset.resultId = row.resultId;
  item.dataset.selected = String(selected);

  const title = doc.createElement('span');
  title.className = 'result-title';
  title.dataset.resultTitle = '';
  title.textContent = row.title === '' ? 'Untitled tab' : row.title;
  title.title = title.textContent;

  const meta = doc.createElement('span');
  meta.className = 'result-context';
  meta.dataset.resultContext = '';
  const context = contextLabel(row);
  meta.textContent = context;
  meta.hidden = context === '';

  if (row.pinned) {
    const pinned = doc.createElement('span');
    pinned.className = 'result-pin';
    pinned.dataset.resultPinned = '';
    pinned.textContent = 'Pinned';
    item.append(title, pinned, meta);
  } else {
    item.append(title, meta);
  }

  const explanation = safeExplanation(row.explanations[0]);
  if (explanation !== '') {
    const detail = doc.createElement('span');
    detail.className = 'result-explanation';
    detail.dataset.resultExplanation = '';
    detail.textContent = explanation;
    item.append(detail);
  }

  return item;
}

function resultElementId(index: number): string {
  return `surface-result-${index}`;
}

function contextLabel(row: QueryRenderRow): string {
  const parts: string[] = [];
  const domain = safeDomain(row.domain);
  if (domain !== '') parts.push(domain);
  if (row.windowLabel !== undefined && row.windowLabel !== '') parts.push(row.windowLabel);
  if (row.groupLabel !== undefined && row.groupLabel !== '') parts.push(row.groupLabel);
  return parts.join(' · ');
}

function rowLabel(row: QueryRenderRow): string {
  const title = row.title === '' ? 'Untitled tab' : row.title;
  const context = contextLabel(row);
  const pinned = row.pinned ? 'Pinned' : '';
  const explanation = safeExplanation(row.explanations[0]);
  return [title, context, pinned, explanation].filter(value => value !== '').join('. ');
}

export function safeDomain(value: string): string {
  const text = value.trim();
  if (text === '') return '';
  try {
    const url = new URL(text.includes('://') ? text : `https://${text}`);
    return url.hostname.slice(0, 120);
  } catch {
    return text.split(/[?#]/, 1)[0].slice(0, 120);
  }
}

export function safeExplanation(value: string | undefined): string {
  if (value === undefined) return '';
  const text = value.trim();
  if (text === '' || text.length > 80) return '';
  if (!/^[a-z0-9 _-]+$/i.test(text)) return '';
  if (/https?|www\.|auth=|token=|[?#=&]/i.test(text)) return '';
  return text;
}

function emptyMessage(state: SurfaceState): string {
  if (state.status === 'ready') return 'Type to search open tabs.';
  if (state.status === 'no_results') return 'No matching open tabs.';
  if (state.status === 'empty_query' && state.rows.length === 0) return 'No recent tabs available.';
  if (state.status === 'private_unavailable') return 'Search is unavailable in private windows.';
  return '';
}
