# Quality — Attributes

| Attribute | Product expectation | Evidence |
| --- | --- | --- |
| Data integrity | Restart/retry/version/delete does not lose or misreport data | Migration/recovery integration tests |
| Availability | Core read/keyword flow survives derived-store failure | Dependency failure tests |
| Security | Input/network/render/log boundaries are controlled | Security suite |
| Usability | Core flow completes without advanced setup | Browser smoke test |
| Accessibility | Keyboard, focus, labels, and non-color status | Manual/automated UI checks |
| Performance | Search/dashboard meets targets on the standard fixture | Reproducible benchmark |
| Recoverability | Backup/restore/rebuild is verified | Recovery transcript |
| Observability | Health/logs support diagnosis without exposing content | Health/redaction tests |
| Installability / environment portability | Core mode runs from a clean checkout; full mode is conditional and degrades only after explicit enablement fails | Install matrix |

## Priority

Data integrity and core usability take priority over semantic recall. Safe fallback takes priority over new dependencies; measurement comes before optimization.

An optional component that was never configured is absent, not degraded. Degraded evidence applies only after the user explicitly enables a full-mode component and that component becomes unavailable or incompatible.
