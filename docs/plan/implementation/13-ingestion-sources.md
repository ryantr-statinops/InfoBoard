# 13 — Ingestion sources

**Status:** `partial`  
**Milestone:** M2  
**Dependencies:** 10, 11, 17  
**Inputs:** text, Markdown, TXT, PDF, public URL

## Outcome

Mọi nguồn nhập tạo một `ExtractedDocument` thống nhất, được normalize, hash, version và đưa vào job mà không làm lộ HTML/script hoặc vượt giới hạn tài nguyên.

## Extractor contract

```text
extract(input) -> title, text, source_type, source_url, original_filename, metadata
```

- Text: title bắt buộc hoặc derive từ dòng đầu; content không rỗng.
- Markdown/TXT: UTF-8 ưu tiên, newline normalize, giữ đoạn; không render Markdown thành HTML.
- PDF: `pypdf` đọc text; PDF scan/OCR trả lỗi `pdf_text_unavailable`.
- URL: chỉ HTTP(S), canonicalize bỏ fragment, tối đa 5 redirect, timeout 30s; từng redirect kiểm tra SSRF.

## Limits

Text sau extract tối đa 1.000.000 ký tự/10.000 chunks; file tối đa 20 MB; response HTML tối đa 5 MB. Vượt giới hạn trả `413`, không cắt âm thầm.

## Deduplication

Content hash dedup cho mọi nguồn; canonical URL dedup bổ sung. Duplicate trả item cũ và chỉ attach collection mới, không ghi đè note/status.

## Commit slices

1. `feat: introduce extractor result and normalization pipeline`
2. `feat: harden markdown txt and pdf extraction`
3. `feat: add bounded public url extraction with redirect checks`
4. `feat: unify source deduplication and metadata`
5. `test: cover all ingestion sources and limits`

## Failure/security behavior

Không log body, secret hoặc full URL query. DNS resolve và socket destination phải thuộc public address; chặn loopback/private/link-local/reserved. Parser lỗi trả mã ổn định và không tạo item một phần.

## Tests và acceptance

Fixture từng source tạo đúng title/content/source metadata/chunk hash; duplicate không tạo row mới; file/HTML/text quá giới hạn bị từ chối; redirect vào mạng nội bộ bị chặn; PDF không text báo lỗi rõ.

## Review gate

Chạy import qua API và fixture trực tiếp, xác nhận job queued, content snapshot không chứa markup nguy hiểm, không có file tạm còn sót và test security pass.

## Execution log

Extractor hiện có bản tối thiểu trong `app/services.py`; cần tách interface và harden.
