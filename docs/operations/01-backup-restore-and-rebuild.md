# Operations — Backup, restore and rebuild

## Current state

Backup/restore/rebuild target đã được thiết kế nhưng chưa có verified release command/transcript.

## Backup contract

Backup authoritative gồm SQLite database, stored snapshots và manifest có app/schema version, timestamp và checksums. ChromaDB/RocksDB/DuckDB artifacts không bắt buộc vì có thể rebuild.

## Target backup flow

1. Resolve và hiển thị chính xác source/destination.
2. Đảm bảo SQLite backup nhất quán; không copy file đang ghi theo cách không an toàn.
3. Copy snapshots và tạo manifest/checksums.
4. Verify backup có thể mở và SQLite integrity check pass.

## Target restore flow

1. Không ghi đè destination có dữ liệu nếu chưa tạo safety backup.
2. Verify manifest/checksums và restore vào vị trí tạm/được xác định rõ.
3. Chạy SQLite integrity check.
4. Chạy migrations theo version.
5. Rebuild FTS, vector và cache từ SQLite/snapshots.
6. Chạy health và core smoke flow trước khi chuyển sang dữ liệu restored.

## Rebuild rules

- Rebuild idempotent/resumable và không sửa notes, collections hoặc organization status.
- Model/provider metadata mismatch tạo index mới, không tái sử dụng cache sai.
- Failure giữ dữ liệu chính nguyên vẹn và cho phép retry.
