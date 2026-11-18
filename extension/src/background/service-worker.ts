import { ChromeBrowserAdapter, type BrowserAdapter } from '../browser/browser-adapter.js';
import { TabObserver } from '../browser/tab-observer.js';
import { createCommandHandler } from './open-search-command.js';
import { touchTab, removeTab, hydrateFromTabs } from './mru-manager.js';

const adapter: BrowserAdapter = new ChromeBrowserAdapter();
const handleCommand = createCommandHandler(adapter);
adapter.registerCommandListener(command => { void handleCommand(command); });

const tabObserver = new TabObserver(adapter);
adapter.registerLifecycleListeners(() => { void tabObserver.start(); }, () => { void tabObserver.start(); });
void tabObserver.start();

// Track tab activation and window focus for MRU order
chrome.tabs.onActivated.addListener(activeInfo => {
  if (typeof activeInfo.tabId === 'number') {
    void touchTab(activeInfo.tabId);
  }
});

chrome.tabs.onRemoved.addListener(tabId => {
  if (typeof tabId === 'number') {
    void removeTab(tabId);
  }
});

chrome.windows.onFocusChanged.addListener(windowId => {
  if (windowId === chrome.windows.WINDOW_ID_NONE) return;
  chrome.tabs.query({ active: true, windowId }, ([activeTab]) => {
    if (activeTab && typeof activeTab.id === 'number') {
      void touchTab(activeTab.id);
    }
  });
});

// Initialize MRU order from open tabs if session storage is empty
const initializeMRU = () => {
  chrome.tabs.query({}, tabs => {
    void hydrateFromTabs(tabs);
  });
};

chrome.runtime.onStartup.addListener(initializeMRU);
chrome.runtime.onInstalled.addListener(initializeMRU);
initializeMRU();

export { tabObserver };
