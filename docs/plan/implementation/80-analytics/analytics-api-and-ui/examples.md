# Analytics API and UI: examples

Examples are synthetic and non-authoritative.

## Contract or workflow example

A request omitting dates receives the previous 30-day half-open UTC range; all-time totals use the totals endpoint and cannot request an unbounded time series.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Record package-specific failure injection, parity, security, and recovery commands in the execution log.

## Evidence example

```text
Task: ANA-API-001
Commit: <git-sha>
Scenario: <bounded failure or consistency case>
Command: <exact command>
Result: PASS
Canonical integrity: verified
Sensitive output: redacted
```
