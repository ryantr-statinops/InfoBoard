# Internal PRD — Success criteria

## User success

- Hoàn thành add → organize → open → note → search mà không cần advanced setup.
- Empty, processing, failed và degraded states đều chỉ ra bước tiếp theo.
- Filter context được giữ khi mở/đóng detail.

## Core functional acceptance

- Năm nguồn MVP tạo snapshot/chunk đúng và dữ liệu tồn tại sau restart.
- Collections nhiều-nhiều, notes, status, edit và soft-delete đúng semantics.
- Keyword search hoạt động khi semantic, cache hoặc analytics unavailable.
- Sửa text, retry hoặc đổi model không trả chunks cũ và không làm mất note.
- Backup/restore khôi phục dữ liệu chính và rebuild index thành công.

## Core quality targets

- Search Việt/Anh có item đích trong top 5 cho ít nhất 16/20 query chuẩn.
- Analytics khớp dữ liệu chính và degraded fallback không làm core UI hỏng.
- Desktop 1440 px và mobile 390 px không tràn ngang trong core flow.
- Với 1.000 item/10.000 chunks, core keyword search p95 dưới một giây không cần model; cấu hình máy được ghi lại.
- Security suite bao phủ limits, unsafe URL/redirect, escaping và origin/host boundary.

## Core release acceptance

- Mọi core requirement trong [Product requirements](02-product-requirements.md) có implementation evidence và test.
- Requirement traceability của core scope không còn `missing` hoặc `partial`, trừ waiver được ghi nhận.
- Data integrity, fallback, security, recovery, lint, test và benchmark gates pass.
- README, runbook và behavior thực tế thống nhất.
- Capability trong Next Plan không trở thành release blocker của M1–M4.

## Conditional full-mode acceptance

Chỉ áp dụng khi một release chủ động công bố hỗ trợ full mode:

- Semantic/hybrid retrieval, related items và các derived indexes có smoke/evaluation evidence.
- Full-mode dependency, model readiness và degraded behavior được kiểm tra trên clean setup.
- Full-mode failure không làm mất core keyword/read flow.
