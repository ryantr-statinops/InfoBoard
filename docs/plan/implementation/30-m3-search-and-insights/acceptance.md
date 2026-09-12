# M3 acceptance checklist

- [ ] FTS5 query sanitization, excerpts/highlights, and shared filters are correct.
- [ ] Core-mode search works without semantic dependencies.
- [ ] Full-mode Chroma/embedding/RRF returns source, score, and excerpt.
- [ ] Related/topic clusters have explicit empty/degraded states.
- [ ] DuckDB analytics matches the SQLite fallback.
- [ ] The Vietnamese/English query set meets the top-five target recorded in epic 15.

**Evidence:** retrieval evaluation, filter-parity report, and benchmark warm-up.
