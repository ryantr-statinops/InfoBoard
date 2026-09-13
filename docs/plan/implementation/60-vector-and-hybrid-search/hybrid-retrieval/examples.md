# Semantic and hybrid retrieval: examples

Examples are synthetic and non-authoritative.

## Contract or workflow example

A hybrid request during provider outage returns actual_mode=keyword with a semantic degradation entry, preserving the normal result schema and filters.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Use a controlled fake provider/store for deterministic failures and a separately recorded configured-provider smoke where required.

## Evidence example

```text
Task: VEC-HYBRID-001
Commit: <git-sha>
Fixture: <fake provider/store condition>
Command: <exact command>
Result: PASS
Sensitive fields reviewed: redacted
Deviation: none
```
