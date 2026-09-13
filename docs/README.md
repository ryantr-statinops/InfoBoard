# InfoBoard documentation

InfoBoard is a private, local-first knowledge bookmark manager. The documentation follows the decision flow from product intent to verified delivery:

```text
Product -> Design -> Architecture -> Quality/Operations -> Implementation
```

## Reading paths

- Product work: [product vision](product/vision.md) → [MVP scope](product/mvp-scope.md) → [requirements](product/requirements.md) → [feature specifications](product/README.md).
- UX work: [design overview](design/README.md) → [information architecture](design/information-architecture.md) → [core flows](design/core-flows.md) → [UI states](design/ui-states.md).
- Engineering: product path → [architecture overview](architecture/README.md) → [API contracts](architecture/api-contracts.md) → [implementation roadmap](plan/implementation/00-program/roadmap.md).
- QA/release: [test strategy](quality/test-strategy.md) → [MVP acceptance](quality/mvp-acceptance.md) → [implementation review index](plan/implementation/index.md).
- Maintainers: [configuration](operations/configuration.md) → [backup, restore, and rebuild](operations/backup-restore-and-rebuild.md).
- New ideas: [idea inbox](product/ideas.md) → [product decisions](product/decisions.md); an unaccepted idea is not a requirement or roadmap commitment.

## Source-of-truth boundaries

| Layer | Owns | Must not own |
| --- | --- | --- |
| [Product](product/README.md) | Goal, user, scope, requirements, feature behavior, and decisions | Storage/API implementation or delivery status |
| [Design](design/README.md) | Navigation, user flows, visible states, and responsive/accessibility behavior | Product scope or database contracts |
| [Architecture](architecture/README.md) | System boundaries, schema, stores, pipelines, APIs, security, and recovery invariants | Claims of implemented behavior |
| [Quality](quality/README.md) | Test strategy, evaluation, and MVP acceptance evidence required | Product commitments or target implementation |
| [Operations](operations/README.md) | Safe configuration, backup, restore, rebuild, and health interpretation | Unsupported commands presented as verified |
| [Implementation](plan/implementation/README.md) | Current-state gaps, dependency order, delivery tasks, status, and evidence links | Redefinition of upstream behavior/contracts |

## Architectural baseline

- SQLite canonical tables own user data; FTS5 is a rebuildable keyword index stored in the same database file.
- RocksDB through `rocksdict` is a rebuildable embedding cache.
- ChromaDB is a rebuildable semantic vector index.
- DuckDB is a rebuildable analytics projection with bounded SQLite fallback.
- A self-hosted OpenAI-compatible endpoint supplies embeddings after explicit installation-level consent.
- Semantic retrieval is required for MVP release; runtime provider failure degrades safely to local keyword retrieval.

Documentation is a target contract, not runtime evidence. A capability is complete only when its implementation task and required quality evidence are verified.
