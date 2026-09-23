# Verification and acceptance

## Verification layers

### Product and UX

- Validate every journey in [user experience](user-experience.md): open, query, navigate, activate, dismiss, no results, stale result, host failure, rebuild, reset, and uninstall.
- Test keyboard-only operation, focus visibility, screen-reader labels, reduced motion, text scaling, and narrow windows.
- Verify the UI never reports an unconfirmed activation or hides a recoverable failure.

### Extension integration

- Use browser fixtures for tab create/update/move/group/pin/activate/remove and multi-window projections.
- Verify service-worker suspension, browser restart, profile separation, private-context separation, permission denial, and missed-event reconciliation.
- Verify Chrome and Edge behavior separately across Linux, macOS, and Windows release environments.

### Runtime and protocol

- Test handshake, version mismatch, profile mismatch, framed messages, request IDs, timeouts, duplicate requests, out-of-order deltas, snapshot replacement, and reconnect.
- Test host absence, host crash, malformed JSON, oversized payloads, interrupted writes, and persistence failure.
- Confirm unknown protocol versions fail closed and provide repair guidance.

### Search and ranking

- Maintain fixed fixtures for exact title/domain matches, prefixes, phrases, typos, duplicate titles, multiple windows/groups, pinned tabs, active-window changes, recency decay, empty queries, stale records, Unicode, long fields, and adversarial input.
- Assert score component precedence, deterministic tie-breaking, bounded result count, and projection convergence.
- Benchmark the declared 1,000-tab envelope and record p50/p95/p99 latency and memory.

### Privacy and security

- Static permission review against requirements.
- Verify no network requests, history APIs, cookies, local storage, page-content reads, or host-side command execution occur in the normal path.
- Inspect Native Messaging manifests and origin allowlists.
- Verify logs and diagnostic exports redact URLs, titles, query strings, and tokens by default.
- Test reset/uninstall ownership boundaries and private-context cleanup.

## Acceptance criteria

The product target is accepted only when all are true:

1. A configured shortcut opens a focused search surface in Chrome and Edge.
2. A query over the supported open-tab fields returns deterministic, understandable results.
3. Keyboard selection activates the confirmed intended tab and window.
4. Tab events, service-worker restart, host reconnect, and full resync converge without duplicate or silently stale records.
5. Host, persistence, permission, and activation failures are visible, recoverable, and isolated from browser state.
6. p95 latency meets NFR-001 through NFR-003 on the declared reference environments.
7. The privacy and permission boundary passes static and runtime inspection.
8. Install, update, repair, reset, and uninstall are idempotent on all supported OS/browser pairs.
9. Accessibility checks pass for keyboard use, focus, semantics, contrast, reduced motion, and text scaling.
10. Documentation, protocol versions, release artifacts, and diagnostics describe the same product contract.

## Release gates

- Requirements traceability has no unowned `MUST`.
- Browser compatibility matrix is green or has an explicit accepted exception.
- Performance benchmark is within target with no unbounded memory growth.
- Security/privacy review has no unresolved high-severity finding.
- Recovery and uninstall tests pass after clean install, update, crash, and reset.
- Release notes identify protocol, ranking, schema, extension, host, and permission changes.
