import {
  createProjectionEpoch,
  parseRevision,
  tabIdentityKey,
  type ContextKind,
  type EligibleTabRecord,
  type ProfileID,
  type ProjectionEpoch,
  type ProjectionRevision,
  type TabIdentity,
} from '../../domain/index.js';
import { evaluateTabEligibility } from './eligibility.js';
import type { BrowserAdapter, BrowserResult } from './browser-adapter.js';
import {
  normalizeBrowserEvent,
  type BrowserApiEvent,
  type BrowserGroupSnapshot,
  type BrowserTabSnapshot,
  type BrowserTabSnapshotBatch,
  type BrowserWindowSnapshot,
  type NormalizedBrowserEvent,
} from './event-normalizer.js';

export type ObserverStatus = 'not_started' | 'starting' | 'ready' | 'degraded' | 'unavailable' | 'stopped';
export type ResyncReason =
  | 'startup'
  | 'worker_restart'
  | 'permission_denied'
  | 'unsupported'
  | 'snapshot_failed'
  | 'malformed_event'
  | 'event_queue_overflow'
  | 'uncertain_order'
  | 'missing_required_state'
  | 'tab_id_reuse'
  | 'private_context_ended'
  | 'browser_shutdown'
  | 'reset'
  | 'disconnect';

export interface ObserverDiagnostics {
  eventsReceived: number;
  effectiveChanges: number;
  duplicateEvents: number;
  excludedTabs: number;
  deferredTabs: number;
  resyncs: number;
  requiredCapabilityErrors: number;
  optionalCapabilityErrors: number;
  handoffListenerErrors: number;
}

export interface ObserverHandoff {
  kind: 'snapshot' | 'upsert' | 'remove' | 'window' | 'status' | 'private_context_ended' | 'disposed';
  profile_id?: ProfileID;
  context_kind?: ContextKind;
  projection_epoch?: ProjectionEpoch;
  previous_revision: ProjectionRevision;
  projection_revision: ProjectionRevision;
  event_sequence: number;
  effective_change: boolean;
  resync_required: boolean;
  resync_reason?: ResyncReason;
  tab_identity?: TabIdentity;
  record?: EligibleTabRecord;
  records?: EligibleTabRecord[];
  diagnostics: ObserverDiagnostics;
}

export interface TabObserverView {
  status: ObserverStatus;
  resync_required: boolean;
  resync_reason?: ResyncReason;
  records: EligibleTabRecord[];
  diagnostics: ObserverDiagnostics;
}

interface ContextProjection {
  contextKind: ContextKind;
  epoch: ProjectionEpoch | null;
  revision: ProjectionRevision;
  records: Map<string, EligibleTabRecord>;
  rawTabs: Map<string, BrowserTabSnapshot>;
  windows: BrowserWindowSnapshot[];
  groups: BrowserGroupSnapshot[];
  status: 'ready' | 'degraded' | 'unavailable';
  resyncRequired: boolean;
}

const MAX_TABS = 10000;
const MAX_QUEUED_EVENTS = 1000;

function newContext(contextKind: ContextKind): ContextProjection {
  return {
    contextKind,
    epoch: null,
    revision: parseRevision(0),
    records: new Map(),
    rawTabs: new Map(),
    windows: [],
    groups: [],
    status: 'unavailable',
    resyncRequired: true,
  };
}

function safeId(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0 ? value : undefined;
}

function contextForTab(tab: BrowserTabSnapshot | undefined): ContextKind | undefined {
  if (tab?.incognito === true) return 'private';
  if (tab?.incognito === false) return 'normal';
  return undefined;
}

function sameRecord(left: EligibleTabRecord, right: EligibleTabRecord): boolean {
  return left.profile_id === right.profile_id
    && left.context_kind === right.context_kind
    && left.tab_identity.tab_id === right.tab_identity.tab_id
    && left.window_id === right.window_id
    && left.group_id === right.group_id
    && left.title_display === right.title_display
    && left.title_search === right.title_search
    && left.url_search === right.url_search
    && left.url_display === right.url_display
    && left.domain_display === right.domain_display
    && left.domain_search === right.domain_search
    && left.window_label_display === right.window_label_display
    && left.window_label_search === right.window_label_search
    && left.group_label_display === right.group_label_display
    && left.group_label_search === right.group_label_search
    && left.pinned === right.pinned
    && left.active === right.active
    && left.eligible === right.eligible
    && left.projection_epoch === right.projection_epoch;
}

function errorReason(result: Extract<BrowserResult<never>, { ok: false }>): ResyncReason {
  if (result.error.kind === 'permission_denied') return 'permission_denied';
  if (result.error.kind === 'unsupported') return 'unsupported';
  return 'snapshot_failed';
}

export class TabObserver {
  private started = false;
  private snapshotting = false;
  private processing = false;
  private overflowPending = false;
  private profileId: ProfileID | undefined;
  private unsubscribe: (() => void) | undefined;
  private privateAccess = false;
  private privateUnavailableReason: ResyncReason = 'unsupported';
  private pendingEvents: BrowserApiEvent[] = [];
  private eventSequence = 0;
  private status: ObserverStatus = 'not_started';
  private resyncRequired = true;
  private resyncReason: ResyncReason | undefined;
  private readonly listeners = new Set<(handoff: ObserverHandoff) => void>();
  private readonly removedTabIds = new Set<number>();
  private readonly reusedTabIds = new Set<number>();
  private readonly diagnostics: ObserverDiagnostics = {
    eventsReceived: 0,
    effectiveChanges: 0,
    duplicateEvents: 0,
    excludedTabs: 0,
    deferredTabs: 0,
    resyncs: 0,
    requiredCapabilityErrors: 0,
    optionalCapabilityErrors: 0,
    handoffListenerErrors: 0,
  };
  private contexts: Record<ContextKind, ContextProjection> = {
    normal: newContext('normal'),
    private: newContext('private'),
  };

  constructor(private readonly adapter: BrowserAdapter, private readonly now: () => number = Date.now) {}

  subscribe(listener: (handoff: ObserverHandoff) => void): () => void {
    this.listeners.add(listener);
    return () => { this.listeners.delete(listener); };
  }

  view(): TabObserverView {
    return {
      status: this.status,
      resync_required: this.resyncRequired,
      ...(this.resyncReason === undefined ? {} : { resync_reason: this.resyncReason }),
      records: [...this.contexts.normal.records.values(), ...this.contexts.private.records.values()],
      diagnostics: { ...this.diagnostics },
    };
  }

  async start(): Promise<void> {
    if (this.started) return;
    this.started = true;
    this.status = 'starting';
    const profile = await this.adapter.currentProfileContext();
    if (!profile.ok) {
      this.markUnavailable(profile.error.kind === 'permission_denied' ? 'permission_denied' : 'snapshot_failed');
      this.started = false;
      return;
    }
    this.profileId = profile.value.profileId;
    const access = await this.adapter.privateContextAccess();
    this.privateAccess = access.ok && access.value;
    this.privateUnavailableReason = access.ok ? 'permission_denied' : access.error.kind === 'unsupported' ? 'unsupported' : 'permission_denied';
    this.snapshotting = true;
    const registration = this.adapter.registerTabObservationListeners(event => this.enqueue(event));
    if (!registration.ok) {
      this.snapshotting = false;
      this.markUnavailable(registration.error.kind === 'permission_denied' ? 'permission_denied' : registration.error.kind === 'unsupported' ? 'unsupported' : 'snapshot_failed');
      this.started = false;
      return;
    }
    this.unsubscribe = registration.value.unsubscribe;
    this.diagnostics.optionalCapabilityErrors += registration.value.missingOptionalCapabilities;
    this.snapshotting = false;
    await this.refreshSnapshot('startup');
    if (this.view().status === 'unavailable') {
      this.unsubscribe?.();
      this.unsubscribe = undefined;
      this.started = false;
      return;
    }
    void this.drain();
  }

  async requestResync(reason: ResyncReason = 'uncertain_order'): Promise<void> {
    if (!this.started || !this.profileId) return;
    await this.refreshSnapshot(reason);
  }

  dispose(reason: 'browser_shutdown' | 'reset' | 'disconnect' = 'browser_shutdown'): void {
    this.unsubscribe?.();
    this.unsubscribe = undefined;
    this.pendingEvents = [];
    for (const contextKind of ['normal', 'private'] as const) {
      const state = this.contexts[contextKind];
      state.records.clear();
      state.rawTabs.clear();
      state.windows = [];
      state.groups = [];
      state.status = 'unavailable';
      state.resyncRequired = true;
    }
    this.started = false;
    this.status = 'stopped';
    this.resyncRequired = true;
    this.resyncReason = reason;
    this.publish({ kind: 'disposed', previous_revision: parseRevision(0), projection_revision: parseRevision(0), event_sequence: this.eventSequence, effective_change: true, resync_required: true, resync_reason: reason });
  }

  private enqueue(event: BrowserApiEvent): void {
    if (!this.started) return;
    if (this.pendingEvents.length >= MAX_QUEUED_EVENTS) {
      this.pendingEvents = [];
      this.overflowPending = true;
    } else {
      this.pendingEvents.push(event);
    }
    if (!this.snapshotting && !this.processing) void this.drain();
  }

  private async drain(): Promise<void> {
    if (this.processing || this.snapshotting || !this.started) return;
    this.processing = true;
    try {
      while (this.pendingEvents.length > 0 || this.overflowPending) {
        if (this.overflowPending) {
          this.overflowPending = false;
          this.pendingEvents = [];
          await this.refreshSnapshot('event_queue_overflow');
          continue;
        }
        const event = this.pendingEvents.shift();
        if (event) await this.handleEvent(event);
      }
    } finally {
      this.processing = false;
      if (this.pendingEvents.length > 0 || this.overflowPending) void this.drain();
    }
  }

  private async handleEvent(input: BrowserApiEvent): Promise<void> {
    this.diagnostics.eventsReceived += 1;
    this.eventSequence += 1;
    if (!Number.isSafeInteger(this.eventSequence)) {
      this.eventSequence = 1;
      await this.refreshSnapshot('uncertain_order');
      return;
    }
    const sequence = this.eventSequence;
    const result = normalizeBrowserEvent(input);
    if (!result.ok) {
      await this.refreshSnapshot(result.reason === 'malformed_event' ? 'malformed_event' : 'missing_required_state');
      return;
    }
    const event = result.event;
    if (!this.privateAccess && event.contextKind === 'private') return;
    if (event.requiresSnapshot) {
      await this.refreshSnapshot(event.kind === 'detach' ? 'uncertain_order' : 'uncertain_order');
      return;
    }
    if (event.kind === 'window_changed') {
      this.publish({
        kind: 'window',
        ...(this.profileId === undefined ? {} : { profile_id: this.profileId }),
        ...(event.contextKind === undefined ? {} : { context_kind: event.contextKind }),
        previous_revision: this.contexts.normal.revision,
        projection_revision: this.contexts.normal.revision,
        event_sequence: sequence,
        effective_change: true,
        resync_required: this.resyncRequired,
        ...(this.resyncReason === undefined ? {} : { resync_reason: this.resyncReason }),
      });
      return;
    }
    if (event.kind === 'window_removed' && event.windowId !== undefined && event.windowId !== null) {
      this.removeWindow(event.windowId, sequence);
      return;
    }
    if (event.kind === 'group') {
      await this.applyGroupEvent(event, sequence);
      return;
    }
    if (event.kind === 'remove' && event.tabId !== undefined) {
      this.removeTab(event.tabId, event.contextKind, sequence);
      return;
    }
    if (event.kind === 'activate' && event.tabId !== undefined && event.windowId !== undefined && event.windowId !== null) {
      await this.applyActivation(event.tabId, event.windowId, sequence);
      return;
    }
    if ((event.kind === 'create' || event.kind === 'update' || event.kind === 'move' || event.kind === 'pin') && event.tabId !== undefined) {
      if (this.reusedTabIds.has(event.tabId) || this.removedTabIds.has(event.tabId) && event.kind !== 'create') {
        this.reusedTabIds.add(event.tabId);
        await this.refreshSnapshot('tab_id_reuse');
        return;
      }
      if (event.kind === 'create' && this.removedTabIds.has(event.tabId)) {
        this.reusedTabIds.add(event.tabId);
        await this.refreshSnapshot('tab_id_reuse');
        return;
      }
      await this.applyTabPatch(event, sequence);
      return;
    }
    await this.refreshSnapshot('missing_required_state');
  }

  private async applyTabPatch(event: NormalizedBrowserEvent, sequence: number): Promise<void> {
    if (!this.profileId || event.tabId === undefined) return;
    const contextKind = event.contextKind ?? this.findContext(event.tabId);
    if (!contextKind) {
      await this.refreshSnapshot('missing_required_state');
      return;
    }
    const state = this.contexts[contextKind];
    const key = this.key(contextKind, event.tabId);
    const previous = state.rawTabs.get(key);
    if (!previous && event.kind !== 'create') {
      await this.refreshSnapshot('uncertain_order');
      return;
    }
    const raw: BrowserTabSnapshot = { ...(previous ?? {}), ...(event.patch ?? {}), id: event.tabId };
    state.rawTabs.set(key, raw);
    const nextRevision = this.nextRevision(state);
    if (!nextRevision) {
      await this.refreshSnapshot('uncertain_order');
      return;
    }
    const decision = evaluateTabEligibility(raw, {
      profileId: this.profileId,
      contextKind,
      projectionEpoch: state.epoch ?? createProjectionEpoch(),
      projectionRevision: nextRevision,
      observedAt: this.now(),
      windows: state.windows,
      groups: state.groups,
    });
    if (decision.decision === 'deferred') {
      this.diagnostics.deferredTabs += 1;
      this.removeRecord(state, key, sequence);
      await this.refreshSnapshot('missing_required_state');
      return;
    }
    if (decision.decision === 'excluded') {
      this.diagnostics.excludedTabs += 1;
      this.removeRecord(state, key, sequence);
      return;
    }
    const existing = state.records.get(key);
    if (existing && sameRecord(existing, decision.record)) {
      this.diagnostics.duplicateEvents += 1;
      this.publish({
        kind: 'upsert', profile_id: this.profileId, context_kind: contextKind,
        ...(state.epoch === null ? {} : { projection_epoch: state.epoch }),
        previous_revision: state.revision, projection_revision: state.revision,
        event_sequence: sequence, effective_change: false, resync_required: this.resyncRequired,
        ...(this.resyncReason === undefined ? {} : { resync_reason: this.resyncReason }),
        tab_identity: decision.identity, record: existing,
      });
      return;
    }
    const previousRevision = state.revision;
    state.revision = decision.record.projection_revision;
    state.records.set(key, decision.record);
    this.diagnostics.effectiveChanges += 1;
    this.publish({
      kind: 'upsert', profile_id: this.profileId, context_kind: contextKind,
      ...(state.epoch === null ? {} : { projection_epoch: state.epoch }),
      previous_revision: previousRevision, projection_revision: state.revision,
      event_sequence: sequence, effective_change: true, resync_required: this.resyncRequired,
      ...(this.resyncReason === undefined ? {} : { resync_reason: this.resyncReason }),
      tab_identity: decision.identity, record: decision.record,
    });
  }

  private async applyActivation(tabId: number, windowId: number, sequence: number): Promise<void> {
    for (const contextKind of ['normal', 'private'] as const) {
      const state = this.contexts[contextKind];
      const targetKey = this.key(contextKind, tabId);
      if (!state.rawTabs.has(targetKey)) continue;
      for (const [key, raw] of state.rawTabs) {
        if (safeId(raw.windowId) !== windowId) continue;
        await this.applyTabPatch({
          source: 'tabs.onActivated',
          kind: 'activate',
          tabId: safeId(raw.id),
          contextKind,
          windowId,
          patch: { active: safeId(raw.id) === tabId },
          changedFields: ['active'],
        }, sequence);
      }
      return;
    }
    await this.refreshSnapshot('uncertain_order');
  }

  private async applyGroupEvent(event: NormalizedBrowserEvent, sequence: number): Promise<void> {
    if (event.requiresSnapshot) {
      await this.refreshSnapshot('uncertain_order');
      return;
    }
    if (event.groupId === undefined) {
      await this.refreshSnapshot('missing_required_state');
      return;
    }
    for (const contextKind of ['normal', 'private'] as const) {
      const state = this.contexts[contextKind];
      const group = state.groups.find(candidate => safeId(candidate.id) === event.groupId);
      if (event.removedGroup) {
        state.groups = state.groups.filter(candidate => safeId(candidate.id) !== event.groupId);
      } else if (group && event.patch && Object.hasOwn(event.patch, 'groupLabel')) {
        const title = event.patch.groupLabel;
        state.groups = state.groups.map(candidate => safeId(candidate.id) === event.groupId ? { ...candidate, title } : candidate);
      }
      for (const [key, raw] of state.rawTabs) {
        if (safeId(raw.groupId) !== event.groupId || event.windowId !== undefined && safeId(raw.windowId) !== event.windowId) continue;
        const patch: BrowserTabSnapshot = event.removedGroup ? { groupId: -1, groupLabel: undefined } : { ...(event.patch ?? {}) };
        await this.applyTabPatch({
          source: event.source,
          kind: 'group',
          tabId: safeId(raw.id),
          contextKind,
          windowId: safeId(raw.windowId),
          groupId: event.groupId,
          patch,
          changedFields: event.changedFields,
        }, sequence);
      }
    }
  }

  private removeWindow(windowId: number, sequence: number): void {
    for (const contextKind of ['normal', 'private'] as const) {
      const state = this.contexts[contextKind];
      const removedPrivateWindow = contextKind === 'private' && state.windows.some(window => safeId(window.id) === windowId);
      state.windows = state.windows.filter(window => safeId(window.id) !== windowId);
      const hadPrivateRecords = removedPrivateWindow && (state.records.size > 0 || state.rawTabs.size > 0);
      for (const [key, raw] of state.rawTabs) {
        if (safeId(raw.windowId) !== windowId) continue;
        const tabId = safeId(raw.id);
        if (tabId !== undefined) this.removedTabIds.add(tabId);
        this.removeRecord(state, key, sequence);
        state.rawTabs.delete(key);
      }
      if (removedPrivateWindow && state.windows.length === 0) {
        state.records.clear();
        state.rawTabs.clear();
        if (hadPrivateRecords) this.publish({
          kind: 'private_context_ended', profile_id: this.profileId, context_kind: 'private',
          ...(state.epoch === null ? {} : { projection_epoch: state.epoch }),
          previous_revision: state.revision, projection_revision: state.revision,
          event_sequence: sequence, effective_change: true, resync_required: this.resyncRequired,
          ...(this.resyncReason === undefined ? {} : { resync_reason: this.resyncReason }),
        });
      }
    }
  }

  private removeTab(tabId: number, eventContext: ContextKind | undefined, sequence: number): void {
    const matches = eventContext === undefined
      ? (['normal', 'private'] as const).filter(contextKind => this.contexts[contextKind].rawTabs.has(this.key(contextKind, tabId)))
      : [eventContext];
    if (matches.length !== 1 || !this.contexts[matches[0]].rawTabs.has(this.key(matches[0], tabId))) {
      this.diagnostics.duplicateEvents += 1;
      if (matches.length > 1) void this.refreshSnapshot('uncertain_order');
      return;
    }
    const contextKind = matches[0];
    const state = this.contexts[contextKind];
    const key = this.key(contextKind, tabId);
    state.rawTabs.delete(key);
    this.removedTabIds.add(tabId);
    this.removeRecord(state, key, sequence);
  }

  private removeRecord(state: ContextProjection, key: string, sequence: number): void {
    const record = state.records.get(key);
    if (!record || !this.profileId) return;
    const previousRevision = state.revision;
    const nextRevision = this.nextRevision(state);
    if (!nextRevision) {
      void this.refreshSnapshot('uncertain_order');
      return;
    }
    state.records.delete(key);
    state.revision = nextRevision;
    this.diagnostics.effectiveChanges += 1;
    this.publish({
      kind: 'remove', profile_id: this.profileId, context_kind: state.contextKind,
      ...(state.epoch === null ? {} : { projection_epoch: state.epoch }),
      previous_revision: previousRevision, projection_revision: state.revision,
      event_sequence: sequence, effective_change: true, resync_required: this.resyncRequired,
      ...(this.resyncReason === undefined ? {} : { resync_reason: this.resyncReason }),
      tab_identity: record.tab_identity,
    });
  }

  private async refreshSnapshot(reason: ResyncReason): Promise<void> {
    if (!this.started || !this.profileId || this.snapshotting) return;
    this.snapshotting = true;
    this.diagnostics.resyncs += reason === 'startup' ? 0 : 1;
    const previousPrivateCount = this.contexts.private.records.size;
    const result = await this.adapter.getOpenTabSnapshot();
    if (!result.ok) {
      this.markUnavailable(errorReason(result));
      this.snapshotting = false;
      return;
    }
    this.applySnapshot(result.value, reason, previousPrivateCount);
    this.snapshotting = false;
  }

  private applySnapshot(batch: BrowserTabSnapshotBatch, reason: ResyncReason, previousPrivateCount: number): void {
    if (batch.tabs.length > MAX_TABS) {
      this.markUnavailable('snapshot_failed');
      return;
    }
    this.diagnostics.optionalCapabilityErrors += batch.optionalCapabilityErrors;
    const next: Record<ContextKind, ContextProjection> = {
      normal: { ...newContext('normal'), epoch: createProjectionEpoch(), revision: parseRevision(1), resyncRequired: false },
      private: { ...newContext('private'), epoch: createProjectionEpoch(), revision: parseRevision(1), resyncRequired: false },
    };
    for (const window of batch.windows) {
      const contextKind = window.incognito === true ? 'private' : window.incognito === false ? 'normal' : undefined;
      if (contextKind && (this.privateAccess || contextKind === 'normal')) next[contextKind].windows.push(window);
    }
    for (const group of batch.groups) {
      const id = safeId(group.id), windowId = safeId(group.windowId);
      const parent = batch.windows.find(window => safeId(window.id) === windowId);
      const contextKind = parent?.incognito === true ? 'private' : parent?.incognito === false ? 'normal' : undefined;
      if (id !== undefined && contextKind && (this.privateAccess || contextKind === 'normal')) next[contextKind].groups.push(group);
    }
    for (const tab of batch.tabs) {
      const contextKind = contextForTab(tab);
      const tabId = safeId(tab.id);
      if (contextKind === 'private' && !this.privateAccess) continue;
      if (!contextKind || tabId === undefined) {
        this.diagnostics.deferredTabs += 1;
        continue;
      }
      const state = next[contextKind];
      const key = this.key(contextKind, tabId);
      state.rawTabs.set(key, tab);
      const decision = evaluateTabEligibility(tab, {
        profileId: this.profileId!,
        contextKind,
        projectionEpoch: state.epoch!,
        projectionRevision: state.revision,
        observedAt: this.now(),
        windows: state.windows,
        groups: state.groups,
      });
      if (decision.decision === 'eligible') {
        if (state.records.has(decision.identityKey)) {
          state.resyncRequired = true;
          this.diagnostics.deferredTabs += 1;
        } else {
          state.records.set(decision.identityKey, decision.record);
        }
      } else if (decision.decision === 'excluded') {
        this.diagnostics.excludedTabs += 1;
      } else {
        state.resyncRequired = true;
        this.diagnostics.deferredTabs += 1;
      }
    }
    this.contexts = next;
    this.removedTabIds.clear();
    const incomplete = next.normal.resyncRequired || this.privateAccess && next.private.resyncRequired;
    if (!this.privateAccess) next.private.status = 'unavailable';
    this.resyncRequired = incomplete;
    this.resyncReason = incomplete ? 'missing_required_state' : undefined;
    this.status = incomplete ? 'degraded' : 'ready';
    for (const contextKind of ['normal', 'private'] as const) {
      const state = next[contextKind];
      state.status = contextKind === 'private' && !this.privateAccess ? 'unavailable' : state.resyncRequired ? 'degraded' : 'ready';
      this.publish({
        kind: 'snapshot', profile_id: this.profileId, context_kind: contextKind,
        projection_epoch: state.epoch!, previous_revision: parseRevision(0), projection_revision: state.revision,
        event_sequence: this.eventSequence, effective_change: true, resync_required: state.resyncRequired || contextKind === 'private' && !this.privateAccess,
        ...(contextKind === 'private' && !this.privateAccess ? { resync_reason: this.privateUnavailableReason } : state.resyncRequired ? { resync_reason: 'missing_required_state' as const } : reason === 'startup' ? {} : { resync_reason: reason }),
        records: [...state.records.values()],
      });
    }
    if (!this.privateAccess) this.publish({
      kind: 'status', profile_id: this.profileId, context_kind: 'private',
      previous_revision: next.private.revision, projection_revision: next.private.revision,
      event_sequence: this.eventSequence, effective_change: false, resync_required: true,
      resync_reason: this.privateUnavailableReason,
    });
    if (previousPrivateCount > 0 && next.private.records.size === 0 && next.private.windows.length === 0) {
      this.publish({
        kind: 'private_context_ended', profile_id: this.profileId, context_kind: 'private',
        projection_epoch: next.private.epoch!, previous_revision: parseRevision(0), projection_revision: next.private.revision,
        event_sequence: this.eventSequence, effective_change: true, resync_required: incomplete,
        ...(this.resyncReason === undefined ? {} : { resync_reason: this.resyncReason }),
      });
    }
  }

  private markUnavailable(reason: ResyncReason): void {
    this.status = 'unavailable';
    this.resyncRequired = true;
    this.resyncReason = reason;
    this.diagnostics.requiredCapabilityErrors += 1;
    for (const contextKind of ['normal', 'private'] as const) this.contexts[contextKind].status = 'unavailable';
    this.publish({
      kind: 'status', ...(this.profileId === undefined ? {} : { profile_id: this.profileId }),
      previous_revision: this.contexts.normal.revision, projection_revision: this.contexts.normal.revision,
      event_sequence: this.eventSequence, effective_change: false, resync_required: true, resync_reason: reason,
    });
  }

  private nextRevision(state: ContextProjection): ProjectionRevision | undefined {
    const revision = Number(state.revision);
    if (!Number.isSafeInteger(revision) || revision >= Number.MAX_SAFE_INTEGER) return undefined;
    return parseRevision(revision + 1);
  }

  private findContext(tabId: number): ContextKind | undefined {
    const matches = (['normal', 'private'] as const).filter(contextKind => this.contexts[contextKind].rawTabs.has(this.key(contextKind, tabId)));
    return matches.length === 1 ? matches[0] : undefined;
  }

  private key(contextKind: ContextKind, tabId: number): string {
    return this.profileId ? tabIdentityKey({ profile_id: this.profileId, context_kind: contextKind, tab_id: tabId }) : `${contextKind}:${tabId}`;
  }

  private publish(event: Omit<ObserverHandoff, 'diagnostics'>): void {
    const handoff: ObserverHandoff = { ...event, diagnostics: { ...this.diagnostics } };
    for (const listener of this.listeners) {
      try { listener(handoff); }
      catch { this.diagnostics.handoffListenerErrors += 1; }
    }
  }
}
