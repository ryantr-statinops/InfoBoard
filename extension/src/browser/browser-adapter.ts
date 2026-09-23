import { createProfileID, parseProfileID, type ContextKind, type ProfileID, type ProfileIDStore } from '../../domain/index.js';

export type BrowserResult<T> =
  | { ok: true; value: T }
  | { ok: false; error: { kind: 'permission_denied' | 'unsupported' | 'not_found' | 'operation_failed' | 'cancelled'; retryable: boolean } };
export interface BrowserProfileContext { browserFamily: 'chrome' | 'edge'; profileId: ProfileID; contextKind: ContextKind }
export interface SurfaceBounds { width: 480; height: 600 }
export interface SearchSurfaceRef { windowId: number; tabId: number }
export interface BrowserAdapter {
  extensionUrl(path: string): string;
  listSearchSurfaces(): Promise<BrowserResult<SearchSurfaceRef[]>>;
  createSearchSurface(url: string, bounds: SurfaceBounds): Promise<BrowserResult<SearchSurfaceRef>>;
  focusSearchSurface(windowId: number, tabId: number): Promise<BrowserResult<SearchSurfaceRef>>;
  closeSearchSurface(windowId: number): Promise<BrowserResult<void>>;
  currentProfileContext(): Promise<BrowserResult<BrowserProfileContext>>;
  registerCommandListener(listener: (command: string) => void): void;
  registerLifecycleListeners(onInstall: (reason: string) => void, onStartup: () => void): void;
  registerMessageListener(listener: (message: unknown, senderTabId: number | undefined, senderWindowId: number | undefined) => void): void;
}
interface Native {
  runtime: { getURL(path: string): string; onInstalled: { addListener(fn: (details: { reason: string }) => void): void }; onStartup: { addListener(fn: () => void): void }; onMessage: { addListener(fn: (message: unknown, sender: { tab?: { id?: number; windowId?: number } }) => boolean | void): void } };
  commands: { onCommand: { addListener(fn: (command: string) => void): void } };
  tabs: { query(q: Record<string, unknown>): Promise<Array<{ id?: number; windowId?: number; url?: string; incognito?: boolean }>>; update(id: number, info: Record<string, unknown>): Promise<unknown> };
  windows: { getLastFocused(): Promise<{ id?: number; incognito?: boolean }>; create(info: Record<string, unknown>): Promise<{ id?: number }>; update(id: number, info: Record<string, unknown>): Promise<unknown>; remove(id: number): Promise<void> };
  storage: { local: { get(key: string): Promise<Record<string, unknown>>; set(items: Record<string, unknown>): Promise<void>; remove(key: string): Promise<void> } };
}
const PROFILE_KEY = 'profile_id';
const SURFACE_PATH = 'dist/search/surface-shell.html';
type ErrorKind = 'permission_denied' | 'unsupported' | 'not_found' | 'operation_failed' | 'cancelled';
function failure(error: unknown): BrowserResult<never> {
  const message = error instanceof Error ? error.message.toLowerCase() : '';
  const kind: ErrorKind = message.includes('permission') ? 'permission_denied' : message.includes('not found') ? 'not_found' : 'operation_failed';
  return { ok: false, error: { kind, retryable: kind === 'operation_failed' } };
}

export class ChromeBrowserAdapter implements BrowserAdapter, ProfileIDStore {
  private readonly api: Native;
  private contextKind: ContextKind = 'normal';
  constructor(api?: Native) { this.api = api ?? (globalThis as typeof globalThis & { chrome: Native }).chrome; }
  extensionUrl(path: string): string { return this.api.runtime.getURL(path); }
  async read(): Promise<unknown | null> {
    const values = await this.api.storage.local.get(PROFILE_KEY);
    return Object.hasOwn(values, PROFILE_KEY) ? values[PROFILE_KEY] : null;
  }
  async write(id: ProfileID): Promise<void> { await this.api.storage.local.set({ [PROFILE_KEY]: id }); }
  async removeAfterDisconnect(): Promise<void> { await this.api.storage.local.remove(PROFILE_KEY); }
  async initializeFirstInstall(): Promise<void> {
    if (await this.read() !== null) return;
    await this.write(createProfileID());
  }
  registerCommandListener(listener: (command: string) => void): void { this.api.commands.onCommand.addListener(listener); }
  registerLifecycleListeners(onInstall: (reason: string) => void, onStartup: () => void): void {
    this.api.runtime.onInstalled.addListener(({ reason }) => {
      onInstall(reason);
      if (reason === 'install') void this.initializeFirstInstall().catch(() => undefined);
    });
    this.api.runtime.onStartup.addListener(onStartup);
  }
  registerMessageListener(listener: (message: unknown, senderTabId: number | undefined, senderWindowId: number | undefined) => void): void {
    this.api.runtime.onMessage.addListener((message, sender) => {
      listener(message, sender.tab?.id, sender.tab?.windowId);
      return false;
    });
  }
  async currentProfileContext(): Promise<BrowserResult<BrowserProfileContext>> {
    try {
      const profileId = parseProfileID(await this.read());
      const focused = await this.api.windows.getLastFocused();
      if (!Number.isSafeInteger(focused.id)) return { ok: false, error: { kind: 'not_found', retryable: true } };
      this.contextKind = focused.incognito ? 'private' : 'normal';
      const ua = globalThis.navigator?.userAgent ?? '';
      return { ok: true, value: { browserFamily: /Edg\//.test(ua) ? 'edge' : 'chrome', profileId, contextKind: this.contextKind } };
    } catch (error) { return failure(error); }
  }
  async listSearchSurfaces(): Promise<BrowserResult<SearchSurfaceRef[]>> {
    try {
      const base = this.extensionUrl(SURFACE_PATH);
      const tabs = await this.api.tabs.query({ url: [base + '*'] });
      const surfaces = tabs.filter(tab => tab.url?.startsWith(base) && (tab.incognito ? 'private' : 'normal') === this.contextKind && Number.isSafeInteger(tab.id) && Number.isSafeInteger(tab.windowId));
      return { ok: true, value: surfaces.map(tab => ({ tabId: tab.id!, windowId: tab.windowId! })) };
    } catch (error) { return failure(error); }
  }
  async createSearchSurface(url: string, bounds: SurfaceBounds): Promise<BrowserResult<SearchSurfaceRef>> {
    if (url !== this.extensionUrl(SURFACE_PATH) || bounds.width !== 480 || bounds.height !== 600) return { ok: false, error: { kind: 'unsupported', retryable: false } };
    try {
      const window = await this.api.windows.create({ url, type: 'popup', focused: true, width: 480, height: 600, incognito: this.contextKind === 'private' });
      if (!Number.isSafeInteger(window.id)) return { ok: false, error: { kind: 'operation_failed', retryable: true } };
      const tabs = await this.api.tabs.query({ windowId: window.id });
      const tab = tabs.find(item => item.url === url && Number.isSafeInteger(item.id));
      return tab ? { ok: true, value: { windowId: window.id!, tabId: tab.id! } } : { ok: false, error: { kind: 'not_found', retryable: true } };
    } catch (error) { return failure(error); }
  }
  async focusSearchSurface(windowId: number, tabId: number): Promise<BrowserResult<SearchSurfaceRef>> {
    try { await this.api.tabs.update(tabId, { active: true }); await this.api.windows.update(windowId, { focused: true }); return { ok: true, value: { windowId, tabId } }; }
    catch (error) { return failure(error); }
  }
  async closeSearchSurface(windowId: number): Promise<BrowserResult<void>> {
    try { await this.api.windows.remove(windowId); return { ok: true, value: undefined }; }
    catch (error) { return failure(error); }
  }
}
export async function resetProfileAfterDisconnect(store: ProfileIDStore, disconnect: () => Promise<void>): Promise<void> {
  await disconnect();
  await store.removeAfterDisconnect();
}
