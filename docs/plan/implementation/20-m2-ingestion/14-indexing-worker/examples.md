# Examples — Indexing worker and cache

## Target contract: state mapping

```text
queued                         → queued
extracting | chunking | embedding → processing
indexed                        → indexed
failed                         → failed
```

The technical state remains in job records; the UI uses the simpler mapping above.

## Target contract: retry transition

```text
failed + retry_count < limit → queued → processing
failed + retry_count = limit → failed with stable reason
```

The transition must be atomic and covered by [T14-003](tasks.md#t14-003).
