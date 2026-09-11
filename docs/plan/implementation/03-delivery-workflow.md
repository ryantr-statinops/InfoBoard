# 03 — Delivery workflow

**Status:** `ready`
**Branch chính:** `main`
**Branch làm việc:** `dev`

## Chu trình feature

1. `git fetch origin`; cập nhật local `main` theo `origin/main`.
2. Fast-forward `dev` theo `main`; xác nhận worktree sạch.
3. Chọn một feature trong epic đã ở `ready` và ghi mục tiêu/acceptance.
4. Chia thành các commit nhỏ theo boundary có thể review độc lập.
5. Sau mỗi commit chạy test liên quan và `uv run ruff check .`.
6. Push từng commit lên `origin/dev`, không force-push và không merge vào `main`.
7. Khi feature hoàn tất, chạy full checks, ghi execution log và report.
8. Dừng ở trạng thái `review`; user tự tạo/review/merge PR.
9. Sau khi user báo merge, fetch `origin/main`, fast-forward `dev`, rồi mới bắt đầu feature tiếp theo.

## Quy tắc commit

- Một commit chỉ có một ý nghĩa chức năng hoặc tài liệu.
- Dùng Conventional Commits: `feat`, `fix`, `test`, `docs`, `chore`, `security`, `ci`.
- Không trộn formatting lớn với behavior change.
- Migration phải đi cùng test nâng cấp hoặc commit test kế tiếp được ghi rõ dependency.
- Commit message phải đủ rõ để tạo changelog.

## Review report tối thiểu

- Feature/epic và acceptance đã đạt.
- Danh sách commit hash theo thứ tự.
- Files/API/schema bị ảnh hưởng.
- Test, lint, benchmark và manual smoke result.
- Known limitation và rollback command.
- Trạng thái `git status`, remote branch và PR link nếu có.

## Rollback

Không dùng `git reset --hard` trên worktree của user. Rollback code qua revert PR; rollback data qua backup/migration procedure trong epic `18`. Không xóa database local khi chưa xác nhận đường dẫn và backup.

## Definition of ready cho feature

Feature có owner, input/output contract, dependency, test cases, failure behavior, acceptance và commit boundaries. Nếu thiếu một mục, giữ trạng thái `draft`.
