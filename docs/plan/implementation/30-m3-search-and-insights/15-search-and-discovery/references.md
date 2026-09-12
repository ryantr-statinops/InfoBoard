# References — Search and discovery

| Type | Reference | Use |
| --- | --- | --- |
| Product | [Product requirements](../../../../product/internal-prd/02-product-requirements.md), `PR-RET-01`, `PR-RET-02`, `PR-RET-03`, `PR-RET-04`, `PR-RET-05` | Retrieval user value |
| Architecture | [Search and analytics](../../../../architecture/05-search-and-analytics.md), [indexing](../../../../architecture/04-indexing-pipeline.md), [interfaces](../../../../architecture/09-interface-contracts.md) | FTS, conditional semantic/RRF, hydration, and response boundaries |
| Design | [Core flows](../../../../design/01-core-flows-and-screens.md), [UI states](../../../../design/02-ui-states-and-responsive.md) | Search and degraded UI |
| Quality | [Search evaluation](../../../../quality/02-search-and-performance-evaluation.md) | Dataset and benchmark method |
| Evidence | [`app/semantic.py`](../../../../../app/semantic.py), [`tests/test_search.py`](../../../../../tests/test_search.py) | Current baseline |
| Official | [SQLite FTS5](https://sqlite.org/fts5.html), [Chroma docs](https://docs.trychroma.com/) | Retrieval dependencies |
