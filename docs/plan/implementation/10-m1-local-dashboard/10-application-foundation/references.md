# References — Application foundation

| Type | Reference | Use |
| --- | --- | --- |
| Product | [PRD overview](../../../../product/internal-prd/00-overview.md), `PR-UX-01` | Local dashboard outcome |
| Architecture | [System overview](../../../../architecture/00-system-overview.md), [tech stack](../../../../architecture/01-tech-stack.md) | Process and runtime boundaries |
| Quality | [Test strategy](../../../../quality/01-test-strategy.md) | Factory and lifecycle verification |
| Operations | [Local setup](../../../../operations/00-local-setup-and-modes.md) | Startup contract |
| Evidence | [`app/main.py`](../../../../../app/main.py), [`tests/test_core.py`](../../../../../tests/test_core.py) | Current baseline |
| Official | [FastAPI lifespan](https://fastapi.tiangolo.com/advanced/events/) | Lifecycle reference; verify version at implementation time |

Target modules are not evidence until they exist and are tested.
