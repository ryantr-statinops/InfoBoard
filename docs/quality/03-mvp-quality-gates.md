# Quality — MVP quality gates

## M1 — Local dashboard

- Fresh install/migration, add text, list/filter/detail, note/collection/status/delete pass.
- Restart preserves data; the desktop/mobile core flow has no layout overflow.

## M2 — Reliable ingestion

- Text, TXT, Markdown, PDF, and public URL produce correct snapshots/chunks.
- Limits/SSRF/parser failures are safe; retry/restart does not lose or duplicate jobs.

## M3 — Search and insights

- Core gate: keyword search, retrieval filters, fallback behavior, analytics parity, and the core query-set target pass without semantic dependencies.
- Optional full-mode gate: if full mode is included in the release scope, semantic/RRF, related-content, and cluster behavior meet their target contract. Missing full-mode dependencies do not block a core-only release.

## M4 — Hardened release

- Security, backup/restore/rebuild, diagnostics, and log-redaction tests pass.
- Full test, lint, type check, migration tests, and benchmark report pass.
- A clean checkout runs core mode without configuring optional components. If full mode is in release scope, its clean installation, model preparation, smoke test, and degraded fallback also pass.
- Core requirement traceability is verified or has a reviewed waiver. Conditional full-mode requirements are verified only when full mode is included in the release scope.

## Evidence rule

Each gate requires the command/output, commit, environment, and manual result in the execution log. Target docs or an unrun checklist do not count as evidence.
