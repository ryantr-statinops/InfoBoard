# 17 — Security and privacy

**Plan status:** `ready` · **Delivery status:** `not_started` · **Baseline coverage:** `partial`

The epic enforces the MVP trust boundary for local writes, untrusted content, network fetches, rendering, secrets, and logs.

**Requirements:** `PR-SEC-01…02`, `RQ-011`  
**Dependencies:** 10, 13, 18, 19

## Reading order

1. [Plan](plan.md)
2. [Tasks](tasks.md)
3. [References](references.md)
4. [Examples](examples.md)
5. [Execution evidence](execution.md)

## Package index

| Area | Location |
| --- | --- |
| Current input boundary | [`app/services.py`](../../../../../app/services.py) |
| Tests | [`tests/test_api_flow.py`](../../../../../tests/test_api_flow.py), [`tests/test_core.py`](../../../../../tests/test_core.py) |
| Security source | [architecture security boundaries](../../../../architecture/07-security-boundaries.md) |
