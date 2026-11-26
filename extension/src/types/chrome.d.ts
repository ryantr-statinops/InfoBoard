declare namespace chrome.tabs {
  export interface Tab {
    id?: number;
    index: number;
    windowId: number;
    highlighted: boolean;
    active: boolean;
    pinned: boolean;
    url?: string;
    title?: string;
    favIconUrl?: string;
    status?: string;
    incognito: boolean;
    width?: number;
    height?: number;
    sessionId?: string;
    groupId?: number;
    lastAccessed?: number;
    audible?: boolean;
    mutedInfo?: {
      muted: boolean;
      reason?: string;
      extensionId?: string;
    };
  }

  export interface ActiveInfo {
    tabId: number;
    windowId: number;
  }

  export interface QueryInfo {
    active?: boolean;
    pinned?: boolean;
    audible?: boolean;
    muted?: boolean;
    highlighted?: boolean;
    discarded?: boolean;
    autoDiscardable?: boolean;
    currentWindow?: boolean;
    lastFocusedWindow?: boolean;
    status?: string;
    title?: string;
    url?: string | string[];
    groupId?: number;
    windowId?: number;
    windowType?: string;
    index?: number;
  }

  export interface UpdateProperties {
    active?: boolean;
    highlighted?: boolean;
    pinned?: boolean;
    muted?: boolean;
    openerTabId?: number;
    url?: string;
    autoDiscardable?: boolean;
  }

  export interface TabActivatedEvent {
    addListener(callback: (activeInfo: ActiveInfo) => void): void;
    removeListener(callback: (activeInfo: ActiveInfo) => void): void;
  }

  export interface TabRemovedEvent {
    addListener(callback: (tabId: number, removeInfo: { windowId: number; isWindowClosing: boolean }) => void): void;
    removeListener(callback: (tabId: number, removeInfo: { windowId: number; isWindowClosing: boolean }) => void): void;
  }

  export const onActivated: TabActivatedEvent;
  export const onRemoved: TabRemovedEvent;

  export function query(queryInfo: QueryInfo): Promise<Tab[]>;
  export function query(queryInfo: QueryInfo, callback: (result: Tab[]) => void): void;
  export function get(tabId: number): Promise<Tab>;
  export function update(tabId: number, updateProperties: UpdateProperties): Promise<Tab>;
  export function remove(tabIds: number | number[]): Promise<void>;
}

declare namespace chrome.windows {
  export const WINDOW_ID_NONE: number;
  export const WINDOW_ID_CURRENT: number;

  export interface Window {
    id: number;
    focused: boolean;
    top?: number;
    left?: number;
    width?: number;
    height?: number;
    tabs?: chrome.tabs.Tab[];
    incognito: boolean;
    type?: string;
    state?: string;
    alwaysOnTop: boolean;
  }

  export interface UpdateInfo {
    left?: number;
    top?: number;
    width?: number;
    height?: number;
    focused?: boolean;
    drawAttention?: boolean;
    state?: string;
  }

  export interface WindowFocusChangedEvent {
    addListener(callback: (windowId: number) => void): void;
    removeListener(callback: (windowId: number) => void): void;
  }

  export const onFocusChanged: WindowFocusChangedEvent;
  export function update(windowId: number, updateInfo: UpdateInfo): Promise<Window>;
  export function getCurrent(): Promise<Window>;
}

declare namespace chrome.storage {
  export interface StorageArea {
    get(keys?: string | string[] | Record<string, any> | null): Promise<Record<string, any>>;
    set(items: Record<string, any>): Promise<void>;
    remove(keys: string | string[]): Promise<void>;
    clear(): Promise<void>;
  }

  export const session: StorageArea;
  export const local: StorageArea;
}

declare namespace chrome.runtime {
  export interface RuntimeEvent {
    addListener(callback: () => void): void;
    removeListener(callback: () => void): void;
  }

  export const onStartup: RuntimeEvent;
  export const onInstalled: RuntimeEvent;
}

declare namespace chrome.i18n {
  export function getMessage(messageName: string, substitutions?: any): string;
  export function getUILanguage(): string;
}
