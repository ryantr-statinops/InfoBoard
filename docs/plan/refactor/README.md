# InfoBoard product refactor

## Status

`product-specification`

This directory is the source of truth for the rebuilt InfoBoard product. The old InfoBoard documentation remains historical and does not define this product. Implementation work starts only after the documents in this directory are internally consistent and the acceptance criteria are testable.

## Product definition

InfoBoard is a Chromium desktop extension with a local companion runtime for fast, keyboard-first search and switching across the user's open browser tabs.

```text
configurable shortcut
-> focused search surface
-> exact/fuzzy query over current open tabs
-> deterministic contextual ranking
-> selected-tab activation
```

The product target is a complete, installable, recoverable desktop product. It is not framed as an MVP-first sequence. Delivery order may reduce implementation risk, but no temporary behavior is promoted to the product contract without an explicit decision.

## Binding product boundaries

- Browsers: Chrome and Edge desktop.
- Operating systems: Linux, macOS, and Windows.
- Browser data: open tabs, title, URL, domain, window, tab group, pinned state, and recent activation.
- Search: local lexical search over normalized tab metadata with deterministic context and recency ranking.
- Runtime: Manifest V3 extension plus a Go Native Messaging host that owns the in-memory index and remains resident while the connection is useful.
- Persistence: local SQLite for configuration, installation state, and bounded activation metadata; current tab state is rebuildable from the browser.
- Privacy: no cookies, local storage, network interception, whole browsing history, cloud indexing, or page-content indexing in the product contract.

## Document map

| Document | Contract |
| --- | --- |
| [Product contract](idea-brief.md) | User, problem, outcomes, capabilities, boundaries, and complete product behavior |
| [Requirements](requirements.md) | Functional, quality, privacy, compatibility, and operational requirements |
| [Browser landscape](browser-landscape.md) | Browser APIs, permissions, profiles, platform constraints, and support boundary |
| [User experience](user-experience.md) | Journeys, states, interaction rules, accessibility, and failure presentation |
| [Domain and privacy](domain-and-privacy.md) | Entities, data lifecycle, trust boundaries, privacy rules, and security invariants |
| [Search and ranking](search-and-ranking.md) | Query parsing, normalization, scoring, deterministic ordering, and explainability |
| [Architecture](architecture.md) | Components, ownership, data flow, process lifecycle, and failure isolation |
| [Runtime protocol](runtime-protocol.md) | Native Messaging frames, synchronization, requests, responses, and compatibility |
| [Persistence and lifecycle](persistence-and-lifecycle.md) | SQLite responsibilities, migrations, startup, shutdown, recovery, and uninstall |
| [Packaging and operations](packaging-and-operations.md) | Install, update, permissions, OS packaging, diagnostics, and recovery |
| [Verification and acceptance](verification-and-acceptance.md) | Test seams, measurable acceptance criteria, compatibility matrix, and release gates |
| [Roadmap](roadmap.md) | Dependency-ordered delivery slices for the complete product target |
| [Decision log](decisions.md) | Binding decisions, rejected alternatives, and revisit triggers |

## Rules for future changes

1. Update the product contract before adding a capability.
2. Every new data source must document user value, privacy impact, retention, and removal behavior.
3. Every runtime or storage choice must name its owner, failure behavior, and measurable acceptance signal.
4. Search results must remain deterministic for the same tab projection, query, and timestamp inputs.
5. Semantic retrieval, page content, history, and cloud services are separate proposals; they are not implicit dependencies.
6. Changes that weaken browser or privacy boundaries require a decision-log entry before implementation.
