import { isUnknownRecord } from './type-guards.js';
import type { BrowserApiEvent, BrowserGroupSnapshot, BrowserTabSnapshot, BrowserTabSnapshotBatch, BrowserWindowSnapshot } from './event-normalizer.js';
import { createProfileID, parseProfileID, type ContextKind, type ProfileID, type ProfileIDStore } from '../../domain/index.js';

export type BrowserResult<T> =
  | { ok: true; value: T }
  | { ok: false; error: { kind: 'permission_denied' | 'unsupported' | 'not_found' | 'operation_failed' | 'cancelled'; retryable: boolean } };
export interface BrowserProfileContext { browserFamily: 'chrome' | 'edge'; profileId: ProfileID; contextKind: ContextKind }
export interface BrowserObserverRegistration { unsubscribe(): void; missingOptionalCapabilities: number }
export interface BrowserAdapter {
  openSearchSurface(): Promise<BrowserResult<void>>;
  closeSearchSurface(): Promise<BrowserResult<void>>;
  currentProfileContext(): Promise<BrowserResult<BrowserProfileContext>>;
  privateContextAccess(): Promise<BrowserResult<boolean>>;
  getOpenTabSnapshot(): Promise<BrowserResult<BrowserTabSnapshotBatch>>;
  registerTabObservationListeners(listener: (event: BrowserApiEvent) => void): BrowserResult<BrowserObserverRegistration>;
  registerCommandListener(listener: (command: string) => void): void;
  registerLifecycleListeners(onInstall: (reason: string) => void, onStartup: () => void): void;
  registerPopupCloseListener(listener: () => void): void;
}
type BrowserApiListener = (...args: readonly unknown[]) => void;
interface BrowserApiListenerEvent { addListener(listener: BrowserApiListener): void; removeListener?(listener: BrowserApiListener): void }
interface BrowserApiTabs {
  query(queryInfo: Record<string, never>): Promise<unknown[]>;
  onCreated?: BrowserApiListenerEvent;
  onUpdated?: BrowserApiListenerEvent;
  onMoved?: BrowserApiListenerEvent;
  onAttached?: BrowserApiListenerEvent;
  onDetached?: BrowserApiListenerEvent;
  onActivated?: BrowserApiListenerEvent;
  onRemoved?: BrowserApiListenerEvent;
}
interface BrowserApiTabGroups {
  query(queryInfo: Record<string, never>): Promise<unknown[]>;
  onUpdated?: BrowserApiListenerEvent;
  onMoved?: BrowserApiListenerEvent;
  onRemoved?: BrowserApiListenerEvent;
}
interface BrowserApiWindows {
  getLastFocused(): Promise<{ id?: number; incognito?: boolean }>;
  getAll?(options?: { populate?: boolean }): Promise<unknown[]>;
  onCreated?: BrowserApiListenerEvent;
  onRemoved?: BrowserApiListenerEvent;
  onFocusChanged?: BrowserApiListenerEvent;
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
  windows: BrowserApiWindows;
  tabs?: BrowserApiTabs;
  tabGroups?: BrowserApiTabGroups;
  extension?: { isAllowedIncognitoAccess(): Promise<boolean> };
  storage: { local: { get(key: string): Promise<Record<string, unknown>>; set(items: Record<string, unknown>): Promise<void>; remove(key: string): Promise<void> } };
}
const PROFILE_KEY = 'profile_id';
type ErrorKind = 'permission_denied' | 'unsupported' | 'not_found' | 'operation_failed' | 'cancelled';
function failure(error: unknown): BrowserResult<never> {
  const message = error instanceof Error ? error.message.toLowerCase() : '';
  const kind: ErrorKind = message.includes('permission') ? 'permission_denied' : message.includes('not found') ? 'not_found' : 'operation_failed';
  return { ok: false, error: { kind, retryable: kind === 'operation_failed' } };
}
function tabSnapshotFromApi(value: unknown): BrowserTabSnapshot {
  const result: BrowserTabSnapshot = {};
  if (!isUnknownRecord(value)) return result;
  for (const field of ['id', 'windowId', 'groupId', 'title', 'url', 'pinned', 'active', 'incognito'] as const) {
    if (Object.hasOwn(value, field)) result[field] = value[field];
  }
  return result;
}

function windowSnapshotFromApi(value: unknown): BrowserWindowSnapshot {
  const result: BrowserWindowSnapshot = {};
  if (!isUnknownRecord(value)) return result;
  for (const field of ['id', 'incognito', 'focused', 'title'] as const) {
    if (Object.hasOwn(value, field)) result[field] = value[field];
  }
  return result;
}

function groupSnapshotFromApi(value: unknown): BrowserGroupSnapshot {
  const result: BrowserGroupSnapshot = {};
  if (!isUnknownRecord(value)) return result;
  for (const field of ['id', 'windowId', 'title'] as const) {
    if (Object.hasOwn(value, field)) result[field] = value[field];
  }
  return result;
}

function attachObserverListener(
  source: BrowserApiEvent['source'],
  event: BrowserApiListenerEvent | undefined,
  listener: (event: BrowserApiEvent) => void,
  removers: Array<() => void>,
): boolean {
  if (!event) return false;
  const callback: BrowserApiListener = (...args) => listener({ source, args });
  event.addListener(callback);
  removers.push(() => event.removeListener?.(callback));
  return true;
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
      if (reason === 'install') {
        void this.initializeFirstInstall().then(() => onInstall(reason)).catch(() => undefined);
      } else {
        onInstall(reason);
      }
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
  async getOpenTabSnapshot(): Promise<BrowserResult<BrowserTabSnapshotBatch>> {
    const tabs = this.api.tabs;
    const getAllWindows = this.api.windows.getAll;
    if (!tabs || !getAllWindows) return { ok: false, error: { kind: 'unsupported', retryable: false } };
    try {
      const [rawTabs, rawWindows] = await Promise.all([tabs.query({}), getAllWindows({ populate: false })]);
      let rawGroups: unknown[] = [];
      let optionalCapabilityErrors = 0;
      const groupApi = this.api.tabGroups;
      if (groupApi) {
        try { rawGroups = await groupApi.query({}); }
        catch { optionalCapabilityErrors += 1; }
      } else {
        optionalCapabilityErrors += 1;
      }
      return { ok: true, value: {
        tabs: rawTabs.map(tabSnapshotFromApi),
        windows: rawWindows.map(windowSnapshotFromApi),
        groups: rawGroups.map(groupSnapshotFromApi),
        optionalCapabilityErrors,
      } };
    } catch (error) { return failure(error); }
  }
  registerTabObservationListeners(listener: (event: BrowserApiEvent) => void): BrowserResult<BrowserObserverRegistration> {
    const tabs = this.api.tabs;
    const windows = this.api.windows;
    if (!tabs || !windows) return { ok: false, error: { kind: 'unsupported', retryable: false } };
    const removers: Array<() => void> = [];
    const requiredSources: Array<[BrowserApiEvent['source'], BrowserApiListenerEvent | undefined]> = [
      ['tabs.onCreated', tabs.onCreated], ['tabs.onUpdated', tabs.onUpdated], ['tabs.onMoved', tabs.onMoved],
      ['tabs.onAttached', tabs.onAttached], ['tabs.onDetached', tabs.onDetached], ['tabs.onActivated', tabs.onActivated],
      ['tabs.onRemoved', tabs.onRemoved], ['windows.onCreated', windows.onCreated], ['windows.onRemoved', windows.onRemoved],
      ['windows.onFocusChanged', windows.onFocusChanged],
    ];
    for (const [source, event] of requiredSources) {
      if (!attachObserverListener(source, event, listener, removers)) {
        for (const remove of removers) { try { remove(); } catch { /* best-effort listener cleanup */ } }
        return { ok: false, error: { kind: 'unsupported', retryable: false } };
      }
    }
    let missingOptionalCapabilities = 0;
    const groups = this.api.tabGroups;
    if (!groups) {
      missingOptionalCapabilities = 1;
    } else {
      const optionalSources: Array<[BrowserApiEvent['source'], BrowserApiListenerEvent | undefined]> = [
        ['tabGroups.onUpdated', groups.onUpdated], ['tabGroups.onMoved', groups.onMoved], ['tabGroups.onRemoved', groups.onRemoved],
      ];
      for (const [source, event] of optionalSources) {
        if (!attachObserverListener(source, event, listener, removers)) missingOptionalCapabilities += 1;
      }
    }
    return { ok: true, value: {
      missingOptionalCapabilities,
      unsubscribe: () => { for (const remove of removers) { try { remove(); } catch { /* best-effort listener cleanup */ } } },
    } };
  }
  async privateContextAccess(): Promise<BrowserResult<boolean>> {
    const isAllowedIncognitoAccess = this.api.extension?.isAllowedIncognitoAccess;
    if (!isAllowedIncognitoAccess) return { ok: false, error: { kind: 'unsupported', retryable: false } };
    try { return { ok: true, value: await isAllowedIncognitoAccess.call(this.api.extension) }; }
    catch (error) { return failure(error); }
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
