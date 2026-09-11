# InfoBoard — Product principles

1. **Local-first và user ownership.** Dữ liệu chính ở local; cloud chỉ là lựa chọn rõ ràng của người dùng.
2. **Import trước, lock-in thấp.** Người dùng phải có đường vào và đường export có ý nghĩa.
3. **Nội dung lâu dài trước trạng thái phiên.** Page content, bookmark, metadata và notes là core; session state là lớp nhạy cảm riêng.
4. **Progressive disclosure.** Simple mode mặc định đơn giản; advanced mode mở quyền kiểm soát cho power user.
5. **Không hứa hẹn portability mù.** Capability chỉ được cam kết khi format, encryption và security boundary đã được kiểm chứng.
6. **Recoverability là nền tảng.** Export, backup và rebuild quan trọng ngang với capture.
7. **Derived data không được làm mất dữ liệu chính.** Index/cache hỏng phải có fallback hoặc rebuild path.
8. **Consent rõ ràng.** Người dùng chọn loại dữ liệu được đọc/ghi; không có background collection ngầm.
9. **Provenance giữ nguyên.** Item cần biết nguồn, thời điểm, snapshot/version và lịch sử biến đổi phù hợp.
