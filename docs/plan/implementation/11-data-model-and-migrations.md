# 11 — Data model và migrations

**Status:** `partial`
**Milestone:** M1
**Dependencies:** 10
**Source of truth:** SQLite + FTS5

## Outcome

Schema có version, nâng cấp lặp lại được và bảo toàn item/note/collection/job hiện tại. Mọi derived index đều có thể rebuild từ SQLite.

## Canonical schema

- `items`: metadata nguồn, `content_hash`, `content_version`, status, timestamps, `deleted_at`.
- `item_contents`: snapshot text theo `(item_id, content_version)`.
- `chunks`: `item_id`, version, position, text, hash và index ổn định.
- `collections`, `item_collections`, `notes`, `search_history`, `index_jobs`.
- `schema_version`: một dòng version hiện tại; migration chạy transaction và ghi history.
- `items_fts`: title + current content/chunks, chỉ index item chưa deleted.

## Data invariants

Foreign keys bật cho mọi connection; quan hệ item–collection unique; note phải thuộc item đang tồn tại; content version tăng đơn điệu; `deleted_at` loại khỏi public query nhưng giữ dữ liệu cho cleanup/restore.

## Migration strategy

1. Baseline migration mô tả schema đang chạy.
2. Mỗi migration là file SQL đánh số, idempotency chỉ dùng cho bootstrap.
3. `migrate()` lock database, chạy transaction, verify expected columns/indexes rồi cập nhật version.
4. Backup trước migration production; rollback dữ liệu bằng restore backup, không down-migration tự động.

## Commit slices

1. `feat: add versioned sqlite migration runner`
2. `feat: normalize item content version and indexes`
3. `feat: make fts synchronization transactional`
4. `test: cover fresh install and upgrade migrations`

## Failure handling

Migration lỗi phải rollback toàn transaction và trả lỗi startup có version/filename. FTS hỏng được rebuild từ canonical tables; không coi FTS là nguồn sự thật.

## Tests và acceptance

- Fresh DB tạo đúng schema/pragma/indexes.
- Nâng từ snapshot hiện tại lên version mới không mất rows.
- Re-run migration không tạo duplicate.
- Concurrent connection không đọc schema nửa chừng.
- Delete/restore giữ đúng note, collection và content version.

## Review gate

Có SQL diff, data compatibility note, fixture trước/sau migration và lệnh backup/restore thử nghiệm. Không merge nếu migration chưa test trên DB copy từ `data/`.

## Execution log

Chưa bắt đầu.
