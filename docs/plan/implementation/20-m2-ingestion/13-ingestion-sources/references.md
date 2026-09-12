# References — Ingestion sources

| Type | Reference | Use |
| --- | --- | --- |
| Product | [Product requirements](../../../../product/internal-prd/02-product-requirements.md), `PR-CAP-01`, `PR-CAP-02`, `PR-CAP-03`, `PR-SEC-01`, `PR-SEC-02` | Capture and safety intent |
| Architecture | [Ingestion pipeline](../../../../architecture/03-ingestion-pipeline.md), [security boundaries](../../../../architecture/07-security-boundaries.md), [interfaces](../../../../architecture/09-interface-contracts.md) | Extractor, source endpoint, and SSRF boundary |
| Quality | [Test strategy](../../../../quality/01-test-strategy.md) | Fixture and limit coverage |
| Operations | [Local setup](../../../../operations/00-local-setup-and-modes.md) | Input/runtime handling |
| Evidence | [`app/services.py`](../../../../../app/services.py), [`tests/test_api_flow.py`](../../../../../tests/test_api_flow.py) | Current baseline |
| Official | [pypdf documentation](https://pypdf.readthedocs.io/), [Python urllib.parse](https://docs.python.org/3/library/urllib.parse.html) | Parser and URL reference |
