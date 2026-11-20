export const MRU_STORAGE_KEY = 'mru_tab_ids';
export const MAX_MRU_SIZE = 100;

export async function getMRUTabIds(): Promise<number[]> {
  try {
    const result = await chrome.storage.session.get(MRU_STORAGE_KEY);
    const ids = result[MRU_STORAGE_KEY];
    return Array.isArray(ids) ? ids : [];
  } catch {
    return [];
  }
}

export async function setMRUTabIds(ids: number[]): Promise<void> {
  try {
    await chrome.storage.session.set({
      [MRU_STORAGE_KEY]: ids.slice(0, MAX_MRU_SIZE)
    });
  } catch (err) {
    console.error('Failed to save MRU tab IDs:', err);
  }
}

export async function touchTab(tabId: number): Promise<void> {
  if (typeof tabId !== 'number' || !Number.isInteger(tabId) || tabId < 0) return;
  const current = await getMRUTabIds();
  const next = [tabId, ...current.filter(id => id !== tabId)].slice(0, MAX_MRU_SIZE);
  await setMRUTabIds(next);
}

export async function removeTab(tabId: number): Promise<void> {
  if (typeof tabId !== 'number' || !Number.isInteger(tabId) || tabId < 0) return;
  const current = await getMRUTabIds();
  const next = current.filter(id => id !== tabId);
  await setMRUTabIds(next);
}

export async function hydrateFromTabs(tabs: chrome.tabs.Tab[]): Promise<void> {
  const current = await getMRUTabIds();
  if (current.length > 0) return;

  const validTabs = tabs
    .filter(t => typeof t.id === 'number')
    .sort((a, b) => (b.lastAccessed ?? 0) - (a.lastAccessed ?? 0));

  const initialIds = validTabs.map(t => t.id as number).slice(0, MAX_MRU_SIZE);
  await setMRUTabIds(initialIds);
}
