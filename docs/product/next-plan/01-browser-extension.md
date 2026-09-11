# Browser extension

**Status:** `discovery`

## Outcome

Người dùng lưu trang, selection hoặc bookmark vào InfoBoard ngay trong browser với provenance rõ và feedback đáng tin cậy.

## Assumptions

- Extension là source adapter, không sở hữu ingestion/indexing pipeline riêng.
- Save page phải hoạt động với local InfoBoard và báo rõ khi app chưa chạy.
- Permission được yêu cầu tối thiểu và giải thích theo hành động.

## Risks

- Permission quá rộng làm giảm niềm tin.
- Nội dung trang chứa script, secret hoặc dữ liệu private.
- Payload/API drift giữa extension và local app.
- Store review và browser APIs khác nhau giữa Chrome/Firefox.

## Dependencies

Ổn định URL/page ingestion contract, local handshake, payload versioning, authentication/consent boundary, retry/offline behavior và browser distribution strategy.

## Implementation-ready gate

Chốt browser matrix, capture modes, permission model, local connection security, sanitized payload, compatibility policy, telemetry boundary và manual/store distribution acceptance.
