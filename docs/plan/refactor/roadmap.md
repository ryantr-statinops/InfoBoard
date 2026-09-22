# Refactor roadmap

This is a discovery roadmap, not an implementation commitment.

## Phase 0 — Product boundary

- Confirm tab search as the first product direction.
- Define the primary user and the exact tab-switching problem.
- Decide whether a warm Go process is acceptable.
- Define Chromium desktop and OS support boundaries.

## Phase 1 — Browser interaction proof

- Register a configurable command shortcut.
- Read the current open-tab set.
- Open and focus the search UI.
- Switch to a selected tab.
- Handle tab creation, update, activation, and removal.

## Phase 2 — Fast lexical search

- Build the Go runtime boundary.
- Add in-memory indexing and fuzzy matching.
- Add recency/window/group/pinned ranking.
- Measure hot-path latency and memory usage.
- Add reconnect and full-index rebuild behavior.

## Phase 3 — Persistence and packaging

- Decide whether SQLite persistence is needed.
- Implement daemon or Native Messaging installation.
- Define startup, shutdown, update, crash recovery, and uninstall behavior.
- Test multiple windows and browser profiles.

## Phase 4 — Optional semantic layer

- Define a user problem that lexical search cannot solve.
- Benchmark whether ChromaDB improves that behavior.
- Add asynchronous embeddings and revisioned semantic index only if evidence supports it.
- Keep semantic failure isolated from normal tab switching.

## Phase 5 — Release hardening

- Permission review.
- Security review for local IPC.
- Chromium compatibility matrix.
- UI customization boundary.
- Performance regression benchmark.

No database or feature is promoted into the core stack without passing through the product boundary and a measured acceptance signal.
