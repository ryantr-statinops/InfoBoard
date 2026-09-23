# Project context

This file is the shared language and operating context for agents working in this project. Keep it concise, current, and specific to decisions that change how work should be planned, implemented, reviewed, or reported.

## Project

- Name: Chromium-first instant tab search extension.
- Purpose: Open a configurable shortcut, search current tabs by exact or fuzzy text, rank results by match and recency/context, and switch to the selected tab.
- Primary users or consumers: Desktop Chromium users who need fast keyboard-driven tab switching.
- Current priority: Product discovery and runtime-boundary decisions under `docs/plan/refactor/`.

## Domain vocabulary

| Term | Meaning in this project | Avoid or distinguish from |
| --- | --- | --- |
| Tab search | Search and switch among currently open browser tabs. | Whole-browser-history search. |
| Hot path | Shortcut-to-results-to-tab-switch interaction. | Background indexing or semantic enrichment. |
| Warm runtime | A resident Go process or Native Messaging host kept ready for low-latency queries. | A process spawned for every shortcut. |
| Context | Window, tab group, pinned state, and recent activation signals used for ranking. | Cookies, local storage, or browsing-history content. |

## Architecture boundaries

- System boundary: Chromium desktop extension first; Chrome and Edge are the initial browser targets.
- Major components: Extension shortcut/search UI, browser-tab state collector, local ranking/search path, optional warm Go or Native Messaging runtime, and optional asynchronous persistence/enrichment.
- Ownership boundaries: The extension owns browser integration and tab switching. The local runtime owns bounded query/ranking work when used. SQLite is persistence only when a concrete product behavior requires it. ChromaDB, RocksDB, and DuckDB remain optional layers and must not block normal tab switching.
- Important data flows: Shortcut -> instant search UI -> query over open-tab metadata -> ranked results -> selected-tab activation. Background collection may maintain title, URL, domain, window, tab group, pinned state, and recent activation metadata.
- External systems: Chromium extension APIs, Chrome, Edge, and any explicitly selected local runtime or storage dependency.

## Naming and conventions

- Naming conventions: Use product terms from this file; name artifacts under `docs/agent/` by decision or workflow purpose.
- Preferred patterns: Small, measurable local hot paths; explicit boundaries; decision records before implementation work.
- Patterns to avoid: Copying retired InfoBoard requirements, APIs, schemas, or migration plans into this product; adding a database or semantic layer without a concrete behavior.
- Relevant project instructions: Active planning documents are under `docs/plan/refactor/`. Historical material is under `docs/product/`, `docs/architecture/`, and `docs/plan/implementation/`; those paths are context only, not active product contracts.

## Testing and verification

- Primary test seams: Shortcut-to-search-to-switch latency, ranking behavior over open-tab metadata, browser API boundary behavior, and runtime discovery/communication.
- Required checks: Keep the hot path local, bounded, and measurable; verify browser compatibility at the supported boundary; validate artifact links and generated skill metadata when the skill library changes.
- Test commands: Use the commands documented by the active implementation once runtime code exists. For the installed skill registry, run the pinned-source validation commands recorded in `docs/agent/skill-integration.md`.
- Independent sources of expected results: Chromium behavior, measured latency/ranking scenarios, active refactor decisions, and pinned SKILLS manifests.

## Issue tracking and decisions

- Issue tracker: GitHub issues for implementation work when opened.
- Issue location or command: Repository issues and `docs/plan/refactor/decisions.md` for current product decisions.
- ADR location: `docs/agent/decisions/` for agent-produced research decisions; current refactor decisions remain in `docs/plan/refactor/decisions.md`.
- Decision naming convention: Lowercase hyphenated topic names ending in `.md`.

## Work artifacts

- Artifact root: `docs/agent/`
- Feature specifications: `docs/agent/specs/`
- Bug diagnostics: `docs/agent/diagnostics/`
- Research decisions: `docs/agent/decisions/`
- Analysis reports: `docs/agent/reports/`
- Handoffs: `docs/agent/handoffs/`

## Constraints and assumptions

- Runtime or deployment constraints: Chromium desktop first; the hot path must remain local and bounded. A warm Go runtime or stable Native Messaging host is a candidate, not a settled decision.
- Compatibility requirements: Start with Chrome and Edge; do not broaden browser support before the first product boundary is validated.
- Security or privacy constraints: Initial data is limited to open tabs, title, URL, domain, window, tab group, pinned state, and recent activation. Do not collect cookies, local storage, network-interception data, or whole-history content in the first product scope.
- Resource or cost constraints: Optional ChromaDB/RocksDB/DuckDB layers are asynchronous and must not block ordinary tab switching.
- Known assumptions: All SKILLS entries are installed for discovery, but only the smallest skill set matching a task should be activated; installation does not imply loading every skill into every task.

## Preferred language

- Use these terms: Chromium-first, open-tab metadata, hot path, warm runtime, Native Messaging, bounded, measurable.
- Avoid these terms: whole-history search, database-first design, retired InfoBoard contracts.
- Explain these terms on first use: Native Messaging host and any semantic/embedding layer introduced later.

## Agent boundaries

- Confirm before: Expanding browser scope, collecting data outside the initial boundary, selecting a persistent/semantic backend, or changing the active product direction.
- Never do without explicit authorization: Access cookies or local storage, intercept network traffic, index whole browsing history, publish secrets, or perform destructive repository changes.
- Report after completion: Changed files, active skill routes, validation commands and results, decisions, and any remaining boundary or runtime risk.
