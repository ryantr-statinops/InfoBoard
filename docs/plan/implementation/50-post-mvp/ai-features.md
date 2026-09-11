# 33 — AI features và providers

**Status:** `discovery`
**Canonical product reference:** [AI and providers](../../../product/next-plan/04-ai-and-providers.md)
**Milestone:** M5–M7
**Dependencies:** 15, 17, 19, 21

## Product outcome

Cung cấp summary, hỏi đáp hoặc recommendation có kiểm soát, minh bạch nguồn và không biến AI thành dependency bắt buộc của core library.

## Discovery questions

- Use case ưu tiên là summary, Q&A trên item, hay recommendation?
- Output cần citation tới chunk nào và người dùng chỉnh/sửa được không?
- Local model có đủ chất lượng/tài nguyên hay cần cloud opt-in?
- Consent, redaction, cost budget, retention và prompt injection xử lý thế nào?
- Evaluation set Việt/Anh và ngưỡng quality nào cho phép release?

## Experiments

Tạo offline evaluation set có expected facts/citations; so sánh local/cloud provider qua adapter giả; đo factuality, citation coverage, latency, token cost và privacy leakage. Không gửi dữ liệu cloud khi chưa có explicit setting.

## Provider contract direction

Provider phải có model ID/revision, token/cost metadata, timeout, cancellation, redaction hook và deterministic test fake. Prompt/template version được lưu cùng kết quả nếu kết quả được persist.

## Implementation-ready gate

Cần use case/UX rõ, consent copy, threat model prompt injection, provider SLA/cost, evaluation threshold, fallback và data retention. AI không được ghi đè canonical content.

## Execution log

Chưa discovery.
