# References — Security and privacy

| Type | Reference | Use |
| --- | --- | --- |
| Product | [Privacy and trust](../../../../product/internal-prd/05-privacy-and-trust.md), `PR-SEC-01…02` | User trust boundary |
| Architecture | [Security boundaries](../../../../architecture/07-security-boundaries.md), [ingestion](../../../../architecture/03-ingestion-pipeline.md) | Input and network controls |
| Quality | [Test strategy](../../../../quality/01-test-strategy.md) | Security regression layers |
| Operations | [Health/troubleshooting](../../../../operations/02-health-and-troubleshooting.md) | Safe diagnostics |
| Evidence | [`app/services.py`](../../../../../app/services.py), [`tests/test_api_flow.py`](../../../../../tests/test_api_flow.py) | Current baseline |
| Official | [OWASP SSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html), [Jinja autoescape](https://jinja.palletsprojects.com/en/stable/templates/) | Security reference |
