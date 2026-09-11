# 17 — Security và privacy

**Status:** `draft`
**Canonical references:** [Privacy/trust](../../../../product/internal-prd/05-privacy-and-trust.md) · [Security boundaries](../../../../architecture/07-security-boundaries.md) · [Test strategy](../../../../quality/01-test-strategy.md)
**Milestone:** M4
**Dependencies:** 10, 13, 18, 19

## Threat boundary

MVP bind `127.0.0.1`, single-user và không có auth. URL/file/text là untrusted input; Chroma/RocksDB/DuckDB là derived local stores. Bất kỳ việc mở network hoặc cloud provider nào đều cần threat-model mới.

## Controls

- Validate Host/Origin cho write requests; không bật CORS wildcard.
- URL chỉ HTTP(S), DNS/socket/redirect đều chặn loopback, private, link-local, reserved và metadata endpoints.
- File 20 MB, HTML 5 MB, text 1M chars/10k chunks.
- Escape Jinja/HTMX output; không render user HTML/Markdown trực tiếp.
- Secrets chỉ đọc từ environment; không ghi DB/log/backup không mã hóa nếu có provider key.
- Log redaction cho body, content, query nhạy cảm và URL có credential/query secret.

## Commit slices

1. `security: enforce host origin and request boundaries`
2. `security: harden url redirect and network validation`
3. `security: add rendering escaping and log redaction`
4. `test: add ingestion and rendering security cases`

## Failure behavior

Từ chối input trước khi tạo row/job; dùng mã lỗi ổn định và message không tiết lộ filesystem/network detail. Security check failure không retry vô hạn.

## Acceptance

SSRF fixture bị chặn ở initial URL và redirect; oversized input trả 413; XSS string render dạng text; secrets không xuất hiện trong captured log; write từ Origin không hợp lệ bị chặn.

## Review gate

Security checklist được chạy cùng test integration; reviewer inspect mọi nơi dùng `urlopen`, template `safe`, logging và subprocess.

## Execution log

Một phần limit/SSRF/escape đã có trong skeleton; cần audit và test đầy đủ.
