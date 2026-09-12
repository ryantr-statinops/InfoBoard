# InfoBoard — Architecture

Architecture docs describe system structure and technical invariants. Each document distinguishes:

- **Current state:** capabilities present in the current runtime snapshot.
- **Target state:** intended design; not evidence of completion.

## Reading order

1. [System overview](00-system-overview.md)
2. [Tech stack](01-tech-stack.md)
3. [Data model and ERD](02-data-model-and-erd.md)
4. [Ingestion pipeline](03-ingestion-pipeline.md)
5. [Indexing pipeline](04-indexing-pipeline.md)
6. [Search and analytics](05-search-and-analytics.md)
7. [Lifecycle and recovery](06-lifecycle-and-recovery.md)
8. [Security boundaries](07-security-boundaries.md)
9. [Architecture decisions](08-architecture-decisions.md)

Implementation status and evidence live in [`docs/plan/implementation/`](../plan/implementation/) and must not be inferred from target diagrams.
