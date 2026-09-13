# Application lifecycle and configuration: examples

Examples are synthetic and non-authoritative; the linked canonical contracts win on conflict.

## Contract or workflow example

Startup order: load settings → validate paths → migrate SQLite → initialize repositories → inspect optional components → recover leases → start worker → serve traffic.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Add package-specific test selectors and failure-injection commands during implementation.

## Evidence example

```text
Task: APP-LIFECYCLE-001
Commit: <git-sha>
Command: <exact command>
Result: PASS (<count and duration>)
Artifact: <path or report link>
Deviation: none
```
