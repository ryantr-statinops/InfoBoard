# 13 — Ingestion sources

**Plan status:** `ready`  
**Delivery status:** `not_started`  
**Baseline coverage:** `partial`  
**Milestone:** M2  
**Dependencies:** 10, 11
**Inputs:** text, Markdown, TXT, PDF, public URL

## Outcome

Every supported input creates one normalized `ExtractedDocument`, is hashed and versioned, and enters a durable job without exposing unsafe markup or exceeding resource limits.

## Extractor contract

```text
extract(input) -> title, text, source_type, source_url, original_filename, metadata
```

- Text derives a title from the first line when necessary and rejects empty content.
- Markdown/TXT prefer UTF-8, normalize newlines, preserve paragraphs, and do not render Markdown as HTML.
- PDF uses `pypdf`; scanned or textless PDFs return `pdf_text_unavailable`.
- URLs accept HTTP(S), remove fragments, allow at most five redirects, enforce a 30-second timeout, and check every redirect for SSRF.

## Limits and deduplication

Text is limited to 1,000,000 characters/10,000 chunks; files to 20 MB; and HTML responses to 5 MB. Exceeding a limit returns `413` rather than silently truncating.

Content-hash deduplication applies to every source. Canonical URL deduplication is additional. A duplicate returns the existing item and may attach a collection without overwriting notes or status.

## Failure, security, and acceptance

Do not log bodies, secrets, or full query strings. Block loopback, private, link-local, and reserved destinations after DNS resolution and socket connection checks. Parser failure must not create a partial item.

- Each fixture produces the expected title, content, source metadata, and chunk hash.
- Duplicate input creates no new item.
- Oversized input and private redirects are rejected.
- PDF text failure is explicit and actionable.

## Review gate

Verify API and direct-fixture paths, queued job creation, safe snapshot content, temporary-file cleanup, and security tests.
