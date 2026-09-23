# InfoBoard documentation

The documentation keeps the existing folder names while separating canonical product contracts from detailed reader views and future implementation plans.

## Source of truth

- [`plan/refactor/`](plan/refactor/README.md) — binding product specification. It is the authority for product behavior, requirements, browser/platform boundaries, privacy, UX, ranking, architecture, runtime, persistence, operations, verification, decisions, and roadmap.
- [`plan/implementation/`](plan/implementation/README.md) — active implementation plans, dependency DAG, commit mappings, and execution tracker. IP-01 is the kickoff phase.

The detailed documents in the folders below explain the same contracts for different review concerns. They must link back to the canonical source and must not create conflicting behavior.

## Folder map

| Folder | Detailed responsibility |
| --- | --- |
| [`agent/`](agent/skill-integration.md) | Coding-agent skill routing and documentation boundaries; not product behavior |
| [`architecture/`](architecture/README.md) | System shape, component ownership, runtime lifecycle, data flow, and failure isolation |
| [`design/`](design/README.md) | Search surface, interaction states, accessibility, ranking behavior, and optional mockups |
| [`operations/`](operations/README.md) | Packaging, installation, update, rollback, repair, diagnostics, reset, and uninstall |
| [`plan/`](plan/README.md) | Canonical product specification plus the retained implementation-plan tree |
| [`product/`](product/README.md) | User problem, domain/data model, requirements, decisions, and delivery roadmap |
| [`quality/`](quality/README.md) | Acceptance matrix, test seams, compatibility/performance, privacy, and security evidence |

## Change rules

1. Update the canonical refactor document before adding a capability or weakening a boundary.
2. Update the relevant detailed view only as a synchronized explanation of that canonical decision.
3. Every new data source documents value, privacy impact, retention, removal, ownership, and acceptance.
4. Implement only work authorized by the assigned phase plan after its dependencies are integrated. Use the README tracker and index branch protocol.
