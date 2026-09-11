# Browser data portability

**Status:** `discovery`
**Product direction:** long-term commitment

## Outcome

Người dùng hiểu dữ liệu nào đang nằm trong browser, có thể mang theo phần dữ liệu được hỗ trợ và nhận báo cáo chính xác về phần không thể chuyển.

## Data layers

| Lớp | Ví dụ | Định hướng |
| --- | --- | --- |
| Portable content | bookmarks, URLs, titles, history references, page snapshots | Ưu tiên đầu tiên |
| Sensitive credentials | passwords, payment data, security keys | Chỉ qua security-reviewed workflow hoặc công cụ chuyên dụng |
| Session state | cookies, active login sessions, tokens | Rủi ro cao, không migrate mặc định |
| Browser storage | localStorage, IndexedDB, Cache Storage, Service Worker data | Cần nghiên cứu format, origin isolation và compatibility |

## Assumptions

- Chrome và Firefox profile không có format tương thích toàn phần.
- Content-oriented portability mang lại giá trị sớm hơn profile cloning.
- User consent phải được lấy theo từng data class và source profile.
- Không bypass encryption hoặc cơ chế bảo vệ của browser.

## Risks

- Mất hoặc làm hỏng profile nguồn/đích.
- Rò rỉ credential hoặc session token.
- Chuyển dữ liệu sai origin, tạo privilege escalation.
- Browser update làm parser hoặc storage mapping không còn tương thích.
- Người dùng hiểu nhầm “portable” là đảm bảo đăng nhập còn hiệu lực.

## Dependencies

Migration assistant, browser/profile discovery, export package, compatibility matrix, threat model và backup/restore verification.

## Implementation-ready gate

Cần chọn browser/platform matrix, phân loại data được hỗ trợ, prototype read-only trên profile copy, consent UX, encrypted handling, per-origin rules, failure/rollback behavior và acceptance fixtures. Mỗi data layer được duyệt độc lập; không có gate “full profile” duy nhất.
