# Architecture — Decisions

## Current state

Các quyết định nền tảng đã xuất hiện trong product/implementation docs nhưng chưa có architecture index canonical.

## Target state — locked decisions

| Decision | Rationale | Consequence |
| --- | --- | --- |
| SQLite là system of record | Local, transactional, recoverable | Mọi derived store hydrate/rebuild từ SQLite |
| Không dùng ORM trong MVP | SQL/schema ownership minh bạch | Migration và data-access SQL phải version-controlled |
| Server-rendered Jinja2 + HTMX | Không cần SPA/Node build | Route/partial contract cần integration test |
| Core/full modes | Native/ML dependency không ổn định mọi máy | Keyword/core flow luôn có fallback |
| Sequential durable worker | Đơn giản hóa local consistency | Cần lease, checkpoint, retry và restart recovery |
| Local embedding mặc định | Privacy và offline operation | Model prepare/revision metadata bắt buộc |

## Decision lifecycle

- Product outcome thay đổi được ghi trong Internal PRD trước.
- Architecture decision mới ghi context, alternatives, decision, consequence và supersession.
- Implementation governance giữ execution-specific decision/evidence và liên kết tới đây.
- Target decision không được đánh dấu implemented nếu current-state/evidence chưa xác nhận.
