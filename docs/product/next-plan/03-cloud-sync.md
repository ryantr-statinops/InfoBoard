# Cloud sync

**Status:** `discovery`

## Outcome

Người dùng tùy chọn đồng bộ dữ liệu giữa thiết bị mà vẫn hiểu nơi dữ liệu được lưu, cách phục hồi và cách xóa/export dữ liệu.

## Assumptions

- Local mode tiếp tục hoạt động độc lập.
- Cloud sync là opt-in và có export/delete path.
- Collaboration chỉ được xây sau khi sync, identity và ownership model ổn định.

## Risks

Conflict offline, delete-vs-update, key recovery, quota/cost, schema drift và mất truy cập khi dịch vụ unavailable. Collaboration còn có accidental sharing, stale permissions và revoke không hoàn chỉnh.

## Dependencies

Change log/version model, identity, encryption/key management, conflict semantics, retention policy và service SLO. Collaboration bổ sung authorization matrix, audit log, invitation và revoke lifecycle.

## Implementation-ready gate

Prototype nhiều local replica, threat/privacy review, conflict acceptance, account/key recovery, quota model, export/delete guarantee, compatibility/versioning và degraded offline behavior phải được duyệt. Collaboration chỉ mở discovery riêng sau khi các gate nền tảng ổn định.
