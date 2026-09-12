# 02 — Master roadmap

**Status:** `ready`
**Release target:** MVP local Advanced Bookmark Manager first; post-MVP follows discovery gates

## Dependency graph

```mermaid
flowchart TD
    F[10 Foundation] --> D[11 Data + migrations]
    F --> U[12 Dashboard + workspace]
    D --> U
    F --> I[13 Ingestion sources]
    D --> I
    D --> W[14 Index worker + cache]
    I --> W
    D --> S[15 Search + discovery]
    U --> S
    W --> S
    D --> A[16 Analytics]
    U --> A
    S --> A
    F --> Q[17 Security + privacy]
    I --> Q
    D --> R[18 Reliability + recovery]
    W --> R
    F --> O[19 Observability]
    W --> O
    S --> T[20 Testing + performance + CI]
    A --> T
    Q --> T
    R --> T
    F --> P[21 Packaging + release]
    S --> P
    R --> P
    T --> P
    U --> E1[30 Extension]
    I --> E1
    Q --> E1
    P --> E1
    R --> E2[31 Cloud sync]
    P --> E2
    E2 --> E3[32 Collaboration]
    S --> E4[33 AI + providers]
    Q --> E4
    T --> E4
    Q --> E5[34 Browser portability]
    R --> E5
    P --> E5
    R --> E6[35 Migration assistant]
    P --> E6
    E5 --> E6
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
