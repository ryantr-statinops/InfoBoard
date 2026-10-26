import { ChromeBrowserAdapter, type BrowserAdapter } from '../browser/browser-adapter.js';
import { TabObserver } from '../browser/tab-observer.js';
import { createCommandHandler } from './open-search-command.js';

interface SessionStorageApi {
  storage?: {
    session?: {
      set(items: Record<string, unknown>): Promise<void>;
    };
  };
}

const adapter: BrowserAdapter = new ChromeBrowserAdapter();
const chromeApi = globalThis as unknown as SessionStorageApi;
const handleCommand = createCommandHandler(adapter, {
  recordCommandStart: async startedAt => {
    await chromeApi.storage?.session?.set({ infoboard_search_command_started_at: startedAt });
  },
});
adapter.registerCommandListener(command => { void handleCommand(command); });
const tabObserver = new TabObserver(adapter);
adapter.registerLifecycleListeners(() => { void tabObserver.start(); }, () => { void tabObserver.start(); });
void tabObserver.start();
export { tabObserver };
