import { touchTab, removeTab, hydrateFromTabs } from './mru-manager.js';

// Track tab activation for MRU order
chrome.tabs.onActivated.addListener(activeInfo => {
  if (typeof activeInfo.tabId === 'number') {
    void touchTab(activeInfo.tabId);
  }
});

// Remove closed tabs from MRU stack
chrome.tabs.onRemoved.addListener(tabId => {
  if (typeof tabId === 'number') {
    void removeTab(tabId);
  }
});

// Track window focus changes to update active tab MRU
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
