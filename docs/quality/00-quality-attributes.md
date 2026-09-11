# Quality — Attributes

| Attribute | Product expectation | Evidence |
| --- | --- | --- |
| Data integrity | Restart/retry/version/delete không mất hoặc trả dữ liệu sai | Migration/recovery integration tests |
| Availability | Core read/keyword flow sống khi derived store lỗi | Dependency failure tests |
| Security | Input/network/render/log boundaries được kiểm soát | Security suite |
| Usability | Core flow hoàn thành không cần advanced setup | Browser smoke test |
| Accessibility | Keyboard, focus, labels và non-color status | Manual/automated UI checks |
| Performance | Search/dashboard đạt target trên fixture chuẩn | Reproducible benchmark |
| Recoverability | Backup/restore/rebuild được kiểm chứng | Recovery transcript |
| Observability | Health/log đủ chẩn đoán mà không lộ content | Health/redaction tests |
| Portability | Core mode chạy clean checkout, full mode degraded rõ | Install matrix |

## Priority

Data integrity và core usability ưu tiên hơn semantic recall. Fallback an toàn ưu tiên hơn dependency mới; measurement có trước optimization.
