# 91 — Decision register

**Status:** `active`

## Quy ước

Mỗi quyết định có ID, ngày, status, context, decision, alternatives, impact và link tới epic. Không sửa lịch sử; quyết định mới tạo record supersede record cũ.

## Decisions

| ID | Status | Quyết định | Ảnh hưởng |
| --- | --- | --- | --- |
| ADR-001 | accepted | SQLite là source of truth; derived stores rebuildable. | 11, 14, 18 |
| ADR-002 | accepted | MVP single-user local-first, bind `127.0.0.1`. | 17, 31, 32 |
| ADR-003 | accepted | UI dùng Jinja2 + HTMX, không SPA/Node build. | 12, 21 |
| ADR-004 | accepted | Core mode không phụ thuộc semantic; full mode là optional extra. | 05, 15, 21 |
| ADR-005 | accepted | Feature làm trên `dev`, commit nhỏ, user review/merge từng PR. | 03, mọi epic |
| ADR-006 | proposed | Chuẩn hóa list response thành `{items,total,limit,offset}`. | 04, 12, 20 |
| ADR-007 | proposed | Worker MVP tuần tự trong process với SQLite checkpoint/lease. | 14, 19 |

## Template record

```text
ID / date / status
Context and problem
Decision
Alternatives rejected and why
Consequences / migration
Owners and affected epics
Evidence and supersedes
```

## Review policy

Decision ảnh hưởng public API, schema, security boundary hoặc dependency phải được ghi trước khi code; nếu cần user preference, chuyển epic về `draft` và hỏi trong review gate.
