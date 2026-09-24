import { createProfileID, parseProfileID, type ContextKind, type ProfileID, type ProfileIDStore } from '../../domain/index.js';

export type BrowserResult<T> =
  | { ok: true; value: T }
  | { ok: false; error: { kind: 'permission_denied' | 'unsupported' | 'not_found' | 'operation_failed' | 'cancelled'; retryable: boolean } };
export interface BrowserProfileContext { browserFamily: 'chrome' | 'edge'; profileId: ProfileID; contextKind: ContextKind }
export interface BrowserAdapter {
  openSearchSurface(): Promise<BrowserResult<void>>;
  closeSearchSurface(): Promise<BrowserResult<void>>;
  currentProfileContext(): Promise<BrowserResult<BrowserProfileContext>>;
  registerCommandListener(listener: (command: string) => void): void;
  registerLifecycleListeners(onInstall: (reason: string) => void, onStartup: () => void): void;
  registerPopupCloseListener(listener: () => void): void;
}
export interface BrowserApi {
  runtime: {
    onInstalled: { addListener(fn: (details: { reason: string }) => void): void };
    onStartup: { addListener(fn: () => void): void };
    onMessage: { addListener(fn: (message: unknown, sender: unknown, sendResponse: (response: unknown) => void) => boolean | void): void };
    sendMessage(message: unknown): Promise<unknown>;
  };
  commands: { onCommand: { addListener(fn: (command: string) => void): void } };
  action: { openPopup(options?: { windowId?: number }): Promise<void> };
  windows: { getLastFocused(): Promise<{ id?: number; incognito?: boolean }> };
  storage: { local: { get(key: string): Promise<Record<string, unknown>>; set(items: Record<string, unknown>): Promise<void>; remove(key: string): Promise<void> } };
}
const PROFILE_KEY = 'profile_id';
type ErrorKind = 'permission_denied' | 'unsupported' | 'not_found' | 'operation_failed' | 'cancelled';
function failure(error: unknown): BrowserResult<never> {
  const message = error instanceof Error ? error.message.toLowerCase() : '';
  const kind: ErrorKind = message.includes('permission') ? 'permission_denied' : message.includes('not found') ? 'not_found' : 'operation_failed';
  return { ok: false, error: { kind, retryable: kind === 'operation_failed' } };
}

export class ChromeBrowserAdapter implements BrowserAdapter, ProfileIDStore {
  private readonly api: BrowserApi;
  constructor(api?: BrowserApi) { this.api = api ?? (globalThis as typeof globalThis & { chrome: BrowserApi }).chrome; }
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
  registerPopupCloseListener(listener: () => void): void {
    this.api.runtime.onMessage.addListener((message, _sender, sendResponse) => {
      if (typeof message !== 'object' || message === null || !('type' in message) || message.type !== 'dismiss-search') return false;
      listener();
      sendResponse({ ok: true });
      return false;
    });
  }
  async currentProfileContext(): Promise<BrowserResult<BrowserProfileContext>> {
    try {
      const profileId = parseProfileID(await this.read());
      const focused = await this.api.windows.getLastFocused();
      if (!Number.isSafeInteger(focused.id)) return { ok: false, error: { kind: 'not_found', retryable: true } };
      const ua = globalThis.navigator?.userAgent ?? '';
      return { ok: true, value: { browserFamily: /Edg\//.test(ua) ? 'edge' : 'chrome', profileId, contextKind: focused.incognito ? 'private' : 'normal' } };
    } catch (error) { return failure(error); }
  }
  async openSearchSurface(): Promise<BrowserResult<void>> {
    try {
      await this.api.action.openPopup();
      return { ok: true, value: undefined };
    } catch (error) { return failure(error); }
  }
  async closeSearchSurface(): Promise<BrowserResult<void>> {
    try {
      await this.api.runtime.sendMessage({ type: 'dismiss-search' });
      return { ok: true, value: undefined };
    } catch (error) { return failure(error); }
  }
}

export async function resetProfileAfterDisconnect(store: ProfileIDStore, disconnect: () => Promise<void>): Promise<void> {
  await disconnect();
  await store.removeAfterDisconnect();
}
