# Migration assistant

**Status:** `discovery`

## Outcome

Người dùng có flow có hướng dẫn để inspect, preview, import/export và kiểm tra kết quả migration theo từng loại dữ liệu.

## Target experience

```text
Select source → Scan read-only → Preview → Choose scope → Backup → Migrate → Verify → Report
```

Simple mode dùng lựa chọn an toàn và giải thích bằng ngôn ngữ phổ thông. Advanced mode cho phép chọn profile, data class, mapping, conflict policy, export format và diagnostic log.

## Product interfaces

- Source adapter nhận diện browser/profile hoặc package.
- Import preview hiển thị count, data class, warnings và unsupported records.
- Export package giữ content, metadata và provenance.
- Migration report phân loại `imported`, `skipped`, `conflicted`, `failed`.
- Consent boundary ghi rõ dữ liệu được đọc và destination được ghi.

Đây là interface cấp product, chưa phải API/schema production.

## Risks

Sai profile, ghi đè dữ liệu, duplicate, conflict không thể đảo ngược, report thiếu chính xác hoặc người dùng bỏ qua cảnh báo dữ liệu nhạy cảm.

## Dependencies

Browser portability decisions, source adapters, backup/restore, conflict model, package versioning và end-to-end fixture matrix.

## Implementation-ready gate

Cần prototype read-only scan, UX test simple/advanced mode, conflict/default policy, backup bắt buộc, resumability, verification strategy và rollback procedure được duyệt.
