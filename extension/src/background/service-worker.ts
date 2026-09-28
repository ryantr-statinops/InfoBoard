import { ChromeBrowserAdapter, type BrowserAdapter } from '../browser/browser-adapter.js';
import { TabObserver } from '../browser/tab-observer.js';
import { createCommandHandler } from './open-search-command.js';

const adapter: BrowserAdapter = new ChromeBrowserAdapter();
const handleCommand = createCommandHandler(adapter);
adapter.registerCommandListener(command => { void handleCommand(command); });
const tabObserver = new TabObserver(adapter);
adapter.registerLifecycleListeners(() => { void tabObserver.start(); }, () => { void tabObserver.start(); });
void tabObserver.start();
export { tabObserver };
