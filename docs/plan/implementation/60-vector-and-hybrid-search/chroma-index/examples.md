# ChromaDB semantic index: examples

Examples are synthetic and non-authoritative.

## Contract or workflow example

A high-score Chroma hit for snapshot version 2 is discarded when SQLite says version 3 is current, even if cleanup has not run yet.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Use a controlled fake provider/store for deterministic failures and a separately recorded configured-provider smoke where required.

## Evidence example

```text
Task: VEC-CHROMA-001
Commit: <git-sha>
Fixture: <fake provider/store condition>
Command: <exact command>
Result: PASS
Sensitive fields reviewed: redacted
Deviation: none
```
