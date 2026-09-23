# Phase 01 — Contract và fixtures

> Plan ID: IP-01
> Status: See README.md execution tracker
> Execution owner: coordinator / Codex
> Dependencies: Canonical refactor documents
> Parallel boundary: None; this phase establishes shared contracts
> Requirement IDs: FR-001..FR-015, NFR-001..NFR-010, operational rules
> Owned paths: `fixtures/catalog.schema.json`, `fixtures/catalog.json`, `tests/implementation/validate_corpus.py`; phase-specific `fixtures/` and `tests/` roots remain `to-create`; `extension/`, `host/`, and `packaging/` remain `to-create`

## 1. Mục tiêu

- Turn the current product contract into an executable implementation map with one primary owner for every FR/NFR requirement.
- Establish fixture IDs, observable outcomes, logical repository roots, naming rules, and verification seams that later phases can consume without inventing incompatible contracts.
- Make the plan corpus auditable from a clean checkout before implementation source exists.

## 2. Phạm vi

- Bao gồm:
  - Traceability for FR-001 through FR-015, NFR-001 through NFR-010, and operational requirements.
  - Fixture inventory for shortcut/focus, query, ranking, browser events, reconnect, stale result, host unavailable, reset, and permission failures.
  - Logical target layout under `extension/`, `host/`, `packaging/`, `fixtures/`, and `tests/`.
  - Shared commands, test seams, metadata conventions, error vocabulary, revision vocabulary, and phase ownership rules.
- Ngoài phạm vi:
  - Implementing runtime code, browser manifests, host binaries, installers, or UI.
  - Choosing a package path that is not present in the scaffold; paths remain `to-create` until observed.
  - Adding browser data sources or remote services outside the canonical contract.

## 3. Điều kiện tiên quyết

- Read [`docs/plan/refactor/README.md`](../refactor/README.md), [`architecture.md`](../refactor/architecture.md), [`requirements.md`](../refactor/requirements.md), [`runtime-protocol.md`](../refactor/runtime-protocol.md), and [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md).
- Read [`CONTEXT.md`](../../../CONTEXT.md) and the repository onboarding and requirements-analysis skills.
- Confirm runtime source roots `extension/`, `host/`, and `packaging/` are still absent; replace only those logical `to-create` paths if a scaffold appears. The shared fixture catalog and corpus validator are observed IP-01 artifacts.

## 4. Đầu ra cần bàn giao

- [x] A requirement matrix in `index.md` with exactly one primary phase owner for every FR and NFR ID.
- [x] A fixture catalog under `fixtures/` with stable IDs and expected observable results:
  - `FX-SHORTCUT-FOCUS`: configured command opens the focused surface and focuses the input.
  - `FX-QUERY-LIVE`: each accepted query revision renders the matching result set in order.
  - `FX-RANKING-DETERMINISTIC`: identical projection/query inputs produce identical ordered output and explanations.
  - `FX-TAB-EVENTS`: create/update/move/group/pin/activate/remove events converge without duplicates.
  - `FX-RECONNECT`: disconnect and reconnect trigger the required snapshot sequence.
  - `FX-STALE-ACTIVATION`: an old result cannot activate a different tab.
  - `FX-HOST-DOWN`: unavailable host yields a visible bounded error and retry state.
  - `FX-RESET-OWNED-DATA`: reset removes only InfoBoard-owned data.
  - `FX-PERMISSION-DENIED`: unsupported or denied browser access fails closed and is diagnosable.
- [x] Logical module map for all five target roots; observed catalog and validator paths are listed, and absent runtime/phase-owned paths remain `to-create`.
- [x] Shared acceptance vocabulary for profile IDs, tab IDs, projection revisions, protocol envelopes, health states, and bounded limits is linked to the canonical contract and catalog invariants.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`requirements-analysis`](../../../.agent/skills/common/foundation/requirements-analysis/SKILL.md): derive testable requirements and ownership.
  - [`repository-onboarding`](../../../.agent/skills/common/foundation/repository-onboarding/SKILL.md): establish repository and path assumptions.
  - [`project-scoping`](../../../.agent/skills/personal/product/project-scoping/SKILL.md): keep the product boundary narrow.
  - [`scope-control`](../../../.agent/skills/personal/decision/scope-control/SKILL.md): reject unowned or non-contract work.
  - [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md): define executable phase work.
  - [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md): maintain the standalone plan corpus.
  - [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md): define observable fixture outcomes.
  - [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md): preserve one-file phase commits and clean merges.
- Tài liệu trong `docs/`: canonical refactor README, architecture, requirements, runtime protocol, persistence/lifecycle, search/ranking, privacy, packaging, verification, roadmap, and decisions.
- Quy ước code, ADR, context ngoài `docs/`: [`CONTEXT.md`](../../../CONTEXT.md); future target roots are `extension/`, `host/`, `packaging/`, `fixtures/`, and `tests/`.

## 6. Công việc triển khai

- [x] `IP-01-T01` Build the requirement matrix from the canonical requirements document. Record the exact primary owner in `index.md`; supporting plans must cite the ID without claiming ownership.
- [x] `IP-01-T02` Define the logical target map: extension browser boundary and UI; Go host lifecycle, protocol, projection, index, persistence and diagnostics; packaging; fixtures; tests. Mark all absent paths `to-create`.
- [x] `IP-01-T03` Define fixture input/output schemas for focused activation, live query, rank ordering, event convergence, full snapshot, reconnect, stale activation, host failure, reset, and permission denial.
- [x] `IP-01-T04` Define future formatting/linting, unit/integration/browser, accessibility, performance, and package/install checks without selecting commands until their source toolchains exist. Implement the stdlib-only Python corpus validator at `tests/implementation/validate_corpus.py` and use its exact root command.
- [x] `IP-01-T05` Define global invariants: current profile only, monotonic projection revisions, deterministic output, no raw title/URL diagnostics, bounded payloads/timeouts, fail-closed activation, and ownership-safe reset.
- [x] `IP-01-T06` Record acceptance seams so each later phase can point to one fixture, one observable output, and one requirement ID.

## 7. Kế hoạch commit

1. `test(implementation): add IP-01 corpus validator`
   - Task IDs: `IP-01-T01`.
   - Owned target paths: `tests/implementation/validate_corpus.py`.
   - Behavior: validate the canonical 25-row FR/NFR owner matrix and the 20-phase plan corpus; reject missing/duplicate owners, task IDs without exactly one commit owner, invalid tracker states, missing phase rows, broken local links, and catalog/seam inconsistencies.
   - Fixture and command: owner-matrix and in-memory negative probes; run `python3 tests/implementation/validate_corpus.py` from repository root.
   - Observable result before commit: exit 0 with counts for phases, task mappings, requirement IDs, fixtures, invariants, and seams; malformed corpus mutations return validation errors and nonzero status.
   - Dependency gate: canonical refactor requirements and the implementation index are present; this branch is based on latest `origin/dev`.

2. `docs(implementation): implement ip-01-t02`
   - Task IDs: `IP-01-T02`.
   - Owned target paths: `docs/plan/implementation/index.md`, `docs/plan/implementation/phase-01-contract-and-fixtures.md`.
   - Behavior: record the observed shared fixture/catalog and validator paths; keep absent extension, Go host, packaging, and phase-specific test paths marked `to-create`.
   - Fixture and command: logical target-map assertions; run `python3 tests/implementation/validate_corpus.py`.
   - Observable result before commit: every target is either an observed path or explicitly `to-create`; the index and phase metadata agree.
   - Dependency gate: canonical refactor requirements are available; runtime source roots remain absent.

3. `docs(implementation): implement ip-01-t03`
   - Task IDs: `IP-01-T03`.
   - Owned target paths: `fixtures/catalog.schema.json`, `fixtures/catalog.json`.
   - Behavior: define the versioned fixture envelope and nine synthetic cases for shortcut focus, live query, deterministic ranking, event convergence, reconnect, stale activation, host unavailable, owned-data reset, and denied permission.
   - Fixture and command: `FX-SHORTCUT-FOCUS`, `FX-QUERY-LIVE`, `FX-RANKING-DETERMINISTIC`, `FX-TAB-EVENTS`, `FX-RECONNECT`, `FX-STALE-ACTIVATION`, `FX-HOST-DOWN`, `FX-RESET-OWNED-DATA`, `FX-PERMISSION-DENIED`; run `python3 tests/implementation/validate_corpus.py`.
   - Observable result before commit: nine unique IDs have input, requirement references, expected observable outputs, and privacy assertions; catalog conforms to its declared schema contract.
   - Dependency gate: IP-01-T02 target map is established; runtime test harnesses remain future phase work.

4. `docs(implementation): implement ip-01-t04`
   - Task IDs: `IP-01-T04`.
   - Owned target paths: `fixtures/catalog.json`, `fixtures/catalog.schema.json`.
   - Behavior: record future verification contracts for language-specific formatting/linting, unit/integration, browser matrix, accessibility, performance, package smoke, and corpus checks without fabricating unavailable runtime commands.
   - Fixture and command: `CHECK-FORMAT-LINT`, `CHECK-UNIT`, `CHECK-INTEGRATION`, `CHECK-BROWSER-MATRIX`, `CHECK-ACCESSIBILITY`, `CHECK-PERFORMANCE`, `CHECK-PACKAGE-SMOKE`, `CHECK-CORPUS`; run `python3 tests/implementation/validate_corpus.py`.
   - Observable result before commit: each future check names its selection rule and pass signal; the corpus validator has a runnable exact command.
   - Dependency gate: shared catalog schema exists; runtime commands remain unselected until their owning toolchains exist.

5. `docs(implementation): implement ip-01-t05`
   - Task IDs: `IP-01-T05`.
   - Owned target paths: `fixtures/catalog.json`, `fixtures/catalog.schema.json`.
   - Behavior: define profile isolation, revision safety, deterministic ranking, diagnostics redaction, bounded inputs, fail-closed activation, and ownership-safe reset invariants.
   - Fixture and command: `INV-001` through `INV-007`; run `python3 tests/implementation/validate_corpus.py`.
   - Observable result before commit: every invariant has an observable guard and the validator checks the invariant catalog shape and IDs.
   - Dependency gate: fixture envelope and base cases exist; detailed runtime assertions remain owned by dependent phases.

6. `docs(implementation): implement ip-01-t06`
   - Task IDs: `IP-01-T06`.
   - Owned target paths: `fixtures/catalog.json`, `fixtures/catalog.schema.json`, `docs/plan/implementation/index.md`, `docs/plan/implementation/README.md`.
   - Behavior: connect each later phase IP-02..IP-20 to a shared fixture, observed signal, requirement, ownership relation, and phase-specific extension note.
   - Fixture and command: `phase_seams` entries for IP-02..IP-20; run `python3 tests/implementation/validate_corpus.py`.
   - Observable result before commit: all 19 later phases have exactly one shared acceptance seam and no seam claims primary ownership contrary to the index.
   - Dependency gate: the owner matrix and shared fixture IDs are established; the coordinator owns tracker updates.
## 8. Kiểm chứng và nghiệm thu

- [x] Run the future corpus validator from repository root against `docs/plan/implementation/`; it must find one primary owner for all 25 FR/NFR IDs, unique Plan IDs, valid links, and all ten section headings.
- [x] Run fixture-schema validation once `fixtures/` exists; each fixture must specify input, expected observable output, failure class where relevant, and owning requirement.
- [x] Manually trace `FX-SHORTCUT-FOCUS`, `FX-QUERY-LIVE`, `FX-RANKING-DETERMINISTIC`, `FX-TAB-EVENTS`, `FX-RECONNECT`, `FX-STALE-ACTIVATION`, `FX-HOST-DOWN`, `FX-RESET-OWNED-DATA`, and `FX-PERMISSION-DENIED` to exactly one primary phase.
- [x] Confirm no target path is presented as an observed source path while the implementation tree is absent.
- [x] Acceptance signal: every FR/NFR and operational rule has a named owner, fixture or evidence seam, and later-phase dependency path.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: implementation scaffolding may appear after this plan is authored. Phương án xử lý đã chọn: reread the scaffold and replace only affected `to-create` paths before the affected phase commits.
- Rủi ro: two plans may later claim one module. Phương án xử lý đã chọn: keep one contract owner in `index.md`, serialize the dependent phase, and reject duplicate ownership.
- Rủi ro: a future toolchain may use different command names. Phương án xử lý đã chọn: require exact commands in the implementation phase that creates the toolchain; this phase records the acceptance seam, not a fabricated command.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: none; all current product decisions are binding in the canonical refactor documents.

## 10. References ngoài `docs/`

- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md), [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md), [`requirements-analysis`](../../../.agent/skills/common/foundation/requirements-analysis/SKILL.md), [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `fixtures/catalog.schema.json`, `fixtures/catalog.json`, `tests/implementation/validate_corpus.py` (observed); `extension/`, `host/`, `packaging/`, and phase-specific fixture/test roots remain `to-create`
- Fixture/tool/artifact ngoài `docs/`: shared fixture catalog/schema at `fixtures/catalog.json` and `fixtures/catalog.schema.json`; run `python3 tests/implementation/validate_corpus.py` from repository root
