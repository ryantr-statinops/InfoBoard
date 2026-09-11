# 05 — Dependencies và risks

**Status:** `ready`
**Environment:** Linux, Python 3.12, `uv`

## Dependency matrix

| Thành phần | Mode | Vai trò | Fallback |
| --- | --- | --- | --- |
| FastAPI/Uvicorn/Jinja2/HTMX assets | core | Web/API/UI | Không có |
| SQLite FTS5 | core | Source of truth + keyword | Block startup nếu không có FTS5 |
| `pypdf` | core | PDF text extraction | Báo unsupported nếu parser lỗi |
| sentence-transformers | full | Local embedding | Keyword search |
| ChromaDB | full | Persistent vector index | Keyword search |
| `rocksdict` | full | Embedding/cache state | SQLite cache/no-cache |
| DuckDB | full | Read-only analytics | SQLite aggregate |

`pyproject.toml` phải cung cấp extra `full`; lockfile được cập nhật sau khi dependency smoke test pass. Model chỉ tải bằng lệnh prepare rõ ràng và lưu metadata model/revision/dimension.

## Rủi ro và mitigation

| Rủi ro | Tác động | Mitigation | Gate |
| --- | --- | --- | --- |
| Native wheel không có trên Linux/Python 3.12 | Full mode không cài | Giữ core mode; pin version đã kiểm chứng; hướng dẫn lỗi | 05/21 |
| Model tải lớn/chậm | Startup và UX chậm | Lazy singleton, prepare command, warm-up benchmark | 15/20 |
| Chroma mất dữ liệu | Semantic unavailable | Rebuild từ SQLite chunks | 18 |
| DNS rebinding/redirect SSRF | Bảo mật local | Resolve/check từng redirect và địa chỉ kết nối | 13/17 |
| Schema drift | Mất dữ liệu | Versioned migration + backup + upgrade test | 11/18 |
| Worker crash giữa bước | Job kẹt | Durable checkpoints, lease/requeue, idempotent upsert | 14 |
| Analytics query nặng | Dashboard chậm | Read-only connection, bounded range, fallback | 16/20 |
| API contract drift | UI hỏng | Canonical contract + integration tests + traceability | 04/20 |

## Dependency acceptance

Mỗi dependency mới phải có lý do, version range, license check, import smoke test, failure message, fallback và cách gỡ bỏ. Không thêm CDN runtime; static assets phải chạy offline.

## Security assumptions

Ứng dụng chỉ bind `127.0.0.1` ở MVP. Hậu MVP mở network phải qua threat-model và authentication gate, không được suy ra từ cấu hình dev.
