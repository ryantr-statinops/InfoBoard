# References — Observability and diagnostics

| Type | Reference | Use |
| --- | --- | --- |
| Product | [Privacy and trust](../../../../product/internal-prd/05-privacy-and-trust.md), `PR-REC-04` | Safe diagnostics promise |
| Architecture | [Lifecycle/recovery](../../../../architecture/06-lifecycle-and-recovery.md), [security](../../../../architecture/07-security-boundaries.md), [interfaces](../../../../architecture/09-interface-contracts.md) | Component health, status, and redaction boundaries |
| Quality | [Quality attributes](../../../../quality/00-quality-attributes.md), [MVP gates](../../../../quality/03-mvp-quality-gates.md) | Observability evidence |
| Operations | [Health and troubleshooting](../../../../operations/02-health-and-troubleshooting.md) | Operator interpretation |
| Evidence | [`app/main.py`](../../../../../app/main.py), [`app/worker.py`](../../../../../app/worker.py), [`tests/test_core.py`](../../../../../tests/test_core.py) | Current baseline |
| Official | [Python logging](https://docs.python.org/3/library/logging.html), [FastAPI response status](https://fastapi.tiangolo.com/tutorial/response-status-code/) | Event/status reference |
