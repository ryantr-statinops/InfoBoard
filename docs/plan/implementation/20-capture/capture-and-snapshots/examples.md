# Bookmark capture and snapshots: examples

Examples are synthetic and non-authoritative; the linked canonical contracts win on conflict.

## Contract or workflow example

A failed recapture records attempt 4 as failed but leaves snapshot version 3 current. Retrying attempt 4 cannot create two snapshot version 4 rows.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Add package-specific test selectors and failure-injection commands during implementation.

## Evidence example

```text
Task: CAP-SNAPSHOT-001
Commit: <git-sha>
Command: <exact command>
Result: PASS (<count and duration>)
Artifact: <path or report link>
Deviation: none
```
