// IP-17 scope binding for extension-side sessions, results, and activation.
//
// A scope is the tuple (browser_family, profile_id, context_kind). A private
// context may add an ephemeral nonce. Nothing in this module is persisted.

import {
  parseProfileID,
  tabIdentityKey,
  validateTab,
  type ContextKind,
  type EligibleTabRecord,
  type ProfileID,
  type ProjectionEpoch,
  type ProjectionRevision,
} from '../../domain/index.js';
import {
  guardActivation,
  type ActivationGuardInput,
  type ActivationGuardOutcome,
} from '../browser/activation-guard.js';

export type BrowserFamily = 'chrome' | 'edge';

export interface Scope {
  readonly browser_family: BrowserFamily;
  readonly profile_id: ProfileID;
  readonly context_kind: ContextKind;
  readonly private_nonce?: string;
}

export type ScopeFailure =
  | 'SCOPE_MISSING'
  | 'PROFILE_MISMATCH'
  | 'CONTEXT_MISMATCH'
  | 'REVISION_MISMATCH'
  | 'PRIVATE_CONTEXT_ENDED'
  | 'INVALID_SCOPE';

export type ScopeDecision =
  | { readonly ok: true }
  | { readonly ok: false; readonly code: ScopeFailure };

export type ScopedActivationOutcome = ActivationGuardOutcome | 'private_context_ended';

const MAX_NONCE_BYTES = 128;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function boundedOpaque(value: unknown): value is string {
  return typeof value === 'string'
    && value.length > 0
    && new TextEncoder().encode(value).length <= MAX_NONCE_BYTES;
}

export function parseScope(input: unknown): Scope {
  if (!isRecord(input)) throw new TypeError('INVALID_SCOPE');
  const browserFamily = input.browser_family;
  if (browserFamily !== 'chrome' && browserFamily !== 'edge') throw new TypeError('INVALID_SCOPE');
  const profileID = parseProfileID(input.profile_id);
  const contextKind = input.context_kind;
  if (contextKind !== 'normal' && contextKind !== 'private') throw new TypeError('INVALID_SCOPE');
  if (contextKind === 'normal') {
    if (input.private_nonce !== undefined) throw new TypeError('INVALID_SCOPE');
    return { browser_family: browserFamily, profile_id: profileID, context_kind: contextKind };
  }
  if (input.private_nonce !== undefined && !boundedOpaque(input.private_nonce)) {
    throw new TypeError('INVALID_SCOPE');
  }
  return {
    browser_family: browserFamily,
    profile_id: profileID,
    context_kind: contextKind,
    ...(input.private_nonce === undefined ? {} : { private_nonce: input.private_nonce }),
  };
}

export function sameScope(left: Scope, right: Scope): boolean {
  return left.browser_family === right.browser_family
    && left.profile_id === right.profile_id
    && left.context_kind === right.context_kind
    && left.private_nonce === right.private_nonce;
}

export function scopeKey(scope: Scope): string {
  return JSON.stringify([
    scope.browser_family,
    scope.profile_id,
    scope.context_kind,
    scope.private_nonce ?? null,
  ]);
}

export interface HostScopeResponse {
  readonly profile_id?: unknown;
  readonly context_kind?: unknown;
  readonly projection_revision?: unknown;
}

export class ScopeGuard {
  private readonly expected: Scope;
  private privateNonce: string | undefined;
  private privateActive: boolean;

  constructor(scope: Scope) {
    this.expected = scope;
    this.privateNonce = scope.private_nonce;
    this.privateActive = scope.context_kind === 'private';
  }

  scope(): Scope {
    return this.expected;
  }

  beginPrivateContext(nonce: string): void {
    if (!boundedOpaque(nonce)) throw new TypeError('INVALID_PRIVATE_NONCE');
    this.privateNonce = nonce;
    this.privateActive = true;
  }

  endPrivateContext(): void {
    this.privateNonce = undefined;
    this.privateActive = false;
  }

  isPrivateNonceCurrent(nonce: string): boolean {
    return this.expected.context_kind === 'private'
      && this.privateActive
      && this.privateNonce !== undefined
      && nonce === this.privateNonce;
  }

  acceptHostResponse(
    response: HostScopeResponse,
    expectedRevision?: ProjectionRevision,
  ): ScopeDecision {
    if (!isRecord(response) || response.profile_id === undefined) {
      return { ok: false, code: 'SCOPE_MISSING' };
    }
    let profile: ProfileID;
    try {
      profile = parseProfileID(response.profile_id);
    } catch {
      return { ok: false, code: 'PROFILE_MISMATCH' };
    }
    if (profile !== this.expected.profile_id) return { ok: false, code: 'PROFILE_MISMATCH' };
    if (response.context_kind !== undefined && response.context_kind !== this.expected.context_kind) {
      return { ok: false, code: 'CONTEXT_MISMATCH' };
    }
    if (expectedRevision !== undefined) {
      if (typeof response.projection_revision !== 'number'
        || !Number.isSafeInteger(response.projection_revision)
        || response.projection_revision !== Number(expectedRevision)) {
        return { ok: false, code: 'REVISION_MISMATCH' };
      }
    }
    if (this.expected.context_kind === 'private' && !this.privateActive) {
      return { ok: false, code: 'PRIVATE_CONTEXT_ENDED' };
    }
    return { ok: true };
  }

  acceptRecord(
    record: EligibleTabRecord,
    epoch: ProjectionEpoch,
    revision: ProjectionRevision,
  ): ScopeDecision {
    if (record.profile_id !== this.expected.profile_id
      || record.tab_identity.profile_id !== this.expected.profile_id) {
      return { ok: false, code: 'PROFILE_MISMATCH' };
    }
    if (record.context_kind !== this.expected.context_kind
      || record.tab_identity.context_kind !== this.expected.context_kind) {
      return { ok: false, code: 'CONTEXT_MISMATCH' };
    }
    if (this.expected.context_kind === 'private' && !this.privateActive) {
      return { ok: false, code: 'PRIVATE_CONTEXT_ENDED' };
    }
    if (!validateTab(record, this.expected.profile_id, this.expected.context_kind, epoch)
      || Number(record.projection_revision) !== Number(revision)) {
      return { ok: false, code: 'REVISION_MISMATCH' };
    }
    return { ok: true };
  }

  acceptActivation(input: ActivationGuardInput): { outcome: ScopedActivationOutcome } {
    if (input.reference.profile_id !== this.expected.profile_id
      || input.projection.profile_id !== this.expected.profile_id) {
      return { outcome: 'profile_mismatch' };
    }
    if (input.reference.context_kind !== this.expected.context_kind
      || input.projection.context_kind !== this.expected.context_kind) {
      return { outcome: 'context_mismatch' };
    }
    if (this.expected.context_kind === 'private' && !this.privateActive) {
      return { outcome: 'private_context_ended' };
    }
    return guardActivation(input);
  }
}

export type ArenaDecision =
  | { readonly ok: true; readonly admitted: number }
  | { readonly ok: false; readonly code: ScopeFailure };

export class ScopeArena {
  private normal = new Map<string, Map<string, EligibleTabRecord>>();
  private private = new Map<string, Map<string, EligibleTabRecord>>();
  private privateNonce: string | undefined;
  private privateActive = false;

  beginPrivateContext(nonce: string): void {
    if (!boundedOpaque(nonce)) throw new TypeError('INVALID_PRIVATE_NONCE');
    this.privateNonce = nonce;
    this.privateActive = true;
  }

  endPrivateContext(): void {
    this.private.clear();
    this.privateNonce = undefined;
    this.privateActive = false;
  }

  clearPrivate(): void {
    this.endPrivateContext();
  }

  admitSnapshot(
    scope: Scope,
    epoch: ProjectionEpoch,
    revision: ProjectionRevision,
    records: readonly EligibleTabRecord[],
  ): ArenaDecision {
    const decision = this.validateScope(scope);
    if (!decision.ok) return decision;
    const staged = new Map<string, EligibleTabRecord>();
    for (const record of records) {
      if (record.profile_id !== scope.profile_id
        || record.tab_identity.profile_id !== scope.profile_id) {
        return { ok: false, code: 'PROFILE_MISMATCH' };
      }
      if (record.context_kind !== scope.context_kind
        || record.tab_identity.context_kind !== scope.context_kind) {
        return { ok: false, code: 'CONTEXT_MISMATCH' };
      }
      if (!validateTab(record, scope.profile_id, scope.context_kind, epoch)
        || Number(record.projection_revision) !== Number(revision)) {
        return { ok: false, code: 'REVISION_MISMATCH' };
      }
      const key = tabIdentityKey(record.tab_identity);
      if (staged.has(key)) return { ok: false, code: 'INVALID_SCOPE' };
      staged.set(key, { ...record });
    }
    const target = scope.context_kind === 'private' ? this.private : this.normal;
    target.set(scopeKey(scope), staged);
    return { ok: true, admitted: staged.size };
  }

  admitRecord(
    scope: Scope,
    epoch: ProjectionEpoch,
    revision: ProjectionRevision,
    record: EligibleTabRecord,
  ): ArenaDecision {
    const decision = this.validateScope(scope);
    if (!decision.ok) return decision;
    if (record.profile_id !== scope.profile_id
      || record.tab_identity.profile_id !== scope.profile_id) {
      return { ok: false, code: 'PROFILE_MISMATCH' };
    }
    if (record.context_kind !== scope.context_kind
      || record.tab_identity.context_kind !== scope.context_kind) {
      return { ok: false, code: 'CONTEXT_MISMATCH' };
    }
    if (!validateTab(record, scope.profile_id, scope.context_kind, epoch)
      || Number(record.projection_revision) !== Number(revision)) {
      return { ok: false, code: 'REVISION_MISMATCH' };
    }
    const target = scope.context_kind === 'private' ? this.private : this.normal;
    const key = scopeKey(scope);
    const arena = target.get(key) ?? new Map<string, EligibleTabRecord>();
    arena.set(tabIdentityKey(record.tab_identity), { ...record });
    target.set(key, arena);
    return { ok: true, admitted: arena.size };
  }

  records(scope: Scope): readonly EligibleTabRecord[] {
    const decision = this.validateScope(scope);
    if (!decision.ok) return [];
    const target = scope.context_kind === 'private' ? this.private : this.normal;
    return [...(target.get(scopeKey(scope))?.values() ?? [])];
  }

  counts(): { readonly normal: number; readonly private: number } {
    const count = (arenas: Map<string, Map<string, EligibleTabRecord>>): number => {
      let total = 0;
      for (const arena of arenas.values()) total += arena.size;
      return total;
    };
    return { normal: count(this.normal), private: count(this.private) };
  }

  private validateScope(scope: Scope): ScopeDecision {
    if (scope.context_kind === 'private') {
      if (!this.privateActive) return { ok: false, code: 'PRIVATE_CONTEXT_ENDED' };
      if (scope.private_nonce !== undefined && scope.private_nonce !== this.privateNonce) {
        return { ok: false, code: 'PRIVATE_CONTEXT_ENDED' };
      }
    }
    return { ok: true };
  }
}
