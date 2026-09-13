# Canonical schema and migrations: examples

Examples are synthetic and non-authoritative; the linked canonical contracts win on conflict.

## Contract or workflow example

A legacy text item with ID 17 becomes bookmark 17 with source_kind=legacy_text; usable content becomes immutable snapshot version 1; no capture attempt is synthesized.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Add package-specific test selectors and failure-injection commands during implementation.

## Evidence example

```text
Task: DB-MIGRATION-001
Commit: <git-sha>
Command: <exact command>
Result: PASS (<count and duration>)
Artifact: <path or report link>
Deviation: none
```
