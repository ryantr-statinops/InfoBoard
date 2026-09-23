# Phase 02 — Domain identifiers và ownership

> Plan ID: IP-02
> Status: See README.md execution tracker
> Execution owner: domain-model / architecture agent
> Dependencies: IP-01
> Parallel boundary: IP-03 (independent after IP-01; no shared implementation files)
> Requirement IDs: None primary; supporting FR-003, FR-009, FR-010, FR-011, FR-014, FR-015, NFR-004, NFR-005, NFR-009
> Owned paths: `extension/domain/profile_id` (to-create), `extension/domain/tab_identity` (to-create), `extension/domain/tab_projection` (to-create), `extension/domain/activation` (to-create), `host/internal/domain/profile_id` (to-create), `host/internal/domain/tab_identity` (to-create), `host/internal/domain/tab_projection` (to-create), `host/internal/domain/activation` (to-create), `fixtures/domain/phase-02` (to-create), `tests/domain/phase-02` (to-create)

## 1. Mục tiêu

- Establish the domain contract that every later implementation can use for profile isolation, tab identity, eligible-tab records, projection revisions, activation references, and ownership boundaries.
- Make the complete lifecycle explicit: profile initialization, full snapshot, ordered changes, query references, activation confirmation, tab removal, reconnect, private-context disposal, reset, and uninstall.
- Choose identities and final tie-break keys that remain deterministic without using title, URL, array order, or another mutable field as an activation key.
- Define what the extension, Go Native Messaging host, SQLite repository, and search UI may read or mutate so later phases cannot silently turn a mirror or historical record into the source of truth.

## 2. Phạm vi

- Bao gồm:
  - An opaque `profile_id` contract, generation and recovery rules, profile/context isolation, and its use as a mandatory partition key.
  - A stable `tab_identity` contract and a runtime `projection_epoch` fence for browser tab-ID reuse and stale result protection.
  - The bounded `eligible_tab` record, including display/search forms, optional window/group context, private-context state, and field limits.
  - A profile-scoped `projection_state` with authoritative revision semantics, atomic full-snapshot replacement, ordered delta application, duplicate suppression, and resync behavior.
  - Activation result references and the bounded activation-metadata contract used only for local recency signals.
  - Ownership and allowed data flow between browser APIs/extension, host memory and index, SQLite, and UI.
  - Fixtures and test seams for profile isolation, identity collisions, event lifecycle, revision fencing, retention, and ownership violations.
- Ngoài phạm vi:
  - Browser event listener implementation, browser API adapters, or eligibility policy wiring; those consume this contract in IP-04.
  - Projection reconciliation transport, Native Messaging envelopes, and host reconnect orchestration; those are implemented in IP-05 through IP-07 and IP-09.
  - Lexical normalization, score weights, or ranking component implementation; IP-10 and IP-11 own those behaviors. This phase owns only the identity-based final ordering key.
  - Concrete SQLite migrations, repository queries, reset execution, or installer cleanup; IP-08, IP-15, and IP-18 own those operations against the records defined here.
  - Search-surface rendering or browser activation calls; IP-13 and IP-14 consume the references and guards defined here.
  - Any browser data source, remote service, or data retention class not named by the refactor contract.

## 3. Điều kiện tiên quyết

- IP-01 is merged and provides the requirement ownership map, logical roots, fixture conventions, bounded-value vocabulary, and global invariants.
- Read the binding [`product boundary`](../refactor/README.md), [`target architecture`](../refactor/architecture.md), [`domain and privacy contract`](../refactor/domain-and-privacy.md), [`persistence and lifecycle`](../refactor/persistence-and-lifecycle.md), [`runtime protocol`](../refactor/runtime-protocol.md), [`requirements`](../refactor/requirements.md), and [`verification and acceptance`](../refactor/verification-and-acceptance.md).
- Read [`CONTEXT.md`](../../../CONTEXT.md) and confirm that implementation roots remain absent. Every target in this plan is therefore a logical `to-create` path, not an observed source path.
- Use the current browser profile and private-context rules from [`browser-landscape.md`](../refactor/browser-landscape.md); do not infer a profile identity from an account, filesystem path, title, URL, or browser history.

## 4. Đầu ra cần bàn giao

- [ ] Domain type contracts under the owned `extension/domain/` and `host/internal/domain/` paths:
  - `ProfileID`: an opaque, cryptographically random, profile-scoped value with a bounded serialized form; it is never an email, browser account ID, path, title, URL, or user-facing label.
  - `ContextKind`: `normal` or `private`, with private state isolated from normal state and never durable after the private context ends.
  - `TabIdentity`: `(profile_id, context_kind, tab_id)`. Equality is strict across all three fields; a title, URL, domain, window label, or array position is never an identity fallback.
  - `ProjectionEpoch`: an opaque runtime generation associated with a live extension/host projection. A result reference carries the epoch and `projection_revision` so an old tab ID cannot be reused silently after restart or reconnect.
  - `ProjectionRevision`: a bounded unsigned sequence within one epoch. Revision `0` is the uninitialized state; the first authoritative snapshot is revision `1`; every accepted effective event advances by one.
  - `EligibleTabRecord` and `ProjectionState` as specified below, with explicit missing/invalid-field behavior.
  - `ActivationReference` and `ActivationMetadata` as specified below, with no query string, fragment, page content, cookie, token, or raw query history.
- [ ] A domain lifecycle implementation that can produce these observable states: `Uninitialized`, `SnapshotReady`, `DeltaApplying`, `Ready`, `ResyncRequired`, `StaleReference`, `PrivateContextEnded`, `Reset`, and `Uninstalled`.
- [ ] A written ownership matrix encoded in module boundaries and tests: the extension is authoritative for browser state; the host mirrors and indexes it; SQLite stores only its declared durable classes; UI carries references and presentation state.
- [ ] Phase fixtures under `fixtures/domain/phase-02/`:
  - `FX-PROFILE-ISOLATION`: equal browser tab IDs in two profiles never share projection, query, configuration, or activation metadata.
  - `FX-IDENTITY-TIEBREAK`: duplicate titles/domains receive deterministic ordering from identity fields, not input order or display text.
  - `FX-PROJECTION-LIFECYCLE`: snapshot plus create/update/move/group/pin/activate/remove operations yields the exact expected eligible ID set with no duplicates.
  - `FX-REVISION-FENCE`: old epoch/revision, a removed tab, a revision gap, and a conflicting duplicate event are rejected without mutation or activation.
  - `FX-ACTIVATION-RETENTION`: successful normal-context activation records are capped at 500 and 30 days; private and failed activations are not durable.
  - `FX-OWNERSHIP-BOUNDARY`: a host or UI attempt to mutate browser state, an SQLite attempt to provide live-tab truth, or a projection record with another profile is rejected.
- [ ] A test seam under `tests/domain/phase-02/` that consumes the fixture input/output schema from IP-01 and reports profile, identity, revision, lifecycle, ownership, and retention failures with bounded diagnostics.

### Domain records and invariants

`ProfileID` is generated once for a browser profile's extension installation and retained in the profile-scoped extension control state. The value is opaque and locally generated; it is not synchronized, uploaded, derived from an account, or shared with another profile. The host receives it during `hello` and keys every in-memory and durable profile record by it. If the control state is missing or malformed after initialization, the extension fails closed and offers repair/reset rather than silently deriving a replacement that could merge or orphan data. An explicit reset may delete the old identity and create a fresh one only after all old sessions are disconnected.

`TabIdentity` is the stable operational identity for a live projection entry: `profile_id`, `context_kind`, and the browser's numeric `tab_id`. The identity is valid only while the current profile projection accepts it. Because browser IDs may be reused after close or restart, a `TabReference` additionally carries `projection_epoch` and `projection_revision`; activation validates all of them and the current browser tab before acting. Historical activation metadata may retain the stable identity plus domain, but it is never an activation authority.

`EligibleTabRecord` contains:

```text
profile_id
context_kind
 tab_identity = { profile_id, context_kind, tab_id }
window_id
group_id (nullable when unsupported or ungrouped)
title_display
title_search
url_search (transient local field; never durable)
url_display (privacy-redacted by default)
domain_display
domain_search
window_label_display / window_label_search (optional)
group_label_display / group_label_search (optional)
pinned
active
eligible
observed_at
projection_epoch
projection_revision
```

The browser adapter creates records only from the current open-tab snapshot or a browser event. `eligible=true` requires a valid profile/context and tab identity plus bounded fields; invalid identity is an exclusion, never a synthetic fallback. Missing optional window/group labels degrade to an explicit absent value. Every record is partitioned by profile and context, and the host indexes only records accepted for the current epoch/revision. Full URLs may be used transiently for local matching, while query strings/fragments are not displayed by default and are never copied into SQLite activation metadata.

`ProjectionState` contains one profile/context partition, its current epoch, one revision, and an atomically replaceable map keyed by `TabIdentity`. A full snapshot is authoritative: it replaces the map as one operation and advances the revision once. A delta is accepted only for the current profile/context, current epoch, and expected predecessor revision/sequence; a valid effective event advances the revision once. Duplicate events with the same identity and sequence are idempotently acknowledged only when their content matches; conflicting duplicates, gaps, out-of-order events, and profile mismatches leave state unchanged and request a full snapshot. A remove event deletes exactly one identity. A full snapshot after reconnect starts a new epoch and invali…

`ActivationReference` returned to the UI is bounded and contains `profile_id`, `context_kind`, `tab_id`, `projection_epoch`, `projection_revision`, and the host's result/request identity. It does not contain enough information to activate a tab by itself. The extension must re-read/validate the current tab and window, profile, eligibility, epoch, and revision before activation. A missing tab, changed revision, changed epoch, or profile mismatch produces a stale/refresh outcome and never silently activates another result.

`ActivationMetadata` is written only after the extension observes a successful activation and sends an `activation_observed` event. Its logical fields are `profile_id`, `context_kind=normal`, `tab_identity`, normalized `domain`, `activated_at`, and a bounded source enum such as `keyboard_enter` or `surface_select`. It stores no title, full URL, query string, fragment, window/group label, page body, cookie, token, or user query. Persistence is optional: when disabled, the host may use session-only recency and must not write SQLite. Private-context activation is session-only and is removed when the private context ends.

Retention is enforced at write and startup/rebuild boundaries: delete activation rows older than 30 days, then keep at most the newest 500 by `(activated_at, storage_sequence)`; a count/age tie is resolved by the storage sequence, never by row traversal order. Reset removes all profile-owned configuration, activation metadata, projection cache, and generated host state; uninstall additionally removes the registered host and generated product files. Neither operation closes tabs, changes browser history, or touches unrelated profile data. Diagnostics have a separate bounded policy owned by IP-16 and must not be smuggled into activation records.

Final result ordering is defined for equal ranking outputs without claiming ranking ownership: sort by the ranker output first, then `context_kind` ordinal (`normal` before `private`), numeric `window_id` (absent is ineligible), numeric `group_id` with an explicit ungrouped sentinel, numeric `tab_id`, and finally the immutable serialized `TabIdentity` bytes. Never use input array order, mutable labels, or URL/title text as the identity tie-break. A duplicate `TabIdentity` is a projection error, not a tie to resolve.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`architecture-review`](../../../.agent/skills/personal/product/architecture-review/SKILL.md): map data flow, failure modes, operational ownership, and the simplest local component boundaries before implementation.
  - [`architecture-tradeoff`](../../../.agent/skills/personal/decision/architecture-tradeoff/SKILL.md): make the profile-ID, runtime-epoch, and browser-tab-ID reuse trade-off explicit and reversible.
  - [`databases`](../../../.agent/skills/personal/engineering/data/databases/SKILL.md): define activation/config access patterns, retention, migration inputs, and the rule that SQLite is not live-tab truth.
  - [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md): constrain identifiers and fields, fail closed on profile/revision mismatch, and prevent sensitive values from logs or durable records.
  - [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md): publish a standalone contract with links, limits, examples, and observable acceptance.
  - [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md): sequence domain work into implementation units with explicit interfaces and edge cases.
  - [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md): specify deterministic success, boundary, duplicate, stale, privacy, and retention fixtures.
  - [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md): keep this one-file phase branch, review its staged path, and publish the required commit.
- Tài liệu trong `docs/`:
  - [`README.md`](../refactor/README.md) for the binding product boundary and profile rules.
  - [`architecture.md`](../refactor/architecture.md) for component ownership and data flow.
  - [`domain-and-privacy.md`](../refactor/domain-and-privacy.md) for entity fields, lifecycle, trust boundaries, privacy/security invariants.
  - [`persistence-and-lifecycle.md`](../refactor/persistence-and-lifecycle.md) for SQLite ownership, retention, reset, restart, and corruption behavior.
  - [`runtime-protocol.md`](../refactor/runtime-protocol.md) for envelope identity, snapshot/delta revision semantics, activation validation, and typed errors.
  - [`requirements.md`](../refactor/requirements.md) and [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md) for supporting requirements and evidence seams.
- Quy ước code, ADR, context ngoài `docs/`: [`CONTEXT.md`](../../../CONTEXT.md); use bounded local data, explicit versions, profile partitions, deterministic behavior, and recoverable failure states. The extension and host paths below remain `to-create` until their implementation phases establish the toolchains.

## 6. Công việc triển khai

- [ ] `IP-02-T01` Add the extension domain value objects at `extension/domain/profile_id`, `extension/domain/tab_identity`, `extension/domain/tab_projection`, and `extension/domain/activation` (all `to-create`): parse bounded profile IDs, model context and tab identity, distinguish stable identity from epoch/revision references, and reject malformed/cross-profile values before indexing or activation.
- [ ] `IP-02-T02` Add matching host domain types at `host/internal/domain/profile_id`, `host/internal/domain/tab_identity`, `host/internal/domain/tab_projection`, and `host/internal/domain/activation` (all `to-create`): validate the extension contract without assigning browser IDs or trusting host-generated substitutes.
- [ ] `IP-02-T03` Implement the profile-control lifecycle: generate one opaque profile ID on first install, retain it in profile-scoped extension control state, carry it through hello/snapshot/delta/query/activation messages, fail closed on loss or mismatch, and allow identity replacement only as part of an explicit reset after session shutdown.
- [ ] `IP-02-T04` Implement the eligibility boundary consumed by IP-04: validate profile/context/tab/window identity, bound all text and URL-derived fields before allocation, keep optional group/label fields explicitly absent, and emit exclusion reasons only as redacted counts or bounded error classes.
- [ ] `IP-02-T05` Implement projection transitions: initialize revision `0`, atomically accept an authoritative revision-`1` snapshot, apply only expected ordered events, suppress matching duplicates, preserve state on conflicts/gaps, and produce `SNAPSHOT_REQUIRED`/`REVISION_MISMATCH` outcomes without returning partial indexes.
- [ ] `IP-02-T06` Add runtime epoch fencing and result references: create a new epoch for a fresh runtime/snapshot lineage, invalidate older references on reconnect or replacement, and ensure UI references contain identity plus epoch/revision rather than mutable title/URL data.
- [ ] `IP-02-T07` Define the ownership seams consumed by later phases: extension writes authoritative projection and invokes browser activation; host validates, mirrors, indexes, ranks, and emits status; SQLite repository stores only profile-scoped configuration and bounded successful activation metadata; UI renders bounded fields and submits references but cannot mutate projection or call browser APIs directly.
- [ ] `IP-02-T08` Add the activation-observed path and retention hooks: write only confirmed normal-context activations when the user setting permits, purge by age then count deterministically, keep private/failed activations session-only, and expose reset/uninstall deletion hooks without deleting browser state.
- [ ] `IP-02-T09` Add the phase fixtures and test seam under `fixtures/domain/phase-02` and `tests/domain/phase-02`: assert exact map contents, identity keys, epoch/revision transitions, cross-profile rejection, stale activation outcomes, tie-break order, retention counts/ages, and ownership violations. Do not assert field-copy plumbing or internal function names.
- [ ] `IP-02-T10` Record all cross-phase handoffs in the implementation index: IP-04 owns browser event/eligibility wiring, IP-05 owns reconciliation, IP-07 owns wire validation/error literals, IP-08 owns schema/migrations, IP-11 owns score components, IP-14 owns browser activation execution, and IP-15/IP-18 own reset/uninstall execution. This phase remains the domain contract owner.

## 7. Kế hoạch commit

1. `feat(domain): implement ip-02-t01`
   - Task IDs: `IP-02-T01`.
   - Owned target paths: extension/domain/profile_id, extension/domain/tab_identity, extension/domain/tab_projection, extension/domain/activation.
   - Behavior: Add the extension domain value objects at `extension/domain/profile_id`, `extension/domain/tab_identity`, `extension/domain/tab_projection`, and `extension/domain/activation` (all `to-create`): parse bounded profile IDs, model context and tab identity, distinguish stable identity from epoch/revision references, and reject malformed/cross-profile values before indexing or activation.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/domain/phase-02/run_fixtures.py --suite phase-02`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Add the extension domain value objects at `extension/domain/profile_id`, `extension/domain/tab_identity`, `extension/domain/tab_projection`, and `extension/domain/activation` (all `to-create`): parse bounded profile IDs, model context and tab identity, distinguish stable identity from epoch/revision references, and reject malformed/cross-profile values before indexing or activation.
   - Dependency gate: all index.md dependencies for IP-02 have merged to dev; phase work branch starts from latest origin/dev.

2. `feat(domain): implement ip-02-t02`
   - Task IDs: `IP-02-T02`.
   - Owned target paths: host/internal/domain/profile_id, host/internal/domain/tab_identity, host/internal/domain/tab_projection, host/internal/domain/activation.
   - Behavior: Add matching host domain types at `host/internal/domain/profile_id`, `host/internal/domain/tab_identity`, `host/internal/domain/tab_projection`, and `host/internal/domain/activation` (all `to-create`): validate the extension contract without assigning browser IDs or trusting host-generated substitutes.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/domain/phase-02/run_fixtures.py --suite phase-02`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Add matching host domain types at `host/internal/domain/profile_id`, `host/internal/domain/tab_identity`, `host/internal/domain/tab_projection`, and `host/internal/domain/activation` (all `to-create`): validate the extension contract without assigning browser IDs or trusting host-generated substitutes.
   - Dependency gate: all index.md dependencies for IP-02 have merged to dev; phase work branch starts from latest origin/dev.

3. `feat(domain): implement ip-02-t03`
   - Task IDs: `IP-02-T03`.
   - Owned target paths: `extension/domain/profile_id` (to-create), `extension/domain/tab_identity` (to-create), `extension/domain/tab_projection` (to-create), `extension/domain/activation` (to-create), `host/internal/domain/profile_id` (to-create), `host/internal/domain/tab_identity` (to-create), `host/internal/domain/tab_projection` (to-create), `host/internal/domain/activation` (to-create), `fixtures/domain/phase-02` (to-create), `tests/domain/phase-02` (to-create).
   - Behavior: Implement the profile-control lifecycle: generate one opaque profile ID on first install, retain it in profile-scoped extension control state, carry it through hello/snapshot/delta/query/activation messages, fail closed on loss or mismatch, and allow identity replacement only as part of an explicit reset after session shutdown.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/domain/phase-02/run_fixtures.py --suite phase-02`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Implement the profile-control lifecycle: generate one opaque profile ID on first install, retain it in profile-scoped extension control state, carry it through hello/snapshot/delta/query/activation messages, fail closed on loss or mismatch, and allow identity replacement only as part of an explicit reset after session shutdown.
   - Dependency gate: all index.md dependencies for IP-02 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(domain): implement ip-02-t04`
   - Task IDs: `IP-02-T04`.
   - Owned target paths: `extension/domain/profile_id` (to-create), `extension/domain/tab_identity` (to-create), `extension/domain/tab_projection` (to-create), `extension/domain/activation` (to-create), `host/internal/domain/profile_id` (to-create), `host/internal/domain/tab_identity` (to-create), `host/internal/domain/tab_projection` (to-create), `host/internal/domain/activation` (to-create), `fixtures/domain/phase-02` (to-create), `tests/domain/phase-02` (to-create).
   - Behavior: Implement the eligibility boundary consumed by IP-04: validate profile/context/tab/window identity, bound all text and URL-derived fields before allocation, keep optional group/label fields explicitly absent, and emit exclusion reasons only as redacted counts or bounded error classes.
   - Fixture and command: IP-04; run `python3 tests/domain/phase-02/run_fixtures.py --suite phase-02`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Implement the eligibility boundary consumed by IP-04: validate profile/context/tab/window identity, bound all text and URL-derived fields before allocation, keep optional group/label fields explicitly absent, and emit exclusion reasons only as redacted counts or bounded error classes.
   - Dependency gate: all index.md dependencies for IP-02 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(domain): implement ip-02-t05`
   - Task IDs: `IP-02-T05`.
   - Owned target paths: `extension/domain/profile_id` (to-create), `extension/domain/tab_identity` (to-create), `extension/domain/tab_projection` (to-create), `extension/domain/activation` (to-create), `host/internal/domain/profile_id` (to-create), `host/internal/domain/tab_identity` (to-create), `host/internal/domain/tab_projection` (to-create), `host/internal/domain/activation` (to-create), `fixtures/domain/phase-02` (to-create), `tests/domain/phase-02` (to-create).
   - Behavior: Implement projection transitions: initialize revision `0`, atomically accept an authoritative revision-`1` snapshot, apply only expected ordered events, suppress matching duplicates, preserve state on conflicts/gaps, and produce `SNAPSHOT_REQUIRED`/`REVISION_MISMATCH` outcomes without returning partial indexes.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/domain/phase-02/run_fixtures.py --suite phase-02`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Implement projection transitions: initialize revision `0`, atomically accept an authoritative revision-`1` snapshot, apply only expected ordered events, suppress matching duplicates, preserve state on conflicts/gaps, and produce `SNAPSHOT_REQUIRED`/`REVISION_MISMATCH` outcomes without returning partial indexes.
   - Dependency gate: all index.md dependencies for IP-02 have merged to dev; phase work branch starts from latest origin/dev.

6. `feat(domain): implement ip-02-t06`
   - Task IDs: `IP-02-T06`.
   - Owned target paths: `extension/domain/profile_id` (to-create), `extension/domain/tab_identity` (to-create), `extension/domain/tab_projection` (to-create), `extension/domain/activation` (to-create), `host/internal/domain/profile_id` (to-create), `host/internal/domain/tab_identity` (to-create), `host/internal/domain/tab_projection` (to-create), `host/internal/domain/activation` (to-create), `fixtures/domain/phase-02` (to-create), `tests/domain/phase-02` (to-create).
   - Behavior: Add runtime epoch fencing and result references: create a new epoch for a fresh runtime/snapshot lineage, invalidate older references on reconnect or replacement, and ensure UI references contain identity plus epoch/revision rather than mutable title/URL data.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/domain/phase-02/run_fixtures.py --suite phase-02`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Add runtime epoch fencing and result references: create a new epoch for a fresh runtime/snapshot lineage, invalidate older references on reconnect or replacement, and ensure UI references contain identity plus epoch/revision rather than mutable title/URL data.
   - Dependency gate: all index.md dependencies for IP-02 have merged to dev; phase work branch starts from latest origin/dev.

7. `feat(domain): implement ip-02-t07`
   - Task IDs: `IP-02-T07`.
   - Owned target paths: `extension/domain/profile_id` (to-create), `extension/domain/tab_identity` (to-create), `extension/domain/tab_projection` (to-create), `extension/domain/activation` (to-create), `host/internal/domain/profile_id` (to-create), `host/internal/domain/tab_identity` (to-create), `host/internal/domain/tab_projection` (to-create), `host/internal/domain/activation` (to-create), `fixtures/domain/phase-02` (to-create), `tests/domain/phase-02` (to-create).
   - Behavior: Define the ownership seams consumed by later phases: extension writes authoritative projection and invokes browser activation; host validates, mirrors, indexes, ranks, and emits status; SQLite repository stores only profile-scoped configuration and bounded successful activation metadata; UI renders bounded fields and submits references but cannot mutate projection or call browser APIs directly.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/domain/phase-02/run_fixtures.py --suite phase-02`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Define the ownership seams consumed by later phases: extension writes authoritative projection and invokes browser activation; host validates, mirrors, indexes, ranks, and emits status; SQLite repository stores only profile-scoped configuration and bounded successful activation metadata; UI renders bounded fields and submits references but cannot mutate projection or call browser APIs directly.
   - Dependency gate: all index.md dependencies for IP-02 have merged to dev; phase work branch starts from latest origin/dev.

8. `feat(domain): implement ip-02-t08`
   - Task IDs: `IP-02-T08`.
   - Owned target paths: `extension/domain/profile_id` (to-create), `extension/domain/tab_identity` (to-create), `extension/domain/tab_projection` (to-create), `extension/domain/activation` (to-create), `host/internal/domain/profile_id` (to-create), `host/internal/domain/tab_identity` (to-create), `host/internal/domain/tab_projection` (to-create), `host/internal/domain/activation` (to-create), `fixtures/domain/phase-02` (to-create), `tests/domain/phase-02` (to-create).
   - Behavior: Add the activation-observed path and retention hooks: write only confirmed normal-context activations when the user setting permits, purge by age then count deterministically, keep private/failed activations session-only, and expose reset/uninstall deletion hooks without deleting browser state.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/domain/phase-02/run_fixtures.py --suite phase-02`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Add the activation-observed path and retention hooks: write only confirmed normal-context activations when the user setting permits, purge by age then count deterministically, keep private/failed activations session-only, and expose reset/uninstall deletion hooks without deleting browser state.
   - Dependency gate: all index.md dependencies for IP-02 have merged to dev; phase work branch starts from latest origin/dev.

9. `test(domain): implement ip-02-t09`
   - Task IDs: `IP-02-T09`.
   - Owned target paths: fixtures/domain/phase-02, tests/domain/phase-02.
   - Behavior: Add the phase fixtures and test seam under `fixtures/domain/phase-02` and `tests/domain/phase-02`: assert exact map contents, identity keys, epoch/revision transitions, cross-profile rejection, stale activation outcomes, tie-break order, retention counts/ages, and ownership violations. Do not assert field-copy plumbing or internal function names.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/domain/phase-02/run_fixtures.py --suite phase-02`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Add the phase fixtures and test seam under `fixtures/domain/phase-02` and `tests/domain/phase-02`: assert exact map contents, identity keys, epoch/revision transitions, cross-profile rejection, stale activation outcomes, tie-break order, retention counts/ages, and ownership violations. Do not assert field-copy plumbing or internal function names.
   - Dependency gate: all index.md dependencies for IP-02 have merged to dev; phase work branch starts from latest origin/dev.

10. `feat(domain): implement ip-02-t10`
   - Task IDs: `IP-02-T10`.
   - Owned target paths: `extension/domain/profile_id` (to-create), `extension/domain/tab_identity` (to-create), `extension/domain/tab_projection` (to-create), `extension/domain/activation` (to-create), `host/internal/domain/profile_id` (to-create), `host/internal/domain/tab_identity` (to-create), `host/internal/domain/tab_projection` (to-create), `host/internal/domain/activation` (to-create), `fixtures/domain/phase-02` (to-create), `tests/domain/phase-02` (to-create).
   - Behavior: Record all cross-phase handoffs in the implementation index: IP-04 owns browser event/eligibility wiring, IP-05 owns reconciliation, IP-07 owns wire validation/error literals, IP-08 owns schema/migrations, IP-11 owns score components, IP-14 owns browser activation execution, and IP-15/IP-18 own reset/uninstall execution. This phase remains the domain contract owner.
   - Fixture and command: IP-04, IP-05, IP-07, IP-08, IP-11, IP-14, IP-15, IP-18; run `python3 tests/domain/phase-02/run_fixtures.py --suite phase-02`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Record all cross-phase handoffs in the implementation index: IP-04 owns browser event/eligibility wiring, IP-05 owns reconciliation, IP-07 owns wire validation/error literals, IP-08 owns schema/migrations, IP-11 owns score components, IP-14 owns browser activation execution, and IP-15/IP-18 own reset/uninstall execution. This phase remains the domain contract owner.
   - Dependency gate: all index.md dependencies for IP-02 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] Future fixture runner (to-create at `tests/domain/phase-02/run_fixtures.py`) is invoked from repository root as `python3 tests/domain/phase-02/run_fixtures.py --suite phase-02`; it must emit a bounded pass/fail record for every fixture ID in `fixtures/domain/phase-02/` and no raw title, URL, query, token, or private record.
- [ ] Future host domain checks (to-create Go module under `host/`) run as `go test ./host/... -run 'Test(Profile|TabIdentity|Projection|Activation)'`; they must cover malformed/cross-profile identity, full-snapshot authority, ordered deltas, duplicate/conflict behavior, epoch/revision fences, and retention boundaries.
- [ ] `FX-PROFILE-ISOLATION` proves two profiles with the same numeric tab/window IDs produce disjoint projection maps, result references, config partitions, and activation rows; a profile mismatch is rejected with no state mutation.
- [ ] `FX-PROJECTION-LIFECYCLE` proves the exact sequence snapshot → create → title/URL update → move/group/pin/active update → remove converges to the browser's eligible IDs with no duplicate identity and with one revision advance per accepted operation.
- [ ] `FX-REVISION-FENCE` proves stale epoch/revision, removed tab, missing required identity, gap, out-of-order delta, and conflicting duplicate cannot return a result or activate a different tab; the state remains unchanged and a bounded resync/stale outcome is observable.
- [ ] `FX-IDENTITY-TIEBREAK` proves repeated runs with identical projection/query/timestamp inputs produce byte-equivalent ordered identities and scores even when titles/domains and all visible context are duplicated. This supports NFR-004 without claiming ranking ownership.
- [ ] `FX-ACTIVATION-RETENTION` proves successful normal activations retain no more than 500 records and no record older than 30 days, with deterministic oldest-first eviction; private and failed activations are absent from SQLite, and disabled persistence leaves only session-local recency.
- [ ] `FX-OWNERSHIP-BOUNDARY` proves the extension is the only browser authority, the host cannot execute or mutate a tab, SQLite cannot answer for a missing live tab, and UI references cannot bypass current profile/epoch/revision validation. Reset/uninstall removes owned records only and leaves tabs and unrelated browser data unchanged.
- [ ] Inspect the generated fixture report and code review for NFR-005: after every authoritative full snapshot, indexed `TabIdentity` values equal the eligible browser IDs exactly; no stale/removed/private record remains in the active index.
- [ ] Inspect permissions, storage writes, logs, and outbound requests for NFR-009: the phase creates no new data source, no network path, no page-content read, no credential/session collection, and no raw sensitive value in diagnostics.
- [ ] At authoring time the implementation roots do not exist, so the commands above are future acceptance contracts rather than claims of executed runtime tests. Current documentation proof is `git diff --check` plus the targeted link/heading checks reported with this phase commit.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: browser tab IDs can be reused after removal or restart. Phương án xử lý đã chọn: keep strict `(profile_id, context_kind, tab_id)` identity, fence every result with a runtime `projection_epoch` and `projection_revision`, and revalidate the live browser tab before activation.
- Rủi ro: an extension service-worker restart can lose in-memory projection state. Phương án xử lý đã chọn: obtain a new authoritative full snapshot in a new epoch before `Ready`; never treat SQLite or a stale host mirror as live-tab truth.
- Rủi ro: a profile identity can be missing, corrupted, or accidentally regenerated. Phương án xử lý đã chọn: fail closed and require repair/reset; only explicit reset may replace the opaque identity after disconnecting old sessions.
- Rủi ro: private-window permission may be unavailable or may end without a reliable cleanup event. Phương án xử lý đã chọn: keep private context as a separate in-memory partition, expose unavailable state, and make private persistence impossible by type/validation rather than relying only on cleanup callbacks.
- Rủi ro: duplicate, delayed, or out-of-order browser events can create phantom records. Phương án xử lý đã chọn: snapshot authority, expected predecessor checks, idempotent matching duplicates, and no mutation on conflicts or gaps.
- Rủi ro: retention policy could be weakened by a ranking-quality optimization. Phương án xử lý đã chọn: enforce age/count limits at write and rebuild, keep persistence opt-in, and keep diagnostics on its own bounded policy.
- Rủi ro: sibling phases may accidentally claim a domain path or source-of-truth role. Phương án xử lý đã chọn: this phase owns the domain contract; event/reconciliation, storage, ranking, activation execution, reset, and packaging phases consume it and must not redefine identity semantics.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: none. A change to profile identity, private-context handling, durable fields, or activation authority requires an update to the canonical refactor decision log before implementation.

## 10. References ngoài `docs/`

- Skill: [`architecture-review`](../../../.agent/skills/personal/product/architecture-review/SKILL.md), [`architecture-tradeoff`](../../../.agent/skills/personal/decision/architecture-tradeoff/SKILL.md), [`databases`](../../../.agent/skills/personal/engineering/data/databases/SKILL.md), [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md), [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md), [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md), [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md), [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `extension/domain/profile_id` (to-create), `extension/domain/tab_identity` (to-create), `extension/domain/tab_projection` (to-create), `extension/domain/activation` (to-create), `host/internal/domain/profile_id` (to-create), `host/internal/domain/tab_identity` (to-create), `host/internal/domain/tab_projection` (to-create), `host/internal/domain/activation` (to-create), `tests/domain/phase-02` (to-create)
- Fixture/tool/artifact ngoài `docs/`: `fixtures/domain/phase-02/FX-PROFILE-ISOLATION`, `FX-IDENTITY-TIEBREAK`, `FX-PROJECTION-LIFECYCLE`, `FX-REVISION-FENCE`, `FX-ACTIVATION-RETENTION`, and `FX-OWNERSHIP-BOUNDARY` (all to-create); `tests/domain/phase-02/run_fixtures.py` (to-create)