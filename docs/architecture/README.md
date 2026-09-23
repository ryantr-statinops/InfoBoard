# Architecture

This folder explains how the product contract is realized as cooperating boundaries. The binding source remains [`docs/plan/refactor/architecture.md`](../plan/refactor/architecture.md); these documents provide a reader-oriented breakdown and must not introduce alternate ownership.

## Reading order

1. [System overview](system-overview.md) — process shape and authority boundaries.
2. [Component ownership](components-and-ownership.md) — responsibilities and forbidden coupling.
3. [Runtime lifecycle](runtime-lifecycle.md) — connection, synchronization, readiness, and recovery.
4. [Data and failure boundaries](data-and-failure-boundaries.md) — data flow, failure isolation, and scale envelope.

Related contracts:

- [Runtime protocol](../plan/refactor/runtime-protocol.md)
- [Persistence and lifecycle](../plan/refactor/persistence-and-lifecycle.md)
- [Domain and privacy](../plan/refactor/domain-and-privacy.md)
- [Browser landscape](../plan/refactor/browser-landscape.md)
