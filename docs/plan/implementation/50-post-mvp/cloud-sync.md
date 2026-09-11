# 31 — Cloud sync và multi-device

**Status:** `discovery`
**Milestone:** M6
**Dependencies:** 11, 18, 21, 32

## Product outcome

Người dùng có thể dùng cùng thư viện trên nhiều thiết bị với offline-first semantics, không mất content/note và hiểu được conflict.

## Discovery questions

- Cloud nào là target và ai sở hữu encryption keys?
- Sync item content, note, collection và vector hay chỉ canonical data?
- Conflict resolution là last-write-wins, version merge hay manual review?
- Có cần end-to-end encryption và data export/erasure không?
- Chi phí storage/network và retention được giới hạn thế nào?

## Experiments

Prototype append-only change log trên một local replica; mô phỏng offline edits, concurrent note edits, delete-vs-update và schema upgrade. Đo latency, conflict rate, payload size và recovery.

## Candidate architecture (chưa cam kết)

SQLite vẫn canonical local; server lưu encrypted change records/object blobs; vectors rebuild theo device/model. Không sync secret/provider key.

## Implementation-ready gate

Cần privacy/threat review, identity model từ epic `32`, conflict contract, encryption/key recovery, server API/versioning, quota/cost, migration/export và failure SLO.

## Non-goals discovery

Chưa chọn vendor, chưa mở CORS/public port, chưa tạo account trong MVP.

## Execution log

Chưa discovery.
