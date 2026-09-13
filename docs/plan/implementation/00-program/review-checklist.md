# Program review checklist

## Structure

- [ ] Every implementation directory has a README.
- [ ] Every leaf package has the required six files.
- [ ] Package IDs, task IDs, anchors, links, and dependencies are unique and valid.
- [ ] Root and milestone statuses match package task state.

## Contract alignment

- [ ] All MVP requirements have feature, architecture, quality, package, task, and evidence links.
- [ ] `/api/v1/*` is canonical and `/api/*` is a one-release adapter only.
- [ ] SQLite integer IDs and canonical authority remain explicit.
- [ ] Failed capture attempts never become snapshots.
- [ ] FTS5, RocksDB, ChromaDB, and DuckDB remain derived and rebuildable.
- [ ] Semantic search is required for configured MVP release evidence.
- [ ] Analytics bounds, secret handling, consent, and legacy behavior are consistent.

## Evidence and release

- [ ] Checked tasks have real, redacted execution evidence.
- [ ] Tests, Ruff, migration, security, recovery, provider, UI, and performance gates pass.
- [ ] Examples match canonical schemas and are labeled non-authoritative.
- [ ] No canonical document links into implementation except the docs root index.
- [ ] Runtime changes and deviations are linked to accepted decisions.
- [ ] Git status is clean and the reviewed commit is present on `origin/dev`.
