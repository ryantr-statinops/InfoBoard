# Operations — Health and troubleshooting

## Current state

Health endpoint cơ bản tồn tại; component-level status, diagnostics bundle và verified runbooks còn draft/partial.

## Target health model

`/api/health` báo `ok`, `degraded` hoặc `unavailable` cho SQLite, FTS, worker, ChromaDB, RocksDB, DuckDB và model. Chỉ SQLite/core-data unavailable làm toàn health trả HTTP 503.

## Troubleshooting routes

### Startup failure

- Kiểm tra Python/dependencies, config/data path, permissions, SQLite integrity và migration version.
- Không xóa database để thử lại; làm việc trên copy khi điều tra corruption.

### Stuck or failed indexing

- Xác định job/item/content version, state, retry count và error code.
- Requeue chỉ current version của non-deleted item.
- Sau retry limit, giữ failed state và dùng rebuild/diagnostic workflow.

### Semantic or analytics degraded

- Xác nhận core keyword/read flow còn hoạt động.
- Kiểm tra dependency/model metadata, derived-store path và health details.
- Rebuild derived store từ SQLite khi integrity của source data đã được xác nhận.

### Search/data mismatch

- So sánh item current version/deleted state với FTS/vector candidates.
- Rebuild index; không chỉnh SQLite dựa trên derived-store content.

## Diagnostics boundary

Bundle/log có component version, timing, job/error codes và hashed identifiers; không chứa raw content, query nhạy cảm, token, secret hoặc filesystem detail không cần thiết.
