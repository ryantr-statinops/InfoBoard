import type { BrowserProfileContext } from '../browser/browser-adapter.js';
import { NativeMessagingClient } from '../runtime/native-messaging-client.js';
import { QueryClient } from '../runtime/query-client.js';

export interface SearchRuntime {
  source: QueryClient;
  close(): void;
}

const HOST_NAME = 'com.infoboard.host';
const EXTENSION_VERSION = '0.1.0';

// The popup owns its short-lived query client. IP-14 will add activation
// confirmation; this phase only opens the IP-12 query contract and keeps the
// keyboard focus path independent of host readiness.
export function createNativeSearchQuerySource(context: BrowserProfileContext): SearchRuntime {
  const client = new NativeMessagingClient({
    profileId: context.profileId,
    contextKind: context.contextKind,
    hostName: HOST_NAME,
    extensionVersion: EXTENSION_VERSION,
    browserFamily: context.browserFamily,
  });
  const source = new QueryClient({ profileId: context.profileId, transport: client });
  void client.start().catch(() => undefined);
  return {
    source,
    close: () => client.close(),
  };
}
