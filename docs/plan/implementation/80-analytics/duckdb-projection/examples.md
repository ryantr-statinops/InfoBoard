# DuckDB analytics projection: examples

Examples are synthetic and non-authoritative.

## Contract or workflow example

If DuckDB checkpoint lags the canonical watermark, a 30-day request uses bounded SQLite and reports actual_backend=sqlite rather than presenting stale DuckDB data as current.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Record package-specific failure injection, parity, security, and recovery commands in the execution log.

## Evidence example

```text
Task: ANA-PROJECTION-001
Commit: <git-sha>
Scenario: <bounded failure or consistency case>
Command: <exact command>
Result: PASS
Canonical integrity: verified
Sensitive output: redacted
```
