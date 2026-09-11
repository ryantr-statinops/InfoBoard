# Design — UI states and responsive behavior

## UI states

| State | Required behavior |
| --- | --- |
| Empty | Giải thích giá trị và dẫn tới Add |
| Loading | Placeholder giữ layout ổn định |
| Processing | Hiển thị bước/job state, không giả định đã indexed |
| Failed | Lý do an toàn, vùng ảnh hưởng và Retry khi phù hợp |
| Degraded | Chỉ rõ semantic/analytics unavailable; core flow vẫn dùng được |
| Duplicate/conflict | Giải thích item hiện có và hành động tiếp theo |
| Deleted | Đóng detail/refresh list nhưng giữ filter context |

## Responsive behavior

- Desktop target 1440 px: sidebar khoảng 240 px, dashboard hai cột, detail panel bên phải.
- Mobile target 390 px: một cột, menu thu gọn, detail thành full-width surface.
- Không có horizontal overflow trong add, list, detail, note và search flow.
- Table/chart có summary hoặc layout thay thế phù hợp mobile.

## Accessibility

- Tất cả action dùng được bằng keyboard và có visible focus.
- Form field có label; error liên kết với input và giữ dữ liệu chưa submit thành công.
- Status không chỉ truyền đạt bằng màu; loading/retry/degraded có text.
- Dialog/panel quản lý focus và Escape/close behavior rõ ràng.
- Imported content được render như untrusted text, không tạo active UI controls.
