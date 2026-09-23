# Plans

## Product specification

[`refactor/`](refactor/README.md) is the canonical and complete InfoBoard product specification. It defines the product contract before implementation work begins.

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

[`implementation/`](implementation/README.md) contains the existing implementation-planning tree. It is retained unchanged by this rebuild and has no new scope from the refactor documents.

Do not mix product decisions into implementation slices. A product or boundary change belongs in `refactor/` first; implementation planning can be revisited only through an explicit future decision.
