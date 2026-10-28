import { ChromeBrowserAdapter } from '../browser/browser-adapter.js';
import { parseProfileID, type ProfileID } from '../../domain/index.js';
import { createSettingsLifecycleClient, type SettingsLifecycleClient } from './lifecycle-client.js';
import { SettingsController } from './settings-controller.js';

export interface SettingsTestDriver {
  profileId: ProfileID | string;
  client: SettingsLifecycleClient;
  confirm?: (message: string) => boolean;
  close?: () => void;
}

declare global {
  interface Window {
    __INFOBOARD_SETTINGS_TEST__?: SettingsTestDriver;
  }
}

function reportError(doc: Document, message: string): void {
  doc.documentElement.dataset.settingsState = 'error';
  const status = doc.querySelector<HTMLElement>('[data-settings-status]');
  if (status) status.textContent = message;
}

async function startController(
  doc: Document,
  profileId: ProfileID,
  client: SettingsLifecycleClient,
  confirm?: (message: string) => boolean,
  close?: () => void,
): Promise<void> {
  const controller = new SettingsController({ doc, profileId, client, confirm, close });
  doc.defaultView?.addEventListener('pagehide', () => controller.stop(), { once: true });
  await controller.start();
}

export async function initializeSettingsPage(doc: Document = document): Promise<void> {
  const driver = (globalThis as typeof globalThis & { __INFOBOARD_SETTINGS_TEST__?: SettingsTestDriver }).__INFOBOARD_SETTINGS_TEST__;
  if (driver) {
    let profileId: ProfileID;
    try {
      profileId = parseProfileID(driver.profileId);
    } catch {
      reportError(doc, 'This browser profile is unavailable. Reopen the extension settings.');
      return;
    }
    await startController(doc, profileId, driver.client, driver.confirm, driver.close);
    return;
  }

  const adapter = new ChromeBrowserAdapter();
  let rawProfile: unknown;
  try {
    rawProfile = await adapter.read();
  } catch {
    reportError(doc, 'This browser profile is unavailable. Reopen the extension settings.');
    return;
  }
  let profileId: ProfileID;
  try {
    profileId = parseProfileID(rawProfile);
  } catch {
    reportError(doc, 'This browser profile is unavailable. Reopen the extension settings.');
    return;
  }
  const browserFamily = /(?:Edg)\//.test(globalThis.navigator?.userAgent ?? '') ? 'edge' : 'chrome';
  const client = createSettingsLifecycleClient({ profileId, contextKind: 'normal', browserFamily });
  await startController(doc, profileId, client);
}

if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', () => { void initializeSettingsPage(); }, { once: true });
else void initializeSettingsPage();
