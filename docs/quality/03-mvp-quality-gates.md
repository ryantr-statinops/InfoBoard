# Quality — MVP quality gates

## M1 — Local dashboard

- Fresh install/migration, add text, list/filter/detail, note/collection/status/delete pass.
- Restart preserves data; the desktop/mobile core flow has no layout overflow.

## M2 — Reliable ingestion

- Text, TXT, Markdown, PDF, and public URL produce correct snapshots/chunks.
- Limits/SSRF/parser failures are safe; retry/restart does not lose or duplicate jobs.

## M3 — Search and insights

- Keyword fallback works when derived dependencies fail.
- Full mode meets the semantic/RRF contract; analytics parity and retrieval targets pass.

## M4 — Hardened release

- Security, backup/restore/rebuild, diagnostics, and log-redaction tests pass.
- Full test, lint, type check, migration tests, and benchmark report pass.
- A clean checkout runs core mode; full mode runs or clearly reports missing dependencies/models.
- Requirement traceability is verified or has a reviewed waiver.

## Evidence rule

Each gate requires the command/output, commit, environment, and manual result in the execution log. Target docs or an unrun checklist do not count as evidence.
