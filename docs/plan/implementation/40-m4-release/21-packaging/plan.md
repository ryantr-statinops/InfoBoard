# 21 — Packaging, release và upgrades

**Status:** `draft`
**Canonical references:** [Setup/modes](../../../../operations/00-local-setup-and-modes.md) · [Upgrade/release/rollback](../../../../operations/03-upgrade-release-and-rollback.md) · [MVP gates](../../../../quality/03-mvp-quality-gates.md)
**Milestone:** M4 release
**Dependencies:** 10, 11, 18, 20

## Install modes

- Core: `uv sync`, SQLite/FTS5, ingestion cơ bản và keyword search.
- Full: `uv sync --extra full`, ChromaDB, sentence-transformers, `rocksdict`, DuckDB và provider thật.
- Model preparation là command explicit; request dashboard không tự tải model.

## Release artifact

Version app, `uv.lock`, migration set, README setup, changelog, architecture diagram, runbooks và compatibility matrix. `data/`/`.env` không commit; `.env.example` không chứa secret.

## Upgrade flow

1. Stop/pause worker và tạo backup manifest.
2. `uv sync`, verify Python/dependency versions.
3. Run migrations transactionally.
4. Health/integrity check; rebuild derived index nếu metadata mismatch.
5. Smoke test dashboard/API; resume worker.

## Rollback

Code rollback bằng revert release/PR; data rollback bằng restore backup. Không downgrade schema tự động; ghi rõ version tối thiểu và migration compatibility.

## Commit slices

1. `chore: define core and full dependency extras`
2. `feat: add model preparation and environment validation`
3. `docs: add release upgrade and rollback instructions`
4. `ci: add release smoke verification`

## Acceptance

Developer mới chạy được core mode theo README; full mode báo thiếu dependency/model rõ ràng; upgrade từ current DB không mất dữ liệu; release checklist có test/lint/backup/restore evidence.

## Review gate

Test trên clean checkout và existing DB copy, verify `uv.lock`, `--extra full`, startup, `/api/health`, dashboard 200 và documented rollback.

## Execution log

README/lockfile core đã có; full extra, model preparation và release automation chưa có.
