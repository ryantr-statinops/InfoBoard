# Examples — Packaging, release, and upgrades

## Current contract: core startup

```bash
uv sync
uv run uvicorn app.main:app --reload
```

## Target contract: upgrade sequence

```text
backup → locked dependency install → migration → health/integrity → smoke → resume
```

Rollback uses restore plus code revert; it does not run an automatic schema downgrade.
