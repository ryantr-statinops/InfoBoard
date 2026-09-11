# M3 acceptance checklist

- [ ] FTS5 query sanitize, excerpt/highlight và shared filters đúng.
- [ ] Core mode search chạy khi thiếu semantic dependencies.
- [ ] Full mode Chroma/embedding/RRF trả source, score và excerpt.
- [ ] Related/topic clusters có empty/degraded state.
- [ ] Analytics DuckDB khớp SQLite fallback.
- [ ] Bộ query Việt/Anh đạt target top-5 đã ghi trong epic 15.

**Evidence:** retrieval evaluation, filter parity report và benchmark warm-up.
