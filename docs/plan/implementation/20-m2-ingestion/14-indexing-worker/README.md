# 14 — Indexing worker and cache

**Plan status:** `ready` · **Delivery status:** `not_started` · **Baseline coverage:** `partial`

The epic turns extracted documents into durable chunks and derived indexes through a restartable worker lifecycle.

**Requirements:** `PR-CAP-05`, `PR-REC-01`, `PR-REC-02`, `PR-REC-03`, `RQ-006`
**Dependencies:** 11, 13

## Reading order

1. [Plan](plan.md)
2. [Tasks](tasks.md)
3. [References](references.md)
4. [Examples](examples.md)
5. [Execution evidence](execution.md)

## Package index

| Area | Location |
| --- | --- |
| Worker baseline | [`app/worker.py`](../../../../../app/worker.py) |
| Semantic adapter | [`app/semantic.py`](../../../../../app/semantic.py) |
| Search tests | [`tests/test_search.py`](../../../../../tests/test_search.py) |
