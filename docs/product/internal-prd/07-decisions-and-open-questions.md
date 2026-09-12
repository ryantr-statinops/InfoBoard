# Internal PRD — Decisions and open questions

## Decisions locked for MVP

| Decision | Outcome |
| --- | --- |
| Product core | Advanced Bookmark Manager: URL/snapshot, notes/collections và keyword search |
| Primary audience | Người dùng phổ thông; power user có advanced path |
| Source scope | Text, public URL, PDF có text, Markdown và TXT |
| System of record | SQLite và stored snapshots |
| Retrieval baseline | Keyword luôn hoạt động; semantic là optional/full mode |
| Optional full mode | Semantic/hybrid retrieval, related items và topic clusters chỉ bật sau explicit setup; không chặn core release |
| Default privacy | Local processing; cloud/provider chỉ opt-in |
| Deletion | Soft delete trước, derived cleanup sau |
| Hậu MVP | Quản lý trong Next Plan và không chặn M1–M4 |

## Open questions

- Model embedding local và revision mặc định là gì?
- Full-mode dependency matrix hỗ trợ Linux distribution nào?
- UX boundary cuối cùng giữa simple mode và advanced settings là gì?
- Export format đầu tiên cho portable content là gì?
- Điều kiện nào đưa bookmark import từ Next Plan vào implementation roadmap?

## Decision process

- Thay đổi MVP scope phải cập nhật Internal PRD, các supporting layer bị ảnh hưởng và traceability trong cùng review cycle; `docs/plan/` chỉ được cập nhật sau khi review hoàn tất.
- Open question chỉ đóng khi có owner, evidence và consequence rõ ràng.
- Capability hậu MVP phải qua [Next Plan discovery gate](../next-plan/README.md) trước khi thêm requirement production.
- Technical decision history nằm trong Architecture; file này chỉ giữ product-level outcomes.
