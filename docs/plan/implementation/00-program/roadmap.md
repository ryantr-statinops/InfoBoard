# 02 — Master roadmap

**Status:** `ready`
**Release target:** MVP local Advanced Bookmark Manager first; post-MVP follows discovery gates

## Dependency graph

```mermaid
flowchart TD
    F[10 Foundation] --> D[11 Data + migrations]
    D --> I[13 Ingestion sources]
    D --> W[14 Index worker + cache]
    F --> U[12 Dashboard + workspace]
    U --> S[15 Search + discovery]
    W --> S
    D --> A[16 Analytics]
    D --> R[18 Reliability + recovery]
    F --> Q[17 Security + privacy]
    Q --> O[19 Observability]
    S --> T[20 Testing + performance + CI]
    A --> T
    R --> T
    T --> P[21 Packaging + release]
    P --> E1[30 Extension]
    P --> E2[31 Cloud sync]
    E2 --> E3[32 Collaboration]
    S --> E4[33 AI + providers]
    P --> E5[34 Browser portability]
    E5 --> E6[35 Migration assistant]
```

## Milestones

### M0 — Documentation and baseline

Implementation packages, gap analysis, contracts, risk register, indexes, and traceability are maintained without changing runtime.

### M1 — Usable local dashboard

Epics `10–12`: foundation, schema contract, dashboard/item workspace, text capture, collections, notes, edit, and delete.

### M2 — Reliable ingestion

Epics `13–14`: five input sources, chunk/versioning, worker retry/restart, and technical-to-UI state mapping.

### M3 — Search and insights

Epics `15–16`: FTS5, optional semantic search, RRF, related/clusters, analytics, evaluation, and fallback.

### M4 — Hardened MVP release

Epics `17–21`: security, recovery, observability, tests/benchmarks, packaging, and release.

### M5+ — Product expansion

Packages `30–35` remain discovery-only until prototype, decision, threat, compatibility, and acceptance gates pass.

## Milestone gate and priority

Do not start the next milestone until migrations are verified, the prior acceptance gate passes, the branch is merged, and blockers have an approved mitigation. Data integrity and local usability outrank semantic recall; safe fallback outranks a new dependency; measurement precedes optimization.
