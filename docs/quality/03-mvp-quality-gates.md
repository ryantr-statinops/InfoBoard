# Quality — MVP quality gates

## M1 — Local dashboard

- Fresh install/migration, add text, list/filter/detail, note/collection/status/delete pass.
- Restart giữ dữ liệu; desktop/mobile core flow không tràn layout.

## M2 — Reliable ingestion

- Text, TXT, Markdown, PDF và public URL tạo snapshot/chunk đúng.
- Limits/SSRF/parser failures an toàn; retry/restart không mất hoặc nhân đôi job.

## M3 — Search and insights

- Keyword fallback hoạt động khi derived dependencies lỗi.
- Full mode có semantic/RRF theo contract; analytics parity và retrieval target pass.

## M4 — Hardened release

- Security, backup/restore/rebuild, diagnostics và log-redaction tests pass.
- Full test, lint, type check, migration tests và benchmark report pass.
- Clean checkout chạy core mode; full mode chạy hoặc báo dependency/model thiếu rõ ràng.
- Requirement traceability verified hoặc có waiver được review.

## Evidence rule

Mỗi gate cần command/output, commit, environment và manual result trong execution log. Tài liệu target hoặc checklist chưa chạy không được tính là evidence.
