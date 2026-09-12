# 04 — Contract compatibility index

**Status:** `ready`
**Role:** compatibility and implementation navigation; not a canonical contract and not runtime evidence

## Canonical sources

- [Architecture interface contracts](../../../architecture/09-interface-contracts.md) own API conventions, error envelopes, states, endpoint boundaries, service interfaces, core/full behavior, and HTMX boundaries.
- [Architecture data model](../../../architecture/02-data-model-and-erd.md) owns schema and persistence invariants.
- [Design flows and screens](../../../design/01-core-flows-and-screens.md) and [UI states](../../../design/02-ui-states-and-responsive.md) own user-visible behavior.
- [Current state and gap analysis](current-state.md) records runtime evidence and known differences from target contracts.

## Implementation ownership

| Contract area | Owning epics |
| --- | --- |
| Application lifecycle, health foundation, and settings | 10, 19 |
| Data/version/list/detail contracts | 11, 12 |
| Extraction and durable job response | 13, 14 |
| Keyword, conditional full-mode retrieval, and analytics | 15, 16 |
| Security, cleanup, recovery, verification, and packaging | 17–21 |

Epic plans and tasks describe delivery and rollout. Their execution logs are the only implementation evidence; this index must not copy target payloads or redefine canonical behavior.
