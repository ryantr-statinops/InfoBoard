# Complete product delivery roadmap

This roadmap sequences implementation dependencies for the complete product contract. It is not an MVP ladder and does not redefine the product as a series of reduced products. Each stage produces a coherent, testable part of the final system.

## Stage 1 — Contract and fixtures

- Freeze the product contract, requirements, data boundary, privacy rules, and decision log.
- Create browser, projection, ranking, protocol, and failure fixtures from the acceptance criteria.
- Define reference environments and measurement harnesses.

**Exit:** every binding requirement has an owner, test seam, and documented acceptance signal.

## Stage 2 — Extension browser boundary

- Implement Manifest V3 command registration and focused search surface shell.
- Implement profile-scoped tab projection and all declared browser events.
- Implement activation, stale-result handling, multi-window behavior, and private-context rules.

**Exit:** the extension can reconcile browser state and complete the UX journey with a local fixture/runtime seam.

## Stage 3 — Go host and protocol

- Implement the Native Messaging host, handshake, limits, request IDs, sync messages, health, errors, reconnect, and full snapshot rebuild.
- Implement in-memory projection and deterministic lexical ranking.
- Connect the extension to the host and measure hot-path latency.

**Exit:** protocol, runtime failure, convergence, and ranking acceptance tests pass at the declared tab envelope.

## Stage 4 — Complete UX and configuration

- Implement all UI states, keyboard behavior, accessibility, display privacy rules, settings, status, repair, and reset flows.
- Add ranking explanations and diagnostics controls without exposing sensitive fields by default.

**Exit:** all user journeys and accessibility criteria pass in Chrome and Edge.

## Stage 5 — Persistence and lifecycle

- Implement SQLite schema, migrations, bounded activation metadata, diagnostics retention, degraded persistence, reset, and uninstall cleanup.
- Verify restart, crash, migration failure, and recovery behavior.

**Exit:** local data lifecycle and recovery acceptance pass without affecting browser state.

## Stage 6 — Cross-platform packaging

- Build Chrome and Edge packages plus Linux, macOS, and Windows host installers.
- Implement origin allowlists, integrity checks, update/rollback, repair, and uninstall verification.
- Run the complete compatibility matrix.

**Exit:** clean install, update, repair, reset, uninstall, and rollback scenarios pass on every supported pair.

## Stage 7 — Release hardening

- Complete security/privacy/permission review.
- Run performance regression and resource-bound benchmarks.
- Validate diagnostics redaction, accessibility, release artifacts, documentation, and support procedures.

**Exit:** all release gates in [verification and acceptance](verification-and-acceptance.md) pass.

## Explicitly deferred proposals

Semantic retrieval, page-content indexing, whole-history search, cloud synchronization, other browsers, and a standalone loopback daemon are separate proposals. They cannot be added as implementation shortcuts; each requires a product, privacy, architecture, and acceptance decision.
