# Provider configuration and consent: examples

Examples are synthetic and non-authoritative.

## Contract or workflow example

Provider verification may embed a fixed synthetic phrase to validate shape, but it cannot grant consent or send a bookmark excerpt.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Use a controlled fake provider/store for deterministic failures and a separately recorded configured-provider smoke where required.

## Evidence example

```text
Task: SEM-CONFIG-001
Commit: <git-sha>
Fixture: <fake provider/store condition>
Command: <exact command>
Result: PASS
Sensitive fields reviewed: redacted
Deviation: none
```
