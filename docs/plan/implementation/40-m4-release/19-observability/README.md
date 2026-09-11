# 19 — Observability and diagnostics

**Plan status:** `ready` · **Delivery status:** `not_started` · **Baseline coverage:** `partial`

The epic makes lifecycle events, health, degraded states, and diagnostics actionable without logging user content or secrets.

**Requirements:** `PR-REC-04`, `RQ-013`  
**Dependencies:** 10, 14, 15, 16, 17

## Reading order

1. [Plan](plan.md)
2. [Tasks](tasks.md)
3. [References](references.md)
4. [Examples](examples.md)
5. [Execution evidence](execution.md)

## Package index

| Area | Location |
| --- | --- |
| Health baseline | [`app/main.py`](../../../../../app/main.py), [`app/worker.py`](../../../../../app/worker.py) |
| Tests | [`tests/test_core.py`](../../../../../tests/test_core.py) |
| Operations | [health and troubleshooting](../../../../operations/02-health-and-troubleshooting.md) |
