# Health and observability: examples

Examples are synthetic and non-authoritative.

## Contract or workflow example

A missing provider configuration reports provider=unconfigured and overall ready when local workflows work; a configured provider timeout reports degraded.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Record package-specific failure injection, parity, security, and recovery commands in the execution log.

## Evidence example

```text
Task: REL-HEALTH-001
Commit: <git-sha>
Scenario: <bounded failure or consistency case>
Command: <exact command>
Result: PASS
Canonical integrity: verified
Sensitive output: redacted
```
