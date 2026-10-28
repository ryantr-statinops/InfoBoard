// Typed settings facade over the native-messaging lifecycle contract. It keeps
// local validation and profile binding in one place so the settings page never
// sends an unvalidated configuration or reads browser state on its own.

import type { ContextKind, ProfileID } from '../../domain/index.js';
import {
  NativeMessagingClient,
  type ClientRequestResult,
  type LifecycleConfiguration,
  type LifecycleOperation,
  type LifecycleResponse,
  type NativeMessagingClientOptions,
} from '../runtime/native-messaging-client.js';
import { validateProfileConfig, type ProfileConfig } from './contract.js';

export interface SettingsLifecycleClient {
  start(): Promise<ClientRequestResult<{ accepted_protocol: number; host_version: string }>>;
  load(): Promise<ClientRequestResult<LifecycleResponse>>;
  update(configuration: ProfileConfig): Promise<ClientRequestResult<LifecycleResponse>>;
  reset(): Promise<ClientRequestResult<LifecycleResponse>>;
  uninstall(): Promise<ClientRequestResult<LifecycleResponse>>;
  close(): void;
}

export interface SettingsLifecycleClientOptions {
  profileId: ProfileID;
  contextKind?: ContextKind;
  browserFamily?: 'chrome' | 'edge';
  browserVersion?: string;
  extensionVersion?: string;
  hostName?: string;
  createPort?: NativeMessagingClientOptions['createPort'];
  limits?: NativeMessagingClientOptions['limits'];
  deadlines?: NativeMessagingClientOptions['deadlines'];
  setTimer?: NativeMessagingClientOptions['setTimer'];
  clearTimer?: NativeMessagingClientOptions['clearTimer'];
}

const DEFAULT_HOST_NAME = 'com.infoboard.host';
const DEFAULT_EXTENSION_VERSION = '0.1.0';

function invalid<T>(): ClientRequestResult<T> {
  return { ok: false, source: 'client', error: { code: 'INVALID_SUBMISSION', retryable: false } };
}

export function createSettingsLifecycleClient(options: SettingsLifecycleClientOptions): SettingsLifecycleClient {
  const client = new NativeMessagingClient({
    profileId: options.profileId,
    contextKind: options.contextKind ?? 'normal',
    browserFamily: options.browserFamily ?? 'chrome',
    browserVersion: options.browserVersion,
    extensionVersion: options.extensionVersion ?? DEFAULT_EXTENSION_VERSION,
    hostName: options.hostName ?? DEFAULT_HOST_NAME,
    createPort: options.createPort,
    limits: options.limits,
    deadlines: options.deadlines,
    setTimer: options.setTimer,
    clearTimer: options.clearTimer,
  });

  const run = (operation: LifecycleOperation): Promise<ClientRequestResult<LifecycleResponse>> =>
    client.lifecycle(operation);

  return {
    start: () => client.start(),
    load: () => run('load'),
    update: configuration => {
      try {
        const validated = validateProfileConfig(configuration, options.profileId);
        return client.lifecycle('update', validated as LifecycleConfiguration);
      } catch {
        return Promise.resolve(invalid<LifecycleResponse>());
      }
    },
    reset: () => run('reset'),
    uninstall: () => run('uninstall'),
    close: () => client.close(),
  };
}
