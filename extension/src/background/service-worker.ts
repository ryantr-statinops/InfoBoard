import { ChromeBrowserAdapter, type BrowserAdapter } from '../browser/browser-adapter.js';
import { createCommandHandler } from './open-search-command.js';

const adapter: BrowserAdapter = new ChromeBrowserAdapter();
const handleCommand = createCommandHandler(adapter);
adapter.registerCommandListener(command => { void handleCommand(command); });
adapter.registerLifecycleListeners(() => undefined, () => undefined);
adapter.registerMessageListener((message, _tabId, windowId) => {
  if (typeof message === 'object' && message !== null && (message as { type?: unknown }).type === 'dismiss-search' && Number.isSafeInteger(windowId)) {
    void adapter.closeSearchSurface(windowId!);
  }
});
