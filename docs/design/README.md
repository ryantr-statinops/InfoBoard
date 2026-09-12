# InfoBoard — Product design

Design docs turn product requirements into an MVP UX specification. Mockups are layout references; details visible in an image do not become requirements automatically.

Core mode is the default experience. Full-mode controls appear only after explicit setup; an unconfigured full mode is not shown as degraded. Advanced mode exposes configuration, maintenance, diagnostics, and technical metadata without changing core data semantics.

1. [Information architecture](00-information-architecture.md)
2. [Core flows and screens](01-core-flows-and-screens.md)
3. [UI states and responsive behavior](02-ui-states-and-responsive.md)
4. [Mockup map](03-mockup-map.md)

The visual component system/design tokens are out of scope for now. Dashboard implementation details belong to epic 12.

## UX requirement map

| Requirement | Design contract | Milestone |
| --- | --- | --- |
| `PR-UX-01` | Add → list/detail → organize → keyword retrieve works without advanced setup | M1, M3 core |
| `PR-UX-02` | Query/filter context survives detail open/close | M1 list; M3 retrieval |
| `PR-UX-03` | Empty, loading, processing, failed, unavailable, and degraded states provide a next action | M1–M4 |
| `PR-UX-04` | Core and enabled optional surfaces avoid horizontal overflow at 1440 px and 390 px | M1–M4 |
| `PR-RET-03` | Semantic/hybrid, related-content, and cluster surfaces appear only for enabled full mode | M3 optional full mode |
