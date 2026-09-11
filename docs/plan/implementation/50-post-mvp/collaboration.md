# 32 — Identity, sharing và collaboration

**Status:** `discovery`
**Canonical product reference:** [Cloud sync and collaboration direction](../../../product/next-plan/03-cloud-sync.md)
**Milestone:** M6–M7
**Dependencies:** 17, 21, 31

## Product outcome

Chuyển single-user local library thành workspace có identity, quyền truy cập và chia sẻ có kiểm soát mà không làm thay đổi semantics core.

## Discovery questions

- Đơn vị quyền là user, workspace, collection hay item?
- Vai trò tối thiểu: owner/editor/viewer hay policy chi tiết hơn?
- Share link có public, expiring và revoke được không?
- Note cá nhân và note chia sẻ khác nhau thế nào?
- Migration accountless local data sang workspace thực hiện ra sao?

## Prototype gate

Thiết kế policy matrix và mock authorization middleware; kiểm thử owner/editor/viewer trên collection/item/note. Không kết nối cloud thật trước khi threat model và data ownership được duyệt.

## Implementation-ready gate

Phải có identity provider, session/token lifecycle, authorization matrix, audit log, invitation/revoke, export/delete và migration plan. Mọi API mới phải versioned, không phá local API.

## Risks

Privilege escalation, share link leak, stale permissions, accidental public content và migration nhầm ownership.

## Execution log

Chưa discovery.
