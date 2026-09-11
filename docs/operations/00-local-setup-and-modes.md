# Operations — Local setup and modes

## Current state

Project chạy bằng Python/FastAPI baseline; clean-install, optional full extras và prepare-model workflow chưa được release-verified.

## Target setup flow

1. Cài Python 3.12 và `uv`.
2. Clone repository, cài core dependencies từ lockfile.
3. Sao chép `.env.example` nếu cần cấu hình; không commit secret.
4. Chạy versioned migrations trên database được xác định rõ.
5. Kiểm tra `/api/health` rồi khởi động `app.main:app` trên `127.0.0.1`.
6. Chỉ cài full extras/prepare model khi người dùng chọn semantic/analytics capability.

## Modes

| Mode | Required | Failure behavior |
| --- | --- | --- |
| Core | FastAPI, Jinja2/HTMX assets, SQLite/FTS5, extraction dependencies | SQLite/FTS unavailable phải fail startup/health |
| Full | Embedding model, ChromaDB, RocksDB, DuckDB | Thiếu derived dependency báo degraded; core mode vẫn hoạt động |

## Configuration principles

- Data directory, bind address và provider config có default local an toàn.
- Model download là prepare action rõ ràng, không xảy ra trong dashboard request.
- Health phải ghi component readiness và hướng dẫn remediation, không lộ secret/path nhạy cảm.
