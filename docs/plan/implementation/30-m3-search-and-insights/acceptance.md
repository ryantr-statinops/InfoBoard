# M3 acceptance checklist

- [ ] FTS5 query sanitization, excerpts/highlights, and shared filters are correct.
- [ ] Core-mode search works without semantic dependencies.
- [ ] DuckDB analytics matches the SQLite fallback.
- [ ] Core-mode Vietnamese/English keyword query set meets the top-five target recorded in epic 15.
- [ ] If full mode is in release scope: Chroma/embedding/RRF returns source, score, and excerpt.
- [ ] If full mode is in release scope: related/topic clusters have explicit empty/degraded states.

**Evidence:** core retrieval evaluation, filter-parity report, and benchmark warm-up. Full-mode evidence is conditional on the release scope.
