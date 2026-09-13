# Backup, restore, and rebuild: examples

Examples are synthetic and non-authoritative.

## Contract or workflow example

A valid restore activates canonical SQLite and snapshot files first; FTS, RocksDB, ChromaDB, and DuckDB may all start empty and are rebuilt afterward.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Record package-specific failure injection, parity, security, and recovery commands in the execution log.

## Evidence example

```text
Task: REL-RECOVERY-001
Commit: <git-sha>
Scenario: <bounded failure or consistency case>
Command: <exact command>
Result: PASS
Canonical integrity: verified
Sensitive output: redacted
```
