# Discovery — Migration assistant

## Target experience

```text
Select source → Scan read-only → Preview → Choose scope → Backup → Migrate → Verify → Report
```

Product interfaces include source adapter, import preview, export package with provenance, migration report (`imported`, `skipped`, `conflicted`, `failed`), and per-source/data-class consent. They are not production APIs yet.

## Implementation-ready gate

Require read-only scan prototype, simple/advanced UX test, conflict/default policy, mandatory backup, resumability, verification, rollback, and end-to-end fixtures.
