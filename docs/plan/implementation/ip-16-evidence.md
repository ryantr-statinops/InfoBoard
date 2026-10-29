# IP-16 evidence - Diagnostics, health, and observability

Status: implementation and phase acceptance checks passed on `dev`.

## Delivered contract

- `host/diagnostics/` now owns the aggregate health state machine, freshness payload projection, versioned diagnostic events, typed error/retry mapping, allowlist redaction, bounded repair orchestration, and retention accounting.
- Health distinguishes host availability, protocol compatibility, projection freshness, index freshness, and SQLite persistence health. A storage-only failure reports `degraded` while lexical availability remains true.
- The existing IP-07 `health_result` wire shape is unchanged. Extended fields such as indexed revision, last handshake time, ranking/schema versions, diagnostic age, and migration status remain host-side for the later wire/UI integration phases.
- Every accepted health transition is paired with one pre-publication transition event. Event IDs and session correlation IDs are opaque; event payloads contain only bounded enums, versions, revisions, counts, and durations.
- Error classes preserve the IP-07 literals and map to stable user-safe statuses with explicit retryability, bounded attempt budgets, backoff, cooldown, and terminal actions. Local host-missing, permission-limited, storage-quarantine, and schema-upgrade conditions are included.
- Redaction is a single allowlist boundary for events, health payloads, mappings, SQLite diagnostic rows, and exports. Unknown fields are dropped; URLs, titles, queries, tokens, page content, browser IDs, tab IDs, and private-context values are not representable.
- Repairs are bounded: connection retry is capped at three attempts in five seconds, index rebuild is cancellable and revision-checked, persistence repair has a ten-second deadline, newer resync revisions cancel older requests, and exhaustion applies a cooldown with a user-actionable terminal result.
- Retention enforces the seven-UTC-day age bound, a 64 KiB per-record bound, and a `10_000_000`-byte logical payload budget. A bounded in-memory ring preserves diagnostics while persistence is unavailable and flushes only records that still satisfy policy after recovery.
- Five diagnostics fixtures were added and the catalog grew from 77 to 82 fixtures while retaining 19 phase seams. The IP-16 seam now references the full diagnostics suite.

## Commands and results

```sh
go test ./host/diagnostics/... -run 'TestHealth(Transitions|Freshness|Invariants)$'
go test ./host/diagnostics/... -run 'Test(ErrorStatusMatrix|RepairBounds|RepairCancellation)$'
go test ./host/diagnostics/... -run 'Test(Redaction|RetentionBoundaries|PersistenceDegraded)$'
go test ./tests/diagnostics/... -count=1
go vet ./host/... ./tests/...
gofmt -l host/diagnostics tests/diagnostics
INFOBOARD_DIAGNOSTIC_FIXTURES=fixtures/diagnostics node --test tests/diagnostics/*.test.mjs
python tests/implementation/validate_corpus.py
npm run build:extension
git diff --check
```

Observed:

- Health transition/freshness/invariant checks: PASS.
- Error matrix, repair bounds, and repair cancellation checks: PASS.
- Redaction, retention boundaries, and persistence-degraded checks: PASS.
- Diagnostics integration package: PASS.
- `go vet` over `host` and `tests`: PASS.
- `gofmt -l` produced no output.
- Diagnostics Node contract suite: 5/5 PASS.
- Corpus validator: PASS (20 phases, 236 tasks, 25 requirements, 82 fixtures, 7 invariants, 19 phase seams).
- Extension build: PASS.
- `git diff --check`: PASS.

## Coverage

| Acceptance area | Evidence |
|---|---|
| FR-012 availability and freshness | `TestHealthTransitions`, `TestHealthFreshness`, `TestHealthInvariants`, and `fixtures/diagnostics/health-transitions.jsonl` assert legal transitions, event sequence, final freshness, and lexical availability. |
| NFR-006 reconnect/rebuild | `TestRepairBounds`, `TestRepairCancellation`, `TestRepairIndexSuccessAndCancellation`, and `TestResyncNewerRevisionCancelsOlder` cover retry count, cancellation, revision checks, and terminal cooldown. |
| Complete error mapping | `TestErrorStatusMatrix`, `TestAllErrorClassesMapAndProtocolLiterals`, and `fixtures/diagnostics/error-status-matrix.json` cover all IP-07 classes plus local lifecycle conditions. |
| Redaction and privacy | `TestRedaction`, `TestRedactionDropsSensitiveFields`, `TestRedactEventAndPayload`, `TestSafeErrorAndMappingRedaction`, and `fixtures/diagnostics/redaction-sensitive-fields.json` inject sensitive values into every context slot and verify absence from exports. |
| Storage-degraded lexical path | `TestPersistenceDegraded` asserts `degraded` health, degraded wire storage, and lexical availability. |
| Age and byte retention | `TestRetentionBoundaries`, `TestRetentionAgeAndByteBoundaries`, `TestRetentionRejectsOversizedAndMalformed`, `TestRetentionFlushesPendingRingAfterRecovery`, and `fixtures/diagnostics/retention-boundaries.json` cover strict seven-day eviction, per-record rejection, bounded ring behavior, and oldest-first store pruning. |
| No unbounded repair/retention loop | `TestRepairExhaustionAndCooldown`, `TestRepairOneInFlight`, `TestRetentionPruneFailureDoesNotLoop`, and `TestRetentionStoreSeamPrunesToBudget` assert cooldown, single-flight, and bounded retry/flush behavior. |
| Fixture/catalog seam | `fixtures/catalog.json` and `tests/implementation/validate_corpus.py` register 82 fixtures and keep 19 phase seams; `tests/diagnostics/diagnostics-contract.test.mjs` checks the IP-16 seam and fixture bounds. |

## Known environment limits

- A broad `go test ./host/... ./tests/...` run is intermittently blocked by the local Windows Application Control policy for the pre-existing `host/internal/projection`, `host/internal/runtime`, `host/storage`, and `host/tests/storage` test binaries. The IP-16 diagnostics packages and integration tests pass, and the failures are environment-level binary launch blocks rather than assertion failures.
- Branded Chrome/Edge verification remains IP-19 work.
- Full `host/cmd` composition remains deferred to IP-18/IP-20; IP-16 verifies the diagnostics seam independently.
