# Internal PRD — Decisions and open questions

## Decisions locked for MVP

| Decision | Outcome |
| --- | --- |
| Product core | Advanced Bookmark Manager: URL/snapshot, notes/collections và keyword retrieval trong core mode |
| Primary audience | Người dùng phổ thông; power user có advanced path |
| Source scope | Text, public URL, PDF có text, Markdown và TXT |
| System of record | SQLite và stored snapshots |
| Retrieval baseline | Keyword luôn hoạt động; semantic/hybrid và related content là optional/full mode |
| Default privacy | Local processing; cloud/provider chỉ opt-in |
| Deletion | Soft delete trước, derived cleanup sau |
| Hậu MVP | Quản lý trong Next Plan và không chặn M1–M4 |

## Open questions by decision horizon

### MVP design review

- UX boundary cuối cùng giữa simple mode và advanced settings là gì? Design review sở hữu câu hỏi này; core flow không được phụ thuộc advanced setup.

### Conditional full-mode discovery

- Model embedding local và revision mặc định là gì?
- Full-mode dependency matrix hỗ trợ Linux distribution nào?

Hai câu hỏi full mode không chặn core MVP. Chúng chỉ phải đóng trước khi full mode được đưa vào release scope.

### Post-MVP discovery

- Export format đầu tiên cho portable content là gì?
- Điều kiện nào đưa bookmark import từ Next Plan vào implementation roadmap?

Hai câu hỏi này tiếp tục thuộc Next Plan và không chặn M1–M4.

## Decision process

- Thay đổi MVP scope phải cập nhật Internal PRD, implementation reference và traceability trong cùng review cycle.
- Open question chỉ đóng khi có owner, evidence và consequence rõ ràng.
- Capability hậu MVP phải qua [Next Plan discovery gate](../next-plan/README.md) trước khi thêm requirement production.
- Technical decision history tiếp tục nằm trong implementation governance; file này giữ product-level outcomes.
