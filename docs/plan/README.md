# Plans

## Product specification

[`refactor/`](refactor/README.md) is the canonical and complete InfoBoard product specification. Its contract is ready for implementation.

Its documents cover:

- product contract and complete capability boundary;
- requirements and acceptance signals;
- browser/platform and privacy/domain boundaries;
- user experience and search/ranking semantics;
- target architecture and Native Messaging protocol;
- persistence/lifecycle and packaging/operations;
- verification/acceptance, decisions, and roadmap.

Detailed reader views live in [`../architecture/`](../architecture/README.md), [`../design/`](../design/README.md), [`../operations/`](../operations/README.md), [`../product/`](../product/README.md), and [`../quality/`](../quality/README.md). They explain the canonical material without replacing it.

## Implementation plans

[`implementation/`](implementation/README.md) contains the active 20-phase implementation plans, dependency DAG, commit mappings, execution dashboard, and branch protocol. Start with IP-01; implementation status is tracked in its README.

Do not mix product decisions into implementation slices. A product or boundary change belongs in `refactor/` first; implement the approved contract by following `implementation/index.md` and its phase acceptance gates.
