import test from 'node:test';
import assert from 'node:assert/strict';

// Setup mock chrome.storage.session before importing module
let sessionStorage = {};

globalThis.chrome = {
  storage: {
    session: {
      get: async (key) => {
        if (typeof key === 'string') {
          return { [key]: sessionStorage[key] };
        }
        return { ...sessionStorage };
      },
      set: async (items) => {
        Object.assign(sessionStorage, items);
      }
    }
  }
};

const {
  MRU_STORAGE_KEY,
  MAX_MRU_SIZE,
  getMRUTabIds,
  setMRUTabIds,
  touchTab,
  removeTab,
  hydrateFromTabs
} = await import('../extension/dist/src/background/mru-manager.js');

test('setup: clean sessionStorage before each test group', () => {
  sessionStorage = {};
});

test('touchTab: adds new tab to top', async () => {
  sessionStorage = {};
  await touchTab(101);
  const ids = await getMRUTabIds();
  assert.deepEqual(ids, [101]);

  await touchTab(102);
  const updated = await getMRUTabIds();
  assert.deepEqual(updated, [102, 101]);
});

test('touchTab: moves existing tab to top without duplicates', async () => {
  sessionStorage = {};
  await setMRUTabIds([1, 2, 3, 4]);

  await touchTab(3);
  const ids = await getMRUTabIds();
  assert.deepEqual(ids, [3, 1, 2, 4]);
});

test('touchTab: caps ring buffer at MAX_MRU_SIZE (100)', async () => {
  sessionStorage = {};
  const initial = Array.from({ length: 100 }, (_, i) => i + 1);
  await setMRUTabIds(initial);

  await touchTab(999);
  const ids = await getMRUTabIds();
  assert.equal(ids.length, MAX_MRU_SIZE);
  assert.equal(ids[0], 999);
  assert.equal(ids.includes(100), false); // Last item was evicted
});

test('touchTab: ignores invalid tab IDs', async () => {
  sessionStorage = {};
  await setMRUTabIds([1, 2]);

  await touchTab(-1);
  await touchTab(NaN);
  // @ts-ignore
  await touchTab('invalid');

  const ids = await getMRUTabIds();
  assert.deepEqual(ids, [1, 2]);
});

test('removeTab: removes existing tab from MRU list', async () => {
  sessionStorage = {};
  await setMRUTabIds([10, 20, 30, 40]);

  await removeTab(20);
  const ids = await getMRUTabIds();
  assert.deepEqual(ids, [10, 30, 40]);
});

test('removeTab: gracefully handles non-existent tab', async () => {
  sessionStorage = {};
  await setMRUTabIds([10, 20]);

  await removeTab(999);
  const ids = await getMRUTabIds();
  assert.deepEqual(ids, [10, 20]);
});

test('hydrateFromTabs: populates empty session storage sorted by lastAccessed desc', async () => {
  sessionStorage = {};
  const mockTabs = [
    { id: 1, lastAccessed: 1000 },
    { id: 2, lastAccessed: 5000 },
    { id: 3, lastAccessed: 3000 },
    { id: 4, lastAccessed: undefined }
  ];

  await hydrateFromTabs(mockTabs);
  const ids = await getMRUTabIds();
  assert.deepEqual(ids, [2, 3, 1, 4]);
});

test('hydrateFromTabs: does not overwrite existing MRU session state', async () => {
  sessionStorage = { [MRU_STORAGE_KEY]: [99, 88] };
  const mockTabs = [
    { id: 1, lastAccessed: 5000 }
  ];

  await hydrateFromTabs(mockTabs);
  const ids = await getMRUTabIds();
  assert.deepEqual(ids, [99, 88]);
});
