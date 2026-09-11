# InfoBoard — Data and privacy boundary

## Data classes

| Lớp dữ liệu | Ví dụ | Định hướng sản phẩm |
| --- | --- | --- |
| Portable content | bookmarks, URLs, titles, history references, page snapshots, metadata | Core/near-term; có thể import/export và lưu lâu dài |
| Sensitive credentials | passwords, payment data, security keys | Dài hạn; cần tool/format chuyên dụng, consent và security gate |
| Session state | cookies, active login sessions, tokens | Dài hạn; rủi ro cao, không mặc định migrate |
| Browser storage | localStorage, IndexedDB, Cache Storage, Service Worker data | Dài hạn; phụ thuộc browser/profile và cần feasibility gate |

## Boundary

MVP chỉ cam kết content-oriented portability: text, URL, file, bookmark/page content và metadata/provenance. Credentials, session state và browser storage không nằm trong acceptance của M1–M4.

Full portability vẫn là product commitment dài hạn, nhưng mỗi lớp phải đi qua threat model, compatibility matrix, consent flow, backup/recovery và fallback riêng. Không coi việc copy trực tiếp Chrome profile sang Firefox là một workflow được hỗ trợ mặc định.

## Privacy invariants

- Không đọc credential/session/storage nếu người dùng chưa chọn rõ phạm vi.
- Không gửi dữ liệu lên cloud mặc định.
- Không bypass encryption hoặc browser security boundary.
- Import/export phải có preview, cảnh báo và migration report.
- Khi một lớp không portable, giữ lại content/provenance có thể chuyển và báo chính xác phần bị bỏ qua.
