# Architecture — Data model and ERD

## Current state

SQLite schema và FTS5 đã tồn tại ở mức partial; chưa có migration history hoặc content-version schema hoàn chỉnh. Runtime hiện dùng `items.status DEFAULT 'active'`, `chunks.position`, một `item_contents` row/item và `index_jobs.attempts`. Target dưới đây chuyển default sang `inbox`, content key sang `(item_id, content_version)` và attempts sang `retry_count` qua epic 11.

## Target state

```mermaid
erDiagram
    ITEMS ||--o{ ITEM_CONTENTS : versions
    ITEMS ||--o{ CHUNKS : contains
    ITEMS ||--o{ NOTES : has
    ITEMS ||--o{ INDEX_JOBS : processes
    ITEMS ||--o{ ITEM_COLLECTIONS : grouped_by
    COLLECTIONS ||--o{ ITEM_COLLECTIONS : contains

    ITEMS {
        integer id PK
        text source_type
        text source_url
        text original_filename
        text title
        text content_hash
        integer content_version
        text status "DEFAULT inbox"
        text created_at
        text updated_at
        text deleted_at
    }
    ITEM_CONTENTS {
        integer item_id PK, FK
        integer content_version PK
        text content
        text content_hash
    }
    CHUNKS {
        integer id PK
        integer item_id FK
        integer content_version
        integer position
        text text
        text content_hash
    }
    COLLECTIONS {
        integer id PK
        text name
    }
    ITEM_COLLECTIONS {
        integer item_id FK
        integer collection_id FK
    }
    NOTES {
        integer id PK
        integer item_id FK
        text content
        text created_at
        text updated_at
    }
    INDEX_JOBS {
        integer id PK
        integer item_id FK
        integer content_version
        text state
        integer retry_count
        text error_code
    }
    SEARCH_HISTORY {
        integer id PK
        text query
        text created_at
    }
    SCHEMA_VERSION {
        integer version
        text applied_at
    }
```

## Invariants

- Foreign keys được bật; `(item_id, collection_id)` là unique.
- `item_contents` dùng composite key `(item_id, content_version)`; `chunks.position` ổn định trong từng version.
- Item mới có organization status mặc định `inbox`; runtime default `active` là migration/behavior gap của M1.
- `index_jobs.retry_count` là target name; runtime `attempts` được chuyển đổi trong migration thay vì tồn tại song song.
- Snapshot/chunk thuộc content version cụ thể.
- `deleted_at` loại item khỏi mọi public read path.
- Notes và organization state không bị ghi đè bởi reindex.
- Schema thay đổi qua numbered migration, backup trước upgrade và upgrade test.
- FTS chỉ phản ánh current, non-deleted content và được giữ đồng bộ transactionally.
