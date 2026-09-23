# Phase 01 — Contract và fixtures

> Plan ID: IP-01
> Status: not_started
> Execution owner: control owner / contract-plan agent
> Dependencies: Canonical refactor documents
> Parallel boundary: None; this phase establishes shared contracts
> Requirement IDs: FR-001..FR-015, NFR-001..NFR-010, operational rules
> Owned paths: `fixtures/` (to-create), `tests/` (to-create), `extension/` (to-create), `host/` (to-create), `packaging/` (to-create)

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
- Confirm the repository has no implementation source tree; if one appears, replace every logical path below with the observed path before merging this plan.

## 4. Đầu ra cần bàn giao

- [ ] A requirement matrix in `index.md` with exactly one primary phase owner for every FR and NFR ID.
- [ ] A fixture catalog under `fixtures/` with stable IDs and expected observable results:
  - `FX-SHORTCUT-FOCUS`: configured command opens the focused surface and focuses the input.
  - `FX-QUERY-LIVE`: each accepted query revision renders the matching result set in order.
  - `FX-RANKING-DETERMINISTIC`: identical projection/query inputs produce identical ordered output and explanations.
  - `FX-TAB-EVENTS`: create/update/move/group/pin/activate/remove events converge without duplicates.
  - `FX-RECONNECT`: disconnect and reconnect trigger the required snapshot sequence.
  - `FX-STALE-ACTIVATION`: an old result cannot activate a different tab.
  - `FX-HOST-DOWN`: unavailable host yields a visible bounded error and retry state.
  - `FX-RESET-OWNED-DATA`: reset removes only InfoBoard-owned data.
  - `FX-PERMISSION-DENIED`: unsupported or denied browser access fails closed and is diagnosable.
- [ ] Exact logical module map for the five target roots, with each path marked `to-create` until source exists.
- [ ] Shared acceptance vocabulary for profile IDs, tab IDs, projection revisions, protocol envelopes, health states, and bounded limits.

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

- [ ] Build the requirement matrix from the canonical requirements document. Record the exact primary owner in `index.md`; supporting plans must cite the ID without claiming ownership.
- [ ] Define the logical target map: extension browser boundary and UI; Go host lifecycle, protocol, projection, index, persistence and diagnostics; packaging; fixtures; tests. Mark all absent paths `to-create`.
- [ ] Define fixture input/output schemas for focused activation, live query, rank ordering, event convergence, full snapshot, reconnect, stale activation, host failure, reset, and permission denial.
- [ ] Define global commands as future contracts: formatting/linting per language, unit/integration/browser checks, accessibility audit, performance benchmark, package/install smoke test, and the Python corpus validator. Do not invent a command before its source toolchain exists.
- [ ] Define global invariants: current profile only, monotonic projection revisions, deterministic output, no raw title/URL diagnostics, bounded payloads/timeouts, fail-closed activation, and ownership-safe reset.
- [ ] Record acceptance seams so each later phase can point to one fixture, one observable output, and one requirement ID.

## 7. Kế hoạch commit

1. `docs(implementation): add phase 01 contract and fixtures`
   - Thay đổi: add this standalone contract, fixture, logical-layout, and ownership plan only.
   - Cách kiểm tra: `git diff --check`; verify headings, metadata, canonical links, skill links, fixture IDs, and all requirement IDs.
2. `docs(implementation): update implementation index`
   - Thay đổi: reserved for the control owner when the phase is merged; no phase branch edits `index.md`.
   - Cách kiểm tra: validate primary ownership and dependency DAG from repository root.

## 8. Kiểm chứng và nghiệm thu

- [ ] Run the future corpus validator from repository root against `docs/plan/implementation/`; it must find one primary owner for all 25 FR/NFR IDs, unique Plan IDs, valid links, and all ten section headings.
- [ ] Run fixture-schema validation once `fixtures/` exists; each fixture must specify input, expected observable output, failure class where relevant, and owning requirement.
- [ ] Manually trace `FX-SHORTCUT-FOCUS`, `FX-QUERY-LIVE`, `FX-RANKING-DETERMINISTIC`, `FX-TAB-EVENTS`, `FX-RECONNECT`, `FX-STALE-ACTIVATION`, `FX-HOST-DOWN`, `FX-RESET-OWNED-DATA`, and `FX-PERMISSION-DENIED` to exactly one primary phase.
- [ ] Confirm no target path is presented as an observed source path while the implementation tree is absent.
- [ ] Acceptance signal: every FR/NFR and operational rule has a named owner, fixture or evidence seam, and later-phase dependency path.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: implementation scaffolding may appear after this plan is authored. Phương án xử lý đã chọn: reread the scaffold and replace only affected `to-create` paths before the affected phase commits.
- Rủi ro: two plans may later claim one module. Phương án xử lý đã chọn: keep one contract owner in `index.md`, serialize the dependent phase, and reject duplicate ownership.
- Rủi ro: a future toolchain may use different command names. Phương án xử lý đã chọn: require exact commands in the implementation phase that creates the toolchain; this phase records the acceptance seam, not a fabricated command.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: none; all current product decisions are binding in the canonical refactor documents.

## 10. References ngoài `docs/`

- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md), [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md), [`requirements-analysis`](../../../.agent/skills/common/foundation/requirements-analysis/SKILL.md), [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `extension/`, `host/`, `packaging/`, `fixtures/`, `tests/` (all `to-create`)
- Fixture/tool/artifact ngoài `docs/`: `fixtures/FX-*` catalog (to-create); future repository-root corpus validator (to-create)
