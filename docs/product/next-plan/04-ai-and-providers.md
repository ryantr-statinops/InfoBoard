# AI and providers

**Status:** `discovery`

## Outcome

Người dùng có thể dùng AI để hiểu hoặc khai thác knowledge base khi có giá trị rõ ràng, trong khi vẫn kiểm soát provider, dữ liệu gửi đi và chi phí.

## Candidate capabilities

- Summary hoặc structured extraction theo yêu cầu.
- Ask/chat với dữ liệu có citation tới item/snapshot.
- Recommendation hoặc grouping có thể giải thích.
- Provider ecosystem cho embedding và generation local/cloud.

## Assumptions

- Keyword retrieval và core knowledge workflow không phụ thuộc AI.
- Cloud provider luôn opt-in; local provider được ưu tiên khi khả thi.
- Output AI không tự động ghi đè dữ liệu gốc.

## Risks

Hallucination, privacy leakage, prompt injection từ imported content, model/provider drift, chi phí không kiểm soát và kết quả không có provenance.

## Dependencies

Retrieval quality, provider interface, consent/configuration UX, data minimization, citation model, evaluation set, budget controls và prompt-injection threat model.

## Implementation-ready gate

Chọn một user outcome ưu tiên, baseline không AI, evaluation metrics, source citation behavior, provider/privacy matrix, cost limits, failure fallback và explicit write-back policy.
