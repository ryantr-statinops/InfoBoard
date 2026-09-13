# Embedding pipeline and revisions: examples

Examples are synthetic and non-authoritative.

## Contract or workflow example

An embedding response with 1536 dimensions cannot enter a revision declared as 768; the job fails safely and no vector index activation changes.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Use a controlled fake provider/store for deterministic failures and a separately recorded configured-provider smoke where required.

## Evidence example

```text
Task: SEM-EMBED-001
Commit: <git-sha>
Fixture: <fake provider/store condition>
Command: <exact command>
Result: PASS
Sensitive fields reviewed: redacted
Deviation: none
```
