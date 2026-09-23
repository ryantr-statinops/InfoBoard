# Project context

This file is the shared language and operating context for agents working in this project. Keep it concise, current, and specific to decisions that change how work should be planned, implemented, reviewed, or reported.

## Project

- Name: InfoBoard.
- Purpose: A complete Chromium desktop product for keyboard-first search and activation of open browser tabs.
- Primary users or consumers: Desktop users who keep many tabs across windows and need fast, predictable tab switching.
- Current priority: Rebuild and complete the product specification in `docs/plan/refactor/` before implementation.

## Domain vocabulary

| Term | Meaning in this project | Avoid or distinguish from |
| --- | --- | --- |
| Tab search | Search and activate currently open browser tabs. | Whole-browser-history search. |
| Hot path | Shortcut-to-focused-input, query-to-results, and selection-to-activation behavior. | Installation, indexing, or diagnostics work. |
| Tab projection | Rebuildable local representation of eligible open tabs in one browser profile. | Persistent browsing history. |
| Native Messaging host | Registered local Go process used by the extension for indexing and ranking. | An application-owned loopback daemon. |
| Context | Window, tab group, pinned state, and recent activation signals used for ranking. | Cookies, local storage, or page contents. |

## Architecture boundaries

- System boundary: Chrome and Edge desktop on Linux, macOS, and Windows.
- Major components: Manifest V3 extension, focused search surface, browser event adapter, profile-scoped tab projection, Go Native Messaging host, in-memory lexical index, deterministic ranker, bounded SQLite repository, and local diagnostics.
- Ownership boundaries: The extension owns browser permissions, browser events, UI, and tab activation. The Go host owns local indexing, ranking, protocol state, and bounded local persistence. SQLite is not the source of truth for live tabs.
- Important data flows: Shortcut -> focused search surface -> local query over open-tab metadata -> deterministic ranked results -> confirmed tab/window activation. Browser events and full snapshots keep the projection current.
- External systems: Chrome and Edge extension APIs, Native Messaging registration, and platform installers. No cloud service or loopback listener is required by the product contract.

## Naming and conventions

- Naming conventions: Use product terms from this file; name artifacts under `docs/agent/` by decision or workflow purpose.
- Preferred patterns: Explicit requirements, bounded local data, deterministic behavior, versioned protocols, measurable acceptance criteria, and recoverable failure states.
- Patterns to avoid: MVP-first scope reduction, database-first design, broad browser permissions, cloud indexing, page-content collection, and copying retired InfoBoard contracts into the rebuilt product.
- Relevant project instructions: The product source of truth is `docs/plan/refactor/`. Historical material remains under `docs/product/`, `docs/architecture/`, and `docs/plan/implementation/` and is not an active product contract.

## Testing and verification

- Primary test seams: Browser event reconciliation, shortcut/search/activation journeys, deterministic ranking fixtures, Native Messaging protocol and recovery, lifecycle/packaging, privacy boundaries, accessibility, and cross-platform compatibility.
- Required checks: Every binding requirement must have a test seam and observable acceptance signal. The hot path must remain local, bounded, and measurable.
- Test commands: Use implementation-specific commands once runtime code exists. Documentation validation uses local-link checking, `git diff --check`, requirements traceability, and the acceptance criteria in `docs/plan/refactor/verification-and-acceptance.md`.
- Independent sources of expected results: Chrome and Edge APIs, browser fixtures, measured latency and memory, protocol contracts, privacy review, and the rebuilt refactor documents.

## Issue tracking and decisions

- Issue tracker: GitHub issues for implementation work when opened.
- Issue location or command: Repository issues and `docs/plan/refactor/decisions.md` for product decisions.
- ADR location: `docs/agent/decisions/` for agent-produced research decisions; product decisions remain in `docs/plan/refactor/decisions.md`.
- Decision naming convention: Lowercase hyphenated topic names ending in `.md` for agent artifacts; `REF-NNN` for product decisions.

## Work artifacts

- Artifact root: `docs/agent/`
- Feature specifications: `docs/agent/specs/`
- Bug diagnostics: `docs/agent/diagnostics/`
- Research decisions: `docs/agent/decisions/`
- Analysis reports: `docs/agent/reports/`
- Handoffs: `docs/agent/handoffs/`

## Constraints and assumptions

- Runtime or deployment constraints: The target runtime is a Go Native Messaging host. It may keep memory state while connected, but the product does not rely on permanent process residency. Native host installers are required for Linux, macOS, and Windows.
- Compatibility requirements: Chrome and Edge desktop are supported. Other browsers and mobile are separate product proposals.
- Security or privacy constraints: Initial and complete product data is limited to open tabs, title, URL, domain, window, tab group, pinned state, and bounded recent activation metadata. Do not collect cookies, local storage, network-interception data, page content, whole history, or cloud copies.
- Resource or cost constraints: The hot path uses in-memory lexical search. SQLite is bounded local storage. Semantic retrieval is not a normal-path dependency.
- Known assumptions: The full product specification is being rebuilt before implementation; all installed SKILLS entries are available for discovery, but only the smallest task-matching skill set should be activated.

## Preferred language

- Use these terms: Chromium desktop, open-tab metadata, tab projection, hot path, Native Messaging host, deterministic ranking, bounded, measurable, recoverable.
- Avoid these terms: whole-history search, database-first design, retired InfoBoard contracts, MVP-first.
- Explain these terms on first use: Native Messaging host, tab projection, and any future semantic/embedding layer.

## Agent boundaries

- Confirm before: Expanding browser or OS support, collecting a new data class, adding cloud services, changing the Native Messaging boundary, or weakening the privacy contract.
- Never do without explicit authorization: Access cookies or local storage, intercept network traffic, index whole browsing history or page contents, publish secrets, or perform destructive repository changes.
- Report after completion: Changed files, active skill routes, validation commands and results, decisions, acceptance gaps, and remaining product risks.
