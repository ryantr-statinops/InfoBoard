# Examples — Reliability, backup, and recovery

## Target contract: restore sequence

```text
verify manifest → create safety copy → restore SQLite → migrate → rebuild derived stores → verify health → resume
```

Never replace the active database before the manifest and checksum pass.
