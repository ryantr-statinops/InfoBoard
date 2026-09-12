# Tasks — Ingestion sources

- [ ] <a id="t13-001"></a>**T13-001 — Introduce the normalized ExtractedDocument contract**
  - Product: `PR-CAP-01`, `PR-CAP-02` · Evidence: `RQ-004`
  - Depends on: `T11-002` · Code: target extractor interface and `original_filename` metadata
  - Verify: text/Markdown/TXT fixture tests
  - Accept when: supported text inputs produce the same normalized document shape.
  - Commit: `feat: introduce extractor result and normalization pipeline`
  - Evidence: [execution entry](execution.md#t13-001)

- [ ] <a id="t13-002"></a>**T13-002 — Harden file and PDF extraction**
  - Product: `PR-CAP-02`, `PR-SEC-01` · Evidence: `RQ-004`, `RQ-011`
  - Depends on: `T13-001` · Code: Markdown/TXT/PDF adapters, `pypdf` dependency boundary
  - Verify: encoding, size, unsafe-content, and textless-PDF fixtures
  - Accept when: limits are enforced and textless PDFs return the stable error.
  - Commit: `feat: harden markdown txt and pdf extraction`
  - Evidence: [execution entry](execution.md#t13-002)

- [ ] <a id="t13-003"></a>**T13-003 — Add bounded public URL extraction**
  - Product: `PR-CAP-02`, `PR-SEC-01`, `PR-SEC-02` · Evidence: `RQ-004`, `RQ-011`
  - Depends on: `T13-001` · Code: HTTP client, redirect and destination checks
  - Verify: local HTTP fixture, timeout, response limit, and private-network redirect tests
  - Accept when: only safe bounded public HTTP(S) fetches are allowed.
  - Commit: `feat: add bounded public url extraction with redirect checks`
  - Evidence: [execution entry](execution.md#t13-003)

- [ ] <a id="t13-004"></a>**T13-004 — Unify hashing, URL canonicalization, and metadata**
  - Product: `PR-CAP-03` · Evidence: `RQ-005`
  - Depends on: `T13-002`, `T13-003` · Code: deduplication and provenance metadata
  - Verify: duplicate fixtures across source types
  - Accept when: duplicates reuse the existing item without overwriting user context.
  - Commit: `feat: unify source deduplication and metadata`
  - Evidence: [execution entry](execution.md#t13-004)

- [ ] <a id="t13-005"></a>**T13-005 — Verify source limits and security behavior**
  - Product: `PR-SEC-01`, `PR-SEC-02` · Evidence: `RQ-004`, `RQ-011`
  - Depends on: `T13-004` · Code: source fixture/security suite
  - Verify: `uv run pytest tests/test_api_flow.py tests/test_core.py -q`
  - Accept when: no partial item, unsafe destination, unbounded read, or temporary-file leak remains.
  - Commit: `test: cover all ingestion sources and limits`
  - Evidence: [execution entry](execution.md#t13-005)
