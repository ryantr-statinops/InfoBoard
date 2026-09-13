# RocksDB embedding cache: examples

Examples are synthetic and non-authoritative.

## Contract or workflow example

A cache entry from model A or chunking revision 1 is a miss for model B or revision 2; it is never coerced into compatibility.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Use a controlled fake provider/store for deterministic failures and a separately recorded configured-provider smoke where required.

## Evidence example

```text
Task: CACHE-ROCKS-001
Commit: <git-sha>
Fixture: <fake provider/store condition>
Command: <exact command>
Result: PASS
Sensitive fields reviewed: redacted
Deviation: none
```
