# M2 — Reliable ingestion

**Gate:** năm nguồn nhập và job durable hoạt động an toàn
**Dependencies:** M1

## Epic packages

- [13 — Ingestion sources](13-ingestion-sources/plan.md)
- [14 — Indexing worker và cache](14-indexing-worker/plan.md)

## Thứ tự

`13 → 14`; extractor phải tạo document contract trước khi worker nhận job.

## Acceptance gate

Text, TXT, Markdown, PDF và public URL tạo snapshot/chunk đúng; quá giới hạn/SSRF bị chặn; retry/restart không mất hoặc nhân đôi job.
