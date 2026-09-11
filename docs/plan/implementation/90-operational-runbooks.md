# 90 — Operational runbooks

**Status:** `draft`
**Scope:** local MVP và full mode

## Runbook index

### First run

1. Kiểm tra Python 3.12 và `uv`.
2. `uv sync` hoặc `uv sync --extra full`.
3. Copy `.env.example` thành `.env`, kiểm tra path không trỏ ngoài scope.
4. Chạy migration/health, sau đó `uv run uvicorn app.main:app --reload`.

### Startup failure

Kiểm tra `INFOBOARD_DB`, quyền thư mục `data/`, schema version và `GET /api/health`. Không xóa DB; copy backup trước khi sửa.

### Stuck/failed indexing

Xem job state/attempt/error code, chạy reindex item hoặc toàn bộ; nếu derived store lỗi, giữ FTS và chạy rebuild. Job vượt retry phải được review nguyên nhân trước khi retry tiếp.

### Backup/restore

Pause worker, tạo manifest/checksum, restore vào path tạm, migrate/verify, rebuild derived stores, chạy smoke test rồi mới đổi active path.

### Semantic degraded

Xác nhận model/Chroma/RocksDB health; dùng keyword search tiếp tục; chạy model prepare/rebuild ngoài request dashboard.

### Data safety

Không chạy destructive command trên `data/` khi chưa xác nhận đường dẫn và backup. Không paste content/secret vào issue/log.

## Evidence checklist

Mỗi incident ghi timestamp, app/schema/dependency version, component state, command, error code, recovery result và backup ID; loại bỏ content/secret.

## Execution log

Runbook này sẽ được cập nhật bằng transcript đã scrub sau mỗi release.
