# Phase 11 — Deterministic ranking and explanations

> Plan ID: IP-11
> Status: See README.md execution tracker
> Execution owner: Go ranking and search-quality implementer
> Dependencies: IP-10
> Parallel boundary: None; IP-12 consumes this phase's result and explanation contract after it is stable
> Requirement IDs: FR-005 (primary); NFR-004 (primary); FR-003, NFR-002 (supporting)
> Owned paths: `host/ranking/` (to-create), `fixtures/ranking/phase-11/` (to-create), `tests/ranking/phase-11/` (to-create)

## 1. Mục tiêu

- Define a local, integer-based ranking engine that orders eligible records from the IP-10 normalized in-memory index using lexical match quality, field importance, browser context, bounded activation recency, and freshness signals.
- Make FR-005 and NFR-004 executable: the same profile projection, normalized query, ranking timestamp, configuration/model version, and context inputs MUST produce identical result IDs, component scores, total scores, provenance, and explanations on every run.
- Make ranking understandable to the user and diagnosable to implementers without exposing hidden URL data. Every result carries bounded provenance and stable explanation codes that the UI can render as labels such as `title match`, `domain match`, `recent`, `current window`, `same group`, or `pinned`.
- Preserve the current open-tab boundary. Ranking consumes only the current profile's eligible projection and IP-10 searchable forms; it performs no browser calls, page evaluation, persistence lookup, network access, randomness, or external inference. No model artifact or network service is required at runtime.
- Supply IP-12 with a versioned result contract that can be rejected when the projection revision is unknown/rebuilding and can be rendered when optional persistence is degraded. Query-to-render measurements at the 1,000-tab/64-character envelope remain observable for NFR-002.

## 2. Phạm vi

- Bao gồm:
  - A pure Go ranking package under `host/ranking/` (to-create) with an explicit input, candidate, score, provenance, explanation, and result ordering contract.
  - Match classification for every required normalized query token: exact full-field, exact token, token prefix, contiguous phrase, fuzzy edit-distance/character-subsequence, and context-only matches on window/group/state labels.
  - The fixed `rank-v1` score expression and named integer components:
    ```text
    score = lexical
          + field_weight
          + context_bonus
          + recency_bonus
          + active_window_bonus
          + pinned_or_group_bonus
          - stale_or_missing_penalty
    ```
  - Match precedence, multi-token coverage, duplicate-title handling, active-window and same-group context, pinned state, bounded recency decay, stale/missing metadata handling, and stable identity tie-breaking.
  - A versioned explanation/provenance payload that is deterministic, bounded, privacy-safe, and suitable for `query_result`; raw URL/query/title values MUST NOT be copied into diagnostics or explanations by default.
  - Fixed ranking fixtures for exact matches, duplicates, typos, multiple windows/groups, pinned tabs, stale records, missing optional fields, explanation reproducibility, and the 1,000-tab/64-character latency envelope.
  - Focused unit and benchmark seams that compare complete observable outputs, not implementation details or map iteration order.
- Ngoài phạm vi:
  - Unicode normalization, case folding, tokenization, field bounds, postings, projection reconciliation, or index rebuild mechanics; IP-10 owns those contracts and this phase consumes them.
  - Query transport, result revision validation, host health, rebuilding/rejection states, UI rendering, keyboard interaction, or browser activation; IP-12, IP-13, and IP-14 own those boundaries.
  - SQLite reads/writes or durable recency retention; IP-08 supplies bounded activation metadata and the ranking input may fall back to session-only recency when persistence is degraded.
  - Any browser-history or page-content source, remote service, network request, page evaluation, SQL query, learned scorer, randomness, or user-specific ranking adaptation.
  - Changing the requirement ownership table: FR-005 and NFR-004 are primary here; FR-003 and NFR-002 receive supporting evidence only.

## 3. Điều kiện tiên quyết

- IP-10 MUST have defined the normalized query and candidate representation, field provenance, token positions, field limits, and immutable-index/rebuild boundary. Ranking MUST reject malformed or unbounded inputs before scoring rather than reimplementing normalization.
- IP-02 and IP-05 contracts are the source for opaque profile/tab/window/group identity, projection epoch/revision, eligible-record lifecycle, and canonical identity ordering. A browser tab ID, title, URL, or input slice position alone is not a sufficient final identity tie-breaker.
- Read the canonical [search and ranking contract](../refactor/search-and-ranking.md), [requirements](../refactor/requirements.md), [architecture](../refactor/architecture.md), [runtime protocol](../refactor/runtime-protocol.md), [domain and privacy contract](../refactor/domain-and-privacy.md), and [verification and acceptance contract](../refactor/verification-and-acceptance.md).
- The repository has no observed implementation source tree for this boundary. Keep all paths below marked `to-create` until implementation creates them; do not invent existing package symbols or move the ranking contract into IP-10's index path.
- The implementation must use an explicit `ranking_timestamp` supplied by the query caller. It MUST NOT read wall-clock time during scoring; this is necessary for repeatable recency tests and FR-005/NFR-004 evidence.

## 4. Đầu ra cần bàn giao

- [ ] `host/ranking/` (to-create) containing a pure ranking boundary with:
  - typed `RankInput` for `profile_id`, `projection_revision`, normalized query tokens, current window/group context, `ranking_timestamp`, `rank-v1` configuration, and bounded candidate records;
  - typed `RankedResult` containing the opaque result reference, total score, named component scores, rank model version, projection revision, provenance, explanation codes, and a deterministic sort key;
  - match classification and coverage logic for all six match classes;
  - integer score calculation with overflow-safe bounds and no floating-point or map-order dependence;
  - deterministic result sorting and bounded result limits.
- [ ] `fixtures/ranking/phase-11/` (to-create) with machine-readable fixtures and expected observable results:
  - `exact-precedence.json`: exact title/domain/URL, title prefix/phrase, fuzzy, and context-only candidates prove the required precedence;
  - `duplicate-title.json`: identical titles in different windows/groups remain distinct and resolve by context, recency, then stable identity;
  - `typo-fuzzy.json`: an edit-distance or character-subsequence candidate is returned below complete exact/token candidates and records fuzzy provenance;
  - `multi-window-context.json`: current-window bonus improves an otherwise comparable match but cannot overtake a materially stronger exact title match;
  - `pinned-group-context.json`: pin and same-group signals are small bounded bonuses and never become unconditional overrides;
  - `stale-missing.json`: stale or optional-metadata-missing records incur the named penalty, are explained, and never outrank an equivalent fresh record;
  - `explanation-reproducibility.json`: repeated runs with identical input and timestamp have byte-equivalent ordered output, scores, provenance, and explanation codes;
  - `rank-1000-tabs.json`: the NFR-002 benchmark input has 1,000 eligible records and a 64-character query with bounded expected result count.
- [ ] `tests/ranking/phase-11/` (to-create) covering exact/token/prefix/phrase/fuzzy/context classes, all-token coverage, duplicate IDs, multi-window context, pin/group signals, stale/missing penalties, stable ties, score-component arithmetic, model version, and explanation reproducibility.
- [ ] A handoff contract for IP-12: result references are profile/revision-bound, result order is already deterministic, total/result component scores are bounded, explanations are safe to display, and `rank-v1` is included in every non-error result.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - `common/engineering/documentation` — document the scoring contract, provenance vocabulary, fixtures, and reproducible verification commands.
  - `common/foundation/task-planning` — order the pure ranking slice, fixture work, integration seam, and acceptance evidence so another implementer can execute it without guessing.
  - `common/engineering/testing` — design deterministic behavioral fixtures for precedence, boundaries, invalid input, stale records, and performance.
  - `common/engineering/git-workflow` — keep the phase plan and future implementation slices isolated and reviewable.
  - `personal/decision/architecture-tradeoff` — compare a transparent fixed rule set with opaque/adaptive alternatives and record the bounded revisit trigger.
  - `personal/engineering/backend/go` — use explicit Go types, standard-library algorithms, bounded allocations, and observable package-level behavior for the host hot path.
- Tài liệu trong `docs/`:
  - [`docs/plan/refactor/README.md`](../refactor/README.md) — binding product boundary and deterministic-search rule.
  - [`docs/plan/refactor/search-and-ranking.md`](../refactor/search-and-ranking.md) — canonical match classes, score expression, context signals, ordering, and explanation rules.
  - [`docs/plan/refactor/requirements.md`](../refactor/requirements.md) — FR-003, FR-005, NFR-002, and NFR-004 acceptance signals.
  - [`docs/plan/refactor/architecture.md`](../refactor/architecture.md) — host/index ownership and failure isolation.
  - [`docs/plan/refactor/runtime-protocol.md`](../refactor/runtime-protocol.md) — result envelope, revision binding, ranking model version, and bounded query behavior.
  - [`docs/plan/refactor/domain-and-privacy.md`](../refactor/domain-and-privacy.md) — profile isolation, allowed fields, and redaction boundary.
  - [`docs/plan/refactor/verification-and-acceptance.md`](../refactor/verification-and-acceptance.md) — ranking fixtures, performance measurement, and privacy checks.
- Quy ước code, ADR, context ngoài `docs/`:
  - `CONTEXT.md` is the project context and repository-wide constraint source.
  - The IP-10 normalized field/provenance contract and IP-02 tab identity contract are inputs, not duplicate definitions. Any incompatible change requires updating the owning phase and the requirement/decision record before implementation.
  - The ranking implementation must remain a pure local package: no global mutable state, no hidden clock, no network client, and no dependency whose output is not pinned by the repository.

## 6. Công việc triển khai

- [ ] `IP-11-T01` Việc 1: Define the ranking input/output types at `host/ranking/` (to-create), including `profile_id`, `projection_revision`, `ranking_timestamp`, current window/group identity, normalized query tokens, candidate identity, field-token provenance, pinned/active state, activation timestamp, freshness/missing flags, `rank-v1`, and result limit. Validate profile/revision/identity/limits before scoring; fail closed for malformed input.
- [ ] `IP-11-T02` Việc 2: Implement per-token coverage. For each normalized query token, select the best eligible field evidence without allowing a candidate to pass merely because one token matched. Primary fields are title, domain, and URL; context fields are window, group, and state labels. A multi-token query is AND-oriented: every required token needs a primary match or an explicitly marked lower-confidence context fallback. Quoted contiguous text is one phrase evidence when IP-10 provides positions; it does not bypass bounded token limits.
- [ ] `IP-11-T03` Việc 3: Implement the match classes and precedence as a named, versioned table:
  - `exact_full_field`: normalized query equals the complete normalized field;
  - `exact_token`: one normalized query token equals one field token;
  - `token_prefix`: a field token starts with the complete query token;
  - `contiguous_phrase`: all phrase tokens occur contiguously and in order in one field;
  - `fuzzy`: bounded edit-distance or character-subsequence evidence after exact/prefix/phrase candidates are considered;
  - `context_only`: evidence comes only from window, group, or state labels and is lower confidence.
  `rank-v1` MUST prefer, within otherwise comparable coverage, exact title/token, exact domain/token, URL token, title prefix/phrase, other fuzzy, then context-only. A stronger exact class cannot be displaced solely by recency, active-window, pin, or group context.
- [ ] `IP-11-T04` Việc 4: Implement `rank-v1` score components as signed 64-bit integers and expose each component before total aggregation. The exact expression is:
  ```text
  score = lexical
        + field_weight
        + context_bonus
        + recency_bonus
        + active_window_bonus
        + pinned_or_group_bonus
        - stale_or_missing_penalty
  ```
  The initial constants MUST be checked into the ranking package and fixture expectations: lexical class bases `exact_full_field=1000`, `exact_token=900`, `token_prefix=700`, `contiguous_phrase=680`, `fuzzy=400`, `context_only=100`; field weights `title=80`, `domain=60`, `url=40`, `window=20`, `group=15`, `state=10`; `context_bonus` is 0–20 for bounded query/context agreement; `active_window_bonus` is 20 when candidate window equals the supplied current window, otherwise 0; `pinned_or_group_bonus` is the sum of pinned 5 and same-group 5 when that context is known, capped at 10; `recency_bonus` is 0–30 using one-minute buckets over a fixed 30-minute horizon; and `stale_or_missing_penalty` is 0 for fresh complete candidates, 40 for missing optional metadata, and 200 for a stale candidate. These values are not a hidden tuning surface: any change increments the ranking model version, updates fixtures, and records the ordering impact.
- [ ] `IP-11-T05` Việc 5: Make contextual signals bounded and subordinate. Active-window and same-group checks compare opaque identities from the query context; they never infer context from display text. Pin is a small stability/display signal and only contributes after primary coverage. Recent activation uses `max(0, ranking_timestamp - activated_at)` with future timestamps clamped to zero age, one-minute bucketing, and a 30-minute cap; absent, cross-profile, or out-of-horizon activation metadata yields zero. Stale/missing penalties cannot turn an invalid identity or profile into a valid result; hard-invalid records are rejected before ranking.
- [ ] `IP-11-T06` Việc 6: Define stable tie-breaking independent of map or input-slice order. Sort descending by total score, then by a canonical match-class histogram (more exact/full-field evidence before prefix, phrase, fuzzy, and context evidence), then by canonical field-preference signature (title, domain, URL, window, group, state), then by latest valid activation timestamp, active-window bit, pinned bit, and finally the opaque `tab_identity` bytes followed by `window_identity` bytes. Canonicalize query-token evidence by normalized token value before building signatures. Never use title, URL, array position, hash-map iteration, or a locale-dependent string comparison as the final identity key.
- [ ] `IP-11-T07` Việc 7: Emit deterministic provenance and explanation data. Each evidence item contains normalized query-token index/value reference, field enum, match class, bounded token/position reference, and component contribution; evidence is sorted by canonical token then field/class. User-facing explanation codes are an ordered, deduplicated set from `title_match`, `domain_match`, `url_match`, `window_context`, `group_context`, `state_context`, `recent_activation`, `current_window`, `same_group`, `pinned`, and `stale_or_missing`. Do not include raw query strings, URL fragments/query parameters, page contents, or private metadata in diagnostics. The result includes `ranking_model_version: "rank-v1"` and a stable explanation schema version.
- [ ] `IP-11-T08` Việc 8: Build the fixture-driven tests. Assert exact ordered opaque IDs, every named component, total score, class/provenance, explanation codes, model version, projection revision, and rejection of malformed/cross-profile input. Repeat each fixture at least twice with different candidate insertion order and assert byte-equivalent serialized output. Include duplicate titles across windows, a typo below exact matches, comparable candidates across multiple windows, pinned/same-group context, stale and missing fields, all-token coverage, empty candidate sets, and result-limit bounding.
- [ ] `IP-11-T09` Việc 9: Add the NFR-002 benchmark seam at `tests/ranking/phase-11/` (to-create). Use a checked-in 1,000-tab projection and a 64-character query, pass a fixed timestamp, run enough iterations to report p50/p95/p99 and allocations, and verify that the benchmark path performs no network or browser calls. This phase supplies ranking timing evidence; IP-12 owns end-to-end query-to-render measurement.

## 7. Kế hoạch commit

1. `feat(ranking): implement ip-11-t01`
   - Task IDs: `IP-11-T01`.
   - Owned target paths: host/ranking/.
   - Behavior: Việc 1: Define the ranking input/output types at `host/ranking/` (to-create), including `profile_id`, `projection_revision`, `ranking_timestamp`, current window/group identity, normalized query tokens, candidate identity, field-token provenance, pinned/active state, activation timestamp, freshness/missing flags, `rank-v1`, and result limit. Validate profile/revision/identity/limits before scoring; fail closed for malformed input.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-11 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 1: Define the ranking input/output types at `host/ranking/` (to-create), including `profile_id`, `projection_revision`, `ranking_timestamp`, current window/group identity, normalized query tokens, candidate identity, field-token provenance, pinned/active state, activation timestamp, freshness/missing flags, `rank-v1`, and result limit. Validate profile/revision/identity/limits before scoring; fail closed for malformed input.
   - Dependency gate: all index.md dependencies for IP-11 have merged to dev; phase work branch starts from latest origin/dev.

2. `feat(ranking): implement ip-11-t02`
   - Task IDs: `IP-11-T02`.
   - Owned target paths: `host/ranking/` (to-create), `fixtures/ranking/phase-11/` (to-create), `tests/ranking/phase-11/` (to-create).
   - Behavior: Việc 2: Implement per-token coverage. For each normalized query token, select the best eligible field evidence without allowing a candidate to pass merely because one token matched. Primary fields are title, domain, and URL; context fields are window, group, and state labels. A multi-token query is AND-oriented: every required token needs a primary match or an explicitly marked lower-confidence context fallback. Quoted contiguous text is one phrase evidence when IP-10 provides positions; it does not bypass bounded token limits.
   - Fixture and command: IP-10; run `the exact phase-11 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 2: Implement per-token coverage. For each normalized query token, select the best eligible field evidence without allowing a candidate to pass merely because one token matched. Primary fields are title, domain, and URL; context fields are window, group, and state labels. A multi-token query is AND-oriented: every required token needs a primary match or an explicitly marked lower-confidence context fallback. Quoted contiguous text is one phrase evidence when IP-10 provides positions; it does not bypass bounded token limits.
   - Dependency gate: all index.md dependencies for IP-11 have merged to dev; phase work branch starts from latest origin/dev.

3. `feat(ranking): implement ip-11-t03`
   - Task IDs: `IP-11-T03`.
   - Owned target paths: `host/ranking/` (to-create), `fixtures/ranking/phase-11/` (to-create), `tests/ranking/phase-11/` (to-create).
   - Behavior: Việc 3: Implement the match classes and precedence as a named, versioned table:
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-11 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 3: Implement the match classes and precedence as a named, versioned table:
   - Dependency gate: all index.md dependencies for IP-11 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(ranking): implement ip-11-t04`
   - Task IDs: `IP-11-T04`.
   - Owned target paths: `host/ranking/` (to-create), `fixtures/ranking/phase-11/` (to-create), `tests/ranking/phase-11/` (to-create).
   - Behavior: Việc 4: Implement `rank-v1` score components as signed 64-bit integers and expose each component before total aggregation. The exact expression is:
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-11 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 4: Implement `rank-v1` score components as signed 64-bit integers and expose each component before total aggregation. The exact expression is:
   - Dependency gate: all index.md dependencies for IP-11 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(ranking): implement ip-11-t05`
   - Task IDs: `IP-11-T05`.
   - Owned target paths: `host/ranking/` (to-create), `fixtures/ranking/phase-11/` (to-create), `tests/ranking/phase-11/` (to-create).
   - Behavior: Việc 5: Make contextual signals bounded and subordinate. Active-window and same-group checks compare opaque identities from the query context; they never infer context from display text. Pin is a small stability/display signal and only contributes after primary coverage. Recent activation uses `max(0, ranking_timestamp - activated_at)` with future timestamps clamped to zero age, one-minute bucketing, and a 30-minute cap; absent, cross-profile, or out-of-horizon activation metadata yields zero. Stale/missing penalties cannot turn an invalid identity or profile into a valid result; hard-invalid records are rejected before ranking.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-11 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 5: Make contextual signals bounded and subordinate. Active-window and same-group checks compare opaque identities from the query context; they never infer context from display text. Pin is a small stability/display signal and only contributes after primary coverage. Recent activation uses `max(0, ranking_timestamp - activated_at)` with future timestamps clamped to zero age, one-minute bucketing, and a 30-minute cap; absent, cross-profile, or out-of-horizon activation metadata yields zero. Stale/missing penalties cannot turn an invalid identity or profile into a valid result; hard-invalid records are rejected before ranking.
   - Dependency gate: all index.md dependencies for IP-11 have merged to dev; phase work branch starts from latest origin/dev.

6. `test(ranking): implement ip-11-t06`
   - Task IDs: `IP-11-T06`.
   - Owned target paths: `host/ranking/` (to-create), `fixtures/ranking/phase-11/` (to-create), `tests/ranking/phase-11/` (to-create).
   - Behavior: Việc 6: Define stable tie-breaking independent of map or input-slice order. Sort descending by total score, then by a canonical match-class histogram (more exact/full-field evidence before prefix, phrase, fuzzy, and context evidence), then by canonical field-preference signature (title, domain, URL, window, group, state), then by latest valid activation timestamp, active-window bit, pinned bit, and finally the opaque `tab_identity` bytes followed by `window_identity` bytes. Canonicalize query-token evidence by normalized token value before building signatures. Never use title, URL, array position, hash-map iteration, or a locale-dependent string comparison as the final identity key.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-11 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 6: Define stable tie-breaking independent of map or input-slice order. Sort descending by total score, then by a canonical match-class histogram (more exact/full-field evidence before prefix, phrase, fuzzy, and context evidence), then by canonical field-preference signature (title, domain, URL, window, group, state), then by latest valid activation timestamp, active-window bit, pinned bit, and finally the opaque `tab_identity` bytes followed by `window_identity` bytes. Canonicalize query-token evidence by normalized token value before building signatures. Never use title, URL, array position, hash-map iteration, or a locale-dependent string comparison as the final identity key.
   - Dependency gate: all index.md dependencies for IP-11 have merged to dev; phase work branch starts from latest origin/dev.

7. `feat(ranking): implement ip-11-t07`
   - Task IDs: `IP-11-T07`.
   - Owned target paths: `host/ranking/` (to-create), `fixtures/ranking/phase-11/` (to-create), `tests/ranking/phase-11/` (to-create).
   - Behavior: Việc 7: Emit deterministic provenance and explanation data. Each evidence item contains normalized query-token index/value reference, field enum, match class, bounded token/position reference, and component contribution; evidence is sorted by canonical token then field/class. User-facing explanation codes are an ordered, deduplicated set from `title_match`, `domain_match`, `url_match`, `window_context`, `group_context`, `state_context`, `recent_activation`, `current_window`, `same_group`, `pinned`, and `stale_or_missing`. Do not include raw query strings, URL fragments/query parameters, page contents, or private metadata in diagnostics. The result includes `ranking_model_version: "rank-v1"` and a stable explanation schema version.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-11 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 7: Emit deterministic provenance and explanation data. Each evidence item contains normalized query-token index/value reference, field enum, match class, bounded token/position reference, and component contribution; evidence is sorted by canonical token then field/class. User-facing explanation codes are an ordered, deduplicated set from `title_match`, `domain_match`, `url_match`, `window_context`, `group_context`, `state_context`, `recent_activation`, `current_window`, `same_group`, `pinned`, and `stale_or_missing`. Do not include raw query strings, URL fragments/query parameters, page contents, or private metadata in diagnostics. The result includes `ranking_model_version: "rank-v1"` and a stable explanation schema version.
   - Dependency gate: all index.md dependencies for IP-11 have merged to dev; phase work branch starts from latest origin/dev.

8. `test(ranking): implement ip-11-t08`
   - Task IDs: `IP-11-T08`.
   - Owned target paths: `host/ranking/` (to-create), `fixtures/ranking/phase-11/` (to-create), `tests/ranking/phase-11/` (to-create).
   - Behavior: Việc 8: Build the fixture-driven tests. Assert exact ordered opaque IDs, every named component, total score, class/provenance, explanation codes, model version, projection revision, and rejection of malformed/cross-profile input. Repeat each fixture at least twice with different candidate insertion order and assert byte-equivalent serialized output. Include duplicate titles across windows, a typo below exact matches, comparable candidates across multiple windows, pinned/same-group context, stale and missing fields, all-token coverage, empty candidate sets, and result-limit bounding.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-11 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 8: Build the fixture-driven tests. Assert exact ordered opaque IDs, every named component, total score, class/provenance, explanation codes, model version, projection revision, and rejection of malformed/cross-profile input. Repeat each fixture at least twice with different candidate insertion order and assert byte-equivalent serialized output. Include duplicate titles across windows, a typo below exact matches, comparable candidates across multiple windows, pinned/same-group context, stale and missing fields, all-token coverage, empty candidate sets, and result-limit bounding.
   - Dependency gate: all index.md dependencies for IP-11 have merged to dev; phase work branch starts from latest origin/dev.

9. `test(ranking): implement ip-11-t09`
   - Task IDs: `IP-11-T09`.
   - Owned target paths: tests/ranking/phase-11/.
   - Behavior: Việc 9: Add the NFR-002 benchmark seam at `tests/ranking/phase-11/` (to-create). Use a checked-in 1,000-tab projection and a 64-character query, pass a fixed timestamp, run enough iterations to report p50/p95/p99 and allocations, and verify that the benchmark path performs no network or browser calls. This phase supplies ranking timing evidence; IP-12 owns end-to-end query-to-render measurement.
   - Fixture and command: NFR-002, IP-12; run `the exact phase-11 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 9: Add the NFR-002 benchmark seam at `tests/ranking/phase-11/` (to-create). Use a checked-in 1,000-tab projection and a 64-character query, pass a fixed timestamp, run enough iterations to report p50/p95/p99 and allocations, and verify that the benchmark path performs no network or browser calls. This phase supplies ranking timing evidence; IP-12 owns end-to-end query-to-render measurement.
   - Dependency gate: all index.md dependencies for IP-11 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] Run exact scoped checks from repository root after the implementation exists:
  - `cd host && go test ./ranking/... -run 'TestPhase11Fixtures|TestExplanationReproducible'`.
  - `cd host && go test ./ranking/... -run '^$' -bench BenchmarkRank1000Tabs64Chars -benchmem`.
  - `cd host && go test ./ranking/... -run 'TestRankV1_(ExactPrecedence|DuplicateTitles|Fuzzy|Context|StaleMissing|TieBreak)'`.
- [ ] `exact-precedence.json` proves the declared ordering; no recent, active-window, pinned, or group bonus outranks a materially stronger exact lexical match.
- [ ] `duplicate-title.json` proves that same-title records remain separate, context signals are bounded, and equal-score ties resolve by the canonical opaque identity rather than input order.
- [ ] `typo-fuzzy.json` proves fuzzy evidence is returned only within bounded limits, is ranked below complete exact/token evidence when otherwise comparable, and carries `fuzzy` provenance.
- [ ] `multi-window-context.json` proves active-window context is a bonus rather than a filter or an exact-match override; `pinned-group-context.json` proves pin/group bonuses are bounded and explainable.
- [ ] `stale-missing.json` proves stale/missing penalties and explanation codes are stable, while hard-invalid profile/revision/identity inputs are rejected before scoring.
- [ ] `explanation-reproducibility.json` proves two or more runs with the same projection, query, timestamp, context, and `rank-v1` emit identical ordered IDs, component scores, total scores, provenance, explanation codes, and model/schema versions despite candidate insertion-order changes.
- [ ] The benchmark records ranking p50/p95/p99 and allocations for 1,000 indexed tabs and a 64-character query; the evidence is handed to IP-12 for the full query-to-render NFR-002 target of p95 <= 50 ms.
- [ ] Every result is bound to one `profile_id` and `projection_revision`, contains `ranking_model_version`, and has no network/browser/SQL/page-evaluation side effect. The result payload remains bounded by the protocol limits before IP-12 sends it.
- [ ] No known in-scope issue remains around match precedence, score arithmetic, deterministic tie-breaking, provenance privacy, or explanation reproducibility.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: integer weights can be changed casually and silently reorder the product. Phương án xử lý đã chọn: check `rank-v1` constants into the package, include component totals in fixtures, and require a model-version increment plus fixture review for any weight/class change.
- Rủi ro: context or recency can overwhelm lexical relevance. Phương án xử lý đã chọn: lexical class bases are at least 100 and context/recency/pin/group signals are explicitly capped; fixtures include an exact-vs-recent and exact-vs-active-window collision.
- Rủi ro: map iteration, candidate insertion order, locale, or wall-clock reads can make results flaky. Phương án xử lý đã chọn: explicit timestamp, canonicalized evidence signatures, opaque identity final tie-break, integer arithmetic, and reversed-input reproducibility tests.
- Rủi ro: stale records could be shown as activatable. Phương án xử lý đã chọn: apply a visible stale penalty and explanation for ranking fixtures, bind every result to projection revision, and require IP-12/IP-14 to reject or refresh stale revisions before activation.
- Rủi ro: explanation fields could leak private URL/query data. Phương án xử lý đã chọn: bounded enums/codes and token/position references only; raw values are excluded from diagnostics and user-facing explanation payloads by default.
- Rủi ro: a fixed rule set may underperform on a materially different tab corpus. Phương án xử lý đã chọn: benchmark fixed fixtures and the declared 1,000-tab envelope first. Revisit only when evidence shows the target latency or deterministic relevance acceptance is missed; any replacement must remain local, versioned, explainable, and covered by the same fixtures.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: whether a future browser API supplies a stable opaque identity representation different from IP-02's current identity; until then, IP-02's identity bytes are authoritative and no browser-specific fallback may be invented.

## 10. References ngoài `docs/`

- Skill: [`architecture-tradeoff`](../../../.agent/skills/personal/decision/architecture-tradeoff/SKILL.md)
- Skill: [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md)
- Skill: [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `host/ranking/` (to-create), `fixtures/ranking/phase-11/` (to-create), `tests/ranking/phase-11/` (to-create)
- Fixture/tool/artifact ngoài `docs/`: `fixtures/ranking/phase-11/{exact-precedence,duplicate-title,typo-fuzzy,multi-window-context,pinned-group-context,stale-missing,explanation-reproducibility,rank-1000-tabs}.json` (to-create); future Go commands in section 7 are the canonical scoped verification tools
