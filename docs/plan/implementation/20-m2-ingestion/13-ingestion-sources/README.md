# 13 — Ingestion sources

**Plan status:** `ready` · **Delivery status:** `not_started` · **Baseline coverage:** `partial`

The epic unifies text, file, PDF, and public URL extraction behind a bounded `ExtractedDocument` contract.

**Requirements:** `PR-CAP-01`, `PR-CAP-02`, `PR-CAP-03`, `PR-SEC-01`, `PR-SEC-02`, `RQ-004`, `RQ-005`, `RQ-011`
**Dependencies:** 10, 11

## Reading order

1. [Plan](plan.md)
2. [Tasks](tasks.md)
3. [References](references.md)
4. [Examples](examples.md)
5. [Execution evidence](execution.md)

## Package index

| Area | Location |
| --- | --- |
| Current extractor baseline | [`app/services.py`](../../../../../app/services.py) |
| API flow tests | [`tests/test_api_flow.py`](../../../../../tests/test_api_flow.py) |
| Security boundaries | [architecture security](../../../../architecture/07-security-boundaries.md) |
