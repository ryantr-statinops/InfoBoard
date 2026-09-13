# Implementation index

This is the review dashboard for the implementation program. Package checklists are authoritative; this file is a rollup and must be updated in the same commit as a package status transition.

## Status legend

`not_started` → `in_progress` → `done`. Use `blocked` when progress cannot continue and record the recovery context in the package execution log.

## Delivery index

| Milestone | Packages | Status | Prerequisites | Exit gate |
| --- | ---: | --- | --- | --- |
| M10 Foundation | 3 | not_started | None | Migration and clean-install evidence |
| M20 Capture | 3 | not_started | M10 | CAP requirements verified |
| M30 Organization and dashboard | 3 | not_started | M10, M20 snapshot projection | ORG and core UX verified |
| M40 Keyword retrieval | 2 | not_started | M20, M30 | Local retrieval verified |
| M50 Semantic provider | 2 | not_started | M20, M40 query contract | Consent and provider smoke verified |
| M60 Vector and hybrid search | 2 | not_started | M40, M50 | Semantic, hybrid, and fallback verified |
| M70 Embedding cache | 1 | not_started | M50 | Cache isolation and recovery verified |
| M80 Analytics | 2 | not_started | M30, M40 | DuckDB/SQLite parity verified |
| M90 Reliability | 3 | not_started | M20–M80 | Security and recovery suites verified |
| M99 Release | 3 | not_started | M90 | All MVP acceptance evidence complete |

## Review views

- Delivery sequence and gates: [roadmap](00-program/roadmap.md)
- Package dependencies: [dependency map](00-program/dependency-map.md)
- Requirement coverage: [traceability matrix](00-program/traceability.md)
- Program rules: [conventions](00-program/conventions.md)
- Final audit: [review checklist](00-program/review-checklist.md)

## Release gates

- [ ] All required package tasks are complete and evidenced.
- [ ] All milestone acceptance files are satisfied.
- [ ] Product requirement coverage has no gaps.
- [ ] Canonical and compatibility API behavior passes.
- [ ] Backup, restore, rebuild, privacy, and security evidence passes.
- [ ] Clean install, upgrade, rollback, UI, performance, and provider smoke evidence passes.
