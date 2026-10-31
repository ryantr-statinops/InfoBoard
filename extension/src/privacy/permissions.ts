// IP-17 permission matrix and static audit.
//
// The matrix is normative for the Manifest V3 package. A capability that is not
// in this file is not approved. The source audit composes denied API names from
// fragments so this policy file does not itself contain a forbidden call.

import type { BrowserResult } from '../browser/browser-adapter.js';

export const CAPABILITIES = [
  'command_registration',
  'tab_metadata_and_activation',
  'window_metadata_and_focus',
  'tab_group_metadata',
  'private_context',
  'native_messaging',
  'extension_local_storage',
  'excluded_browser_surfaces',
] as const;

export type Capability = (typeof CAPABILITIES)[number];

export interface PermissionMatrixRow {
  readonly capability: Capability;
  readonly product_use: string;
  readonly owner: string;
  readonly requirement_ids: readonly string[];
  readonly manifest_permission: string | null;
  readonly denial_behavior: string;
  readonly evidence: string;
}

export const PERMISSION_MATRIX: readonly PermissionMatrixRow[] = [
  {
    capability: 'command_registration',
    product_use: 'Open the focused search surface.',
    owner: 'extension',
    requirement_ids: ['FR-001', 'FR-002'],
    manifest_permission: null,
    denial_behavior: 'Command remains visible or reports an install/configuration error; no runtime capability escalation.',
    evidence: 'Manifest command inspection and FX-SHORTCUT-FOCUS.',
  },
  {
    capability: 'tab_metadata_and_activation',
    product_use: 'Read eligible open-tab metadata and activate the validated current-scope tab.',
    owner: 'extension',
    requirement_ids: ['FR-003', 'FR-007'],
    manifest_permission: 'tabs',
    denial_behavior: 'Missing fields stay absent; search degrades to available fields and activation fails closed.',
    evidence: 'Manifest audit, API spy, and tab event fixtures.',
  },
  {
    capability: 'window_metadata_and_focus',
    product_use: 'Read window identity/label and focus the selected current-scope window.',
    owner: 'extension',
    requirement_ids: ['FR-003', 'FR-007'],
    manifest_permission: null,
    denial_behavior: 'Missing labels degrade to no-label context; focus reports a bounded failure and never chooses another profile.',
    evidence: 'Browser adapter capability test and activation fixture.',
  },
  {
    capability: 'tab_group_metadata',
    product_use: 'Use optional group identity/label as display context.',
    owner: 'extension',
    requirement_ids: ['FR-003'],
    manifest_permission: 'tabGroups',
    denial_behavior: 'Group fields are omitted; ranking and display continue without group context.',
    evidence: 'Permission matrix and denied-group fixture.',
  },
  {
    capability: 'private_context',
    product_use: 'Separate eligible private records only when the browser grants access.',
    owner: 'extension',
    requirement_ids: ['FR-014', 'NFR-009'],
    manifest_permission: null,
    denial_behavior: 'Report unavailable private context, collect nothing, and keep normal scope unchanged.',
    evidence: 'Private grant/deny/teardown fixtures.',
  },
  {
    capability: 'native_messaging',
    product_use: 'Send bounded local protocol messages to the registered Go host.',
    owner: 'extension+host',
    requirement_ids: ['FR-004', 'NFR-007'],
    manifest_permission: 'nativeMessaging',
    denial_behavior: 'Missing registration or origin mismatch yields unavailable/repairable state and sends no tab data.',
    evidence: 'Manifest/host-registration inspection and handshake fixture.',
  },
  {
    capability: 'extension_local_storage',
    product_use: 'Store only the opaque profile identity key in extension-local storage.',
    owner: 'extension',
    requirement_ids: ['NFR-009'],
    manifest_permission: 'storage',
    denial_behavior: 'Read/write failure fails closed before host/tab work; an invalid stored ID is not replaced.',
    evidence: 'Exact-key storage spy and malformed-value fixture.',
  },
  {
    capability: 'excluded_browser_surfaces',
    product_use: 'Not used. History, saved items, downloads, cookies, page content, network interception, and broad host access are outside the product boundary.',
    owner: 'none',
    requirement_ids: ['NFR-007', 'NFR-009'],
    manifest_permission: null,
    denial_behavior: 'No data is collected and no permission prompt is shown. A future request requires a new decision record.',
    evidence: 'Manifest denylist, source/API scan, and runtime spies.',
  },
];

export const APPROVED_MANIFEST_PERMISSIONS = ['nativeMessaging', 'storage', 'tabGroups', 'tabs'] as const;
export const APPROVED_COMMANDS = ['close-search', 'open-search'] as const;
export const DENIED_MANIFEST_PERMISSIONS = [
  'bookmarks',
  'browsingData',
  'contentSettings',
  'cookies',
  'debugger',
  'declarativeNetRequest',
  'downloads',
  'history',
  'management',
  'privacy',
  'proxy',
  'scripting',
  'sessions',
  'topSites',
  'webNavigation',
  'webRequest',
] as const;

export interface ManifestAuditResult {
  readonly ok: boolean;
  readonly findings: readonly string[];
  readonly permissions: readonly string[];
  readonly commands: readonly string[];
}

export interface SourceAuditResult {
  readonly ok: boolean;
  readonly findings: readonly string[];
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function stringArray(value: unknown): string[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'string') ? [...value] : [];
}

function sorted(values: readonly string[]): string[] {
  return [...values].sort();
}

function sameSet(left: readonly string[], right: readonly string[]): boolean {
  return sorted(left).join('\u0000') === sorted(right).join('\u0000');
}

export function auditManifest(manifest: unknown): ManifestAuditResult {
  const findings: string[] = [];
  if (!isRecord(manifest)) {
    return { ok: false, findings: ['manifest_not_object'], permissions: [], commands: [] };
  }
  if (manifest.manifest_version !== 3) findings.push('manifest_version_not_3');

  const permissions = stringArray(manifest.permissions);
  if (!sameSet(permissions, APPROVED_MANIFEST_PERMISSIONS)) findings.push('permissions_not_approved_set');
  for (const denied of DENIED_MANIFEST_PERMISSIONS) {
    if (permissions.includes(denied)) findings.push(`denied_permission:${denied}`);
  }

  const hostPermissions = stringArray(manifest.host_permissions);
  const optionalPermissions = stringArray(manifest.optional_permissions);
  const optionalHostPermissions = stringArray(manifest.optional_host_permissions);
  if (hostPermissions.length > 0) findings.push('host_permissions_present');
  if (optionalPermissions.length > 0) findings.push('optional_permissions_present');
  if (optionalHostPermissions.length > 0) findings.push('optional_host_permissions_present');
  for (const pattern of [...hostPermissions, ...optionalHostPermissions]) {
    if (pattern === '<all_urls>' || pattern.includes('*')) findings.push('broad_host_pattern');
  }
  if (Array.isArray(manifest.content_scripts) && manifest.content_scripts.length > 0) findings.push('content_scripts_present');
  if (Array.isArray(manifest.web_accessible_resources) && manifest.web_accessible_resources.length > 0) {
    findings.push('web_accessible_resources_present');
  }

  const commands = isRecord(manifest.commands) ? Object.keys(manifest.commands).sort() : [];
  if (!sameSet(commands, APPROVED_COMMANDS)) findings.push('commands_not_approved_set');

  return { ok: findings.length === 0, findings, permissions: sorted(permissions), commands };
}

function deniedSourcePatterns(): readonly RegExp[] {
  const chromeApi = ['chrome', ''].join('.');
  const deniedApis = [
    'bookmarks',
    'browsingData',
    'contentSettings',
    'cookies',
    'debugger',
    'downloads',
    'history',
    'management',
    'privacy',
    'proxy',
    'scripting',
    'sessions',
    'topSites',
    'webNavigation',
    'webRequest',
  ];
  const tabMutations = ['update', 'create', 'remove', 'move', 'discard', 'reload'];
  const windowMutations = ['create', 'update', 'remove'];
  return [
    new RegExp('\\b' + 'fetch' + '\\s*\\('),
    new RegExp('\\bnew\\s+' + 'XMLHttpRequest' + '\\b'),
    new RegExp('\\bnew\\s+' + 'WebSocket' + '\\b'),
    new RegExp('\\bnew\\s+' + 'EventSource' + '\\b'),
    new RegExp('\\bnavigator\\.' + 'sendBeacon' + '\\b'),
    new RegExp('\\b' + chromeApi + '\\.tabs\\.(?:' + tabMutations.join('|') + ')\\s*\\('),
    new RegExp('\\b' + chromeApi + '\\.windows\\.(?:' + windowMutations.join('|') + ')\\s*\\('),
    ...deniedApis.map((name) => new RegExp('\\b' + chromeApi + '\\.' + name + '\\b')),
    new RegExp('\\b' + 'storage' + '\\.' + 'sync' + '\\b'),
    new RegExp('\\b' + 'document' + '\\.' + 'cookie' + '\\b'),
    new RegExp('\\b' + 'local' + 'Storage' + '\\s*(?:\\.|\\[)'),
    new RegExp('\\b' + 'session' + 'Storage' + '\\s*(?:\\.|\\[)'),
    new RegExp('\\b' + 'eval' + '\\s*\\('),
    new RegExp('\\bnew\\s+' + 'Function' + '\\s*\\('),
  ];
}

const SOURCE_PATTERNS = deniedSourcePatterns();

export function auditSource(source: string): SourceAuditResult {
  const findings: string[] = [];
  for (const [index, pattern] of SOURCE_PATTERNS.entries()) {
    if (pattern.test(source)) findings.push(`denied_source_pattern:${index}`);
  }
  return { ok: findings.length === 0, findings };
}

export type CapabilityState = 'available' | 'denied' | 'unsupported' | 'degraded' | 'unavailable';

export function capabilityState(result: BrowserResult<unknown>): CapabilityState {
  if (result.ok) return 'available';
  switch (result.error.kind) {
    case 'permission_denied':
      return 'denied';
    case 'unsupported':
      return 'unsupported';
    case 'cancelled':
    case 'operation_failed':
      return 'degraded';
    case 'not_found':
      return 'unavailable';
    default:
      return 'unavailable';
  }
}

export interface PrivateAccessDecision {
  readonly state: 'available' | 'unavailable' | 'denied' | 'unsupported';
  readonly collectPrivate: boolean;
  readonly escalate: false;
}

export function privateAccessDecision(result: BrowserResult<boolean>): PrivateAccessDecision {
  if (!result.ok) {
    if (result.error.kind === 'permission_denied') {
      return { state: 'denied', collectPrivate: false, escalate: false };
    }
    if (result.error.kind === 'unsupported') {
      return { state: 'unsupported', collectPrivate: false, escalate: false };
    }
    return { state: 'unavailable', collectPrivate: false, escalate: false };
  }
  return result.value
    ? { state: 'available', collectPrivate: true, escalate: false }
    : { state: 'unavailable', collectPrivate: false, escalate: false };
}
