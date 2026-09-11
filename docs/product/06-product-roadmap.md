# InfoBoard — Product roadmap

Roadmap này mô tả outcome sản phẩm; roadmap triển khai hiện tại vẫn là M1–M4 trong [`docs/plan/implementation/`](../plan/implementation/).

## Foundation — Local knowledge hub

Hoàn thiện capture, collections, notes, snapshot, dashboard, keyword search, reliable local storage và recovery. Đây là phạm vi MVP hiện tại.

## Capture expansion — Browser và nguồn dữ liệu

Thêm bookmark import, browser source adapters, page capture và export nội dung/metadata. Mục tiêu là đưa dữ liệu có giá trị vào InfoBoard với provenance rõ ràng.

## Portability assistant — Import/export có hướng dẫn

Cung cấp preview, mapping, conflict handling và migration report cho từng lớp dữ liệu portable. Simple mode ưu tiên one-click; advanced mode cho phép power user kiểm soát sâu.

## Advanced portability — Dữ liệu nhạy cảm và browser-specific

Nghiên cứu credentials, session state, localStorage, IndexedDB và các storage khác sau security review, compatibility matrix, consent design và recovery verification. Không tự động đưa các capability này vào MVP.

## Ecosystem

Browser extension, cloud sync, identity/collaboration và AI/provider integrations chỉ chuyển từ discovery sang implementation khi đã có outcome, data boundary, risk gate, fallback và acceptance rõ ràng.

## Decision gate cho capability mới

Mỗi capability phải trả lời: user value là gì, dữ liệu nào bị ảnh hưởng, consent/privacy boundary ra sao, fallback khi không hỗ trợ là gì, và bằng chứng nào cho phép chuyển sang implementation planning.
