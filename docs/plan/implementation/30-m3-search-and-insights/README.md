# M3 — Search and insights

**Gate:** keyword luôn usable, full mode có semantic/hybrid và analytics
**Dependencies:** M2 chunks/index jobs

## Epic packages

- [15 — Search và discovery](15-search-and-discovery/plan.md)
- [16 — Analytics và insights](16-analytics/plan.md)

## Thứ tự

Hardening FTS trước, sau đó semantic/related/clusters; analytics dùng cùng filter contract và có thể phát triển song song.

## Acceptance gate

Keyword fallback hoạt động khi derived store lỗi; full mode có Chroma/RRF; analytics khớp SQLite; kết quả Việt/Anh đạt retrieval target.
