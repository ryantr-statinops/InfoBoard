// IP-17 private-context lifetime and data flow.
//
// Private projection records live in an isolated arena. They are never merged
// with normal records, never persisted, and are discarded when the context ends.

import type { BrowserAdapter, BrowserResult } from '../browser/browser-adapter.js';
import type {
  ContextKind,
  EligibleTabRecord,
  ProjectionEpoch,
  ProjectionRevision,
} from '../../domain/index.js';
import {
  ScopeArena,
  ScopeGuard,
  type Scope,
  type ScopeDecision,
  type ScopedActivationOutcome,
} from './scope.js';

export type PrivateCapabilityState = 'available' | 'denied' | 'unsupported' | 'unavailable';

export interface PrivateCapability {
  readonly state: PrivateCapabilityState;
  readonly collectPrivate: boolean;
  readonly escalate: false;
}

export interface PrivateActivationOutcome {
  readonly persisted: false;
  readonly outcome: 'not_persisted' | 'ignored';
}

export function shouldPersistActivation(contextKind: ContextKind): boolean {
  return contextKind === 'normal';
}

export function privateActivationOutcome(
  contextKind: ContextKind,
  contextActive: boolean,
): PrivateActivationOutcome {
  if (contextKind === 'private') {
    return { persisted: false, outcome: contextActive ? 'not_persisted' : 'ignored' };
  }
  return { persisted: false, outcome: 'ignored' };
}

export async function probePrivateContext(
  adapter: Pick<BrowserAdapter, 'privateContextAccess'>,
): Promise<PrivateCapability> {
  let result: BrowserResult<boolean>;
  try {
    result = await adapter.privateContextAccess();
  } catch {
    return { state: 'unavailable', collectPrivate: false, escalate: false };
  }
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

export class PrivateLifetime {
  private readonly normalScope: Scope;
  private readonly privateScope: Scope;
  private readonly guard: ScopeGuard;
  private readonly arena = new ScopeArena();
  private active = false;

  constructor(normalScope: Scope, privateScope: Scope) {
    if (normalScope.context_kind !== 'normal' || privateScope.context_kind !== 'private') {
      throw new TypeError('INVALID_PRIVATE_SCOPE');
    }
    this.normalScope = normalScope;
    this.privateScope = privateScope;
    this.guard = new ScopeGuard(privateScope);
  }

  begin(nonce: string): void {
    this.guard.beginPrivateContext(nonce);
    this.arena.beginPrivateContext(nonce);
    this.active = true;
  }

  end(): void {
    this.guard.endPrivateContext();
    this.arena.endPrivateContext();
    this.active = false;
  }

  isActive(): boolean {
    return this.active;
  }

  admitNormalSnapshot(
    epoch: ProjectionEpoch,
    revision: ProjectionRevision,
    records: readonly EligibleTabRecord[],
  ) {
    return this.arena.admitSnapshot(this.normalScope, epoch, revision, records);
  }

  admitPrivateSnapshot(
    epoch: ProjectionEpoch,
    revision: ProjectionRevision,
    records: readonly EligibleTabRecord[],
  ) {
    if (!this.active) return { ok: false as const, code: 'PRIVATE_CONTEXT_ENDED' as const };
    return this.arena.admitSnapshot(this.privateScope, epoch, revision, records);
  }

  admitPrivateRecord(
    epoch: ProjectionEpoch,
    revision: ProjectionRevision,
    record: EligibleTabRecord,
  ) {
    if (!this.active) return { ok: false as const, code: 'PRIVATE_CONTEXT_ENDED' as const };
    return this.arena.admitRecord(this.privateScope, epoch, revision, record);
  }

  acceptLatePrivateResponse(
    nonce: string,
    response: { profile_id?: unknown; context_kind?: unknown; projection_revision?: unknown },
    revision: ProjectionRevision,
  ): ScopeDecision {
    if (!this.guard.isPrivateNonceCurrent(nonce)) {
      return { ok: false, code: 'PRIVATE_CONTEXT_ENDED' };
    }
    return this.guard.acceptHostResponse(response, revision);
  }

  acceptPrivateActivation(input: Parameters<ScopeGuard['acceptActivation']>[0]): {
    outcome: ScopedActivationOutcome;
  } {
    if (!this.active) return { outcome: 'private_context_ended' };
    return this.guard.acceptActivation(input);
  }

  normalRecords(): readonly EligibleTabRecord[] {
    return this.arena.records(this.normalScope);
  }

  privateRecords(): readonly EligibleTabRecord[] {
    return this.active ? this.arena.records(this.privateScope) : [];
  }

  counts(): { readonly normal: number; readonly private: number } {
    return this.arena.counts();
  }
}
