# Internal PRD — Success criteria

## User success

- Hoàn thành add → organize → open → note → search mà không cần advanced setup.
- Empty, processing, failed và degraded states đều chỉ ra bước tiếp theo.
- Filter context được giữ khi mở/đóng detail.

## Functional acceptance

- Năm nguồn MVP tạo snapshot/chunk đúng và dữ liệu tồn tại sau restart.
- Collections nhiều-nhiều, notes, status, edit và soft-delete đúng semantics.
- Keyword search hoạt động khi semantic, cache hoặc analytics unavailable.
- Sửa text, retry hoặc đổi model không trả chunks cũ và không làm mất note.
- Backup/restore khôi phục dữ liệu chính và rebuild index thành công.

## Quality targets

- Core-mode keyword search Việt/Anh có item đích trong top 5 cho ít nhất 16/20 query chuẩn.
- Nếu phát hành full mode, semantic/hybrid retrieval được đánh giá bằng query set riêng và phải đạt gate đã công bố cho full mode.
- Analytics khớp dữ liệu chính và degraded fallback không làm core UI hỏng.
- Desktop 1440 px và mobile 390 px không tràn ngang trong core flow.
- Với 1.000 item/10.000 chunks, search p95 dưới một giây khi model đã tải; cấu hình máy được ghi lại.
- Security suite bao phủ limits, unsafe URL/redirect, escaping và origin/host boundary.

## Release acceptance

- Mọi core-MVP requirement trong [Product requirements](02-product-requirements.md) có implementation evidence và test; future requirements có discovery decision riêng.
- Requirement traceability của core release không còn `missing` hoặc `partial`, trừ waiver được ghi nhận. `PR-RET-03` và implementation evidence tương ứng chỉ trở thành điều kiện bắt buộc nếu full mode được đưa vào release scope.
- Data integrity, fallback, security, recovery, lint, test và benchmark gates pass.
- README, runbook và behavior thực tế thống nhất.
- Capability trong Next Plan không trở thành release blocker của M1–M4.
