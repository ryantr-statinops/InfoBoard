# Quality — Search and performance evaluation

## Retrieval evaluation

- Dùng versioned query set tối thiểu 20 query Việt/Anh, gồm synonym và có/không dấu.
- Mỗi query khai báo item đích và filters áp dụng.
- Core mode và full mode được đánh giá riêng.
- Release target: item đích trong top 5 cho ít nhất 16/20 query.
- Ghi retrieval mode, model ID/revision, index version và fixture checksum.

## Performance benchmark

- Dataset chuẩn: 1.000 items, khoảng 10.000 chunks.
- Đo search p50/p95, dashboard aggregate latency, ingestion throughput và rebuild duration.
- Search target p95 dưới một giây khi model đã tải; loại cold model download/startup khỏi con số này.
- Ghi CPU, RAM, OS, Python/dependency versions, model và số lần warm-up/sample.
- Không coi target là bảo đảm cho mọi hardware; regression so với baseline cùng môi trường phải được giải thích.

## Failure evaluation

- Chroma unavailable: keyword response vẫn thành công và báo mode.
- DuckDB unavailable: SQLite bounded fallback hoặc degraded card rõ ràng.
- Model/cache mismatch: không dùng vector/cache sai revision/dimension.
- Deleted/stale chunks: không xuất hiện trong kết quả.
