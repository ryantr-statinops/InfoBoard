# Canonical API foundation: examples

Examples are synthetic and non-authoritative; the linked canonical contracts win on conflict.

## Contract or workflow example

`PATCH /api/v1/bookmarks/42` with `If-Match: "7"` returns version 8; a second write with version 7 returns `409 RESOURCE_VERSION_CONFLICT` and a correlation ID.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Add package-specific test selectors and failure-injection commands during implementation.

## Evidence example

```text
Task: API-CANONICAL-001
Commit: <git-sha>
Command: <exact command>
Result: PASS (<count and duration>)
Artifact: <path or report link>
Deviation: none
```
