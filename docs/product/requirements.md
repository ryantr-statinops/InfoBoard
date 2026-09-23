# Product requirements

The complete requirement rows remain in the [canonical requirements document](../plan/refactor/requirements.md). This view groups them for product review.

## Search journey

- **FR-001–FR-008:** configurable command, focused input, live current-profile search, deterministic ranking, distinguishable rows, keyboard navigation, activation, and side-effect-free dismissal.
- **FR-011–FR-012:** stale-result safety and visible runtime/index freshness.

## Projection and resilience

- **FR-009–FR-010:** reconcile all supported tab events and rebuild from a full snapshot after missed events or reconnect.
- **FR-013:** optional semantic or durable-storage failure cannot remove normal lexical search.
- **FR-014–FR-015:** configuration is profile-scoped; reset/uninstall removes only InfoBoard-owned local data.

## Quality requirements

- **NFR-001–NFR-003:** p95 shortcut-to-focus, query-to-render, and selection-to-activation targets.
- **NFR-004–NFR-006:** deterministic ranking, projection convergence, and recoverable runtime state.
- **NFR-007–NFR-009:** bounded local hot path, accessibility, and privacy invariants.
- **NFR-010:** Chrome and Edge on Linux, macOS, and Windows for the declared release matrix.

## Permission and operations rules

Only permissions required by the product contract may be requested. History, bookmarks, downloads, cookies, page content, broad host access, and unrelated storage require a new decision and privacy review. Protocol requests have bounded timeouts and typed errors. Installer, updater, reset, and uninstaller behavior is idempotent. Logs default to redacted metadata.
