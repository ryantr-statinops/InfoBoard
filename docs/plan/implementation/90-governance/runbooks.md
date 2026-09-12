# 90 — Operational runbooks

**Status:** `draft`
**Scope:** local MVP and full mode

## Runbook index

### First run

1. Check Python 3.12 and `uv`.
2. Run `uv sync` or the explicitly supported full-mode install.
3. Export supported variables such as `INFOBOARD_DB` and verify paths stay within scope. Copying `.env.example` alone does not load configuration in the current runtime.
4. Run migration/health checks, then `uv run uvicorn app.main:app --reload`.

### Startup failure

Check `INFOBOARD_DB`, `data/` permissions, schema version, and `GET /api/health`. Do not delete the database; copy a backup before repair.

### Stuck or failed indexing

Inspect job state, retry count, and error code. Reindex the item or all eligible items. If a derived store fails, preserve FTS and run the rebuild path. Review the cause before retrying an exhausted job.

### Backup and restore

Pause the worker, create a manifest/checksum, restore to a temporary path, migrate/verify, rebuild derived stores, run smoke tests, and only then change the active path. See the [recovery guide](../40-m4-release/18-reliability/guides/recovery.md).

### Semantic degraded mode

Check model/Chroma/RocksDB health, continue with keyword search, and run model preparation/rebuild outside dashboard requests.

### Data safety

Never run destructive commands on `data/` without confirming the exact path and backup. Do not paste content or secrets into issues/logs.

## Evidence checklist

For each incident record timestamp, app/schema/dependency version, component state, command, error code, recovery result, and backup ID. Remove content and secrets.

## Execution log

Update this runbook with scrubbed transcripts after each release.
