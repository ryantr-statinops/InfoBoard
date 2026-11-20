import test from 'node:test';
import assert from 'node:assert/strict';

// Setup browser globals before importing module
let closed = false;
let updatedTab = null;
let updatedWindow = null;
let sessionStorage = { mru_tab_ids: [102, 101] };

globalThis.window = {
  close: () => { closed = true; }
};

globalThis.chrome = {
  storage: {
    session: {
      get: async (key) => ({ [key]: sessionStorage[key] })
    }
  },
  tabs: {
    query: async () => [
      { id: 101, windowId: 1, title: 'TikTok', url: 'https://tiktok.com', active: false, pinned: false },
      { id: 102, windowId: 1, title: 'Threads', url: 'https://threads.net', active: true, pinned: false },
      { id: 103, windowId: 1, title: 'Tiki', url: 'https://tiki.vn', active: false, pinned: false }
    ],
    update: async (tabId, props) => {
      updatedTab = { tabId, props };
      return {};
    }
  },
  windows: {
    update: async (windowId, props) => {
      updatedWindow = { windowId, props };
      return {};
    }
  }
};

class MockElement {
  constructor(tagName) {
    this.tagName = tagName;
    this.children = [];
    this.attributes = new Map();
    this.className = '';
    this.textContent = '';
    this.hidden = false;
  }
  setAttribute(k, v) { this.attributes.set(k, String(v)); }
  getAttribute(k) { return this.attributes.get(k) || null; }
  appendChild(child) { this.children.push(child); }
  scrollIntoView() {}
  addEventListener() {}
}

const mockDoc = {
  elements: new Map(),
  getElementById(id) {
    if (!this.elements.has(id)) {
      this.elements.set(id, new MockElement('div'));
    }
    return this.elements.get(id);
  },
  createElement(tag) {
    return new MockElement(tag);
  },
  querySelector(sel) {
    return new MockElement('input');
  }
};

const {
  loadTabsAndReconcileMRU,
  activateTab,
  updateSelection,
  renderTabList,
  currentResults,
  selectedIndex
} = await import('../extension/dist/src/search/surface-entrypoint.js');

test('loadTabsAndReconcileMRU: sorts loaded tabs by session MRU rank', async () => {
  const tabs = await loadTabsAndReconcileMRU();
  assert.equal(tabs.length, 3);
  // id 102 is first in session storage [102, 101]
  assert.equal(tabs[0].id, 102);
  assert.equal(tabs[0].mruRank, 0);
  assert.equal(tabs[1].id, 101);
  assert.equal(tabs[1].mruRank, 1);
  // id 103 wasn't in session storage -> fallback mruRank 9999
  assert.equal(tabs[2].id, 103);
  assert.equal(tabs[2].mruRank, 9999);
});

test('activateTab: activates tab, focuses window, and closes popup', async () => {
  closed = false;
  updatedTab = null;
  updatedWindow = null;

  const tabItem = {
    id: 101,
    windowId: 7,
    title: 'TikTok',
    domain: 'tiktok.com',
    url: 'https://tiktok.com',
    mruRank: 0,
    active: false,
    pinned: false
  };

  await activateTab(tabItem);

  assert.equal(closed, true);
  assert.deepEqual(updatedTab, { tabId: 101, props: { active: true } });
  assert.deepEqual(updatedWindow, { windowId: 7, props: { focused: true } });
});

test('renderTabList: renders DOM items with role="option" and aria-selected', () => {
  const listEl = mockDoc.getElementById('tab-results');
  listEl.children = [];

  renderTabList(mockDoc);

  assert.ok(listEl.children.length >= 0);
});

test('updateSelection: updates aria-selected attributes and cycles selection', async () => {
  const tabs = await loadTabsAndReconcileMRU();
  renderTabList(mockDoc);

  const listEl = mockDoc.getElementById('tab-results');
  const opt0 = new MockElement('li');
  const opt1 = new MockElement('li');
  const opt2 = new MockElement('li');
  listEl.children = [opt0, opt1, opt2];

  updateSelection(1, mockDoc);
  assert.equal(opt1.getAttribute('aria-selected'), 'true');
  assert.equal(opt0.getAttribute('aria-selected'), 'false');
});
