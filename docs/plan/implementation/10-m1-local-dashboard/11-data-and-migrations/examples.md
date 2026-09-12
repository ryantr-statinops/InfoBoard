# Examples — Data model and migrations

## Target contract: content-version key

```sql
CREATE TABLE item_contents (
    item_id INTEGER NOT NULL REFERENCES items(id),
    content_version INTEGER NOT NULL,
    content_text TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    PRIMARY KEY (item_id, content_version)
);
```

This is a target schema example. It must be validated by [T11-002](tasks.md#t11-002) against the canonical ERD before becoming evidence.

## Target contract: migration transaction

```text
lock database → begin transaction → apply migration → verify schema → update version → commit
```

On failure, roll back the transaction and restore from backup when data recovery is required.
