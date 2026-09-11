# Cloud sync and collaboration

**Status:** `discovery`

## Outcome

Người dùng tùy chọn đồng bộ dữ liệu giữa thiết bị và chia sẻ/cộng tác mà vẫn hiểu nơi dữ liệu được lưu, ai có quyền truy cập và cách thu hồi quyền.

## Assumptions

- Local mode tiếp tục hoạt động độc lập.
- Cloud sync là opt-in và có export/delete path.
- Collaboration chỉ được xây sau khi identity và ownership model ổn định.

## Risks

Conflict offline, delete-vs-update, key recovery, accidental sharing, stale permission, quota/cost, schema drift và mất khả năng truy cập khi dịch vụ unavailable.

## Dependencies

Change log/version model, identity, encryption/key management, authorization matrix, audit log, conflict semantics, retention policy và service SLO.

## Implementation-ready gate

Prototype nhiều local replica, threat/privacy review, conflict acceptance, account/key recovery, quota model, export/delete guarantee, compatibility/versioning và degraded offline behavior phải được duyệt.
