# Product roadmap

The roadmap sequences dependencies; it does not authorize implementation changes by itself. The binding sequence is [`../plan/refactor/roadmap.md`](../plan/refactor/roadmap.md).

## Delivery order

1. **Contract and boundaries** — product behavior, browser support, permissions, privacy, domain projection, and decisions.
2. **Extension projection** — profile separation, eligible-tab snapshot, browser event adapter, and activation controller.
3. **Native runtime boundary** — host registration, handshake, framed protocol, limits, synchronization, and typed errors.
4. **Lexical search path** — normalization, index, deterministic ranking, bounded result references, and model versioning.
5. **Focused UX** — keyboard journey, result states, accessibility, configuration, runtime status, and activation recovery.
6. **Persistence and lifecycle** — SQLite schema, bounded activation metadata, migrations, startup/shutdown, reset, and uninstall.
7. **Packaging and operations** — install, update, rollback, repair, diagnostics, redaction, and OS/browser release matrix.
8. **Verification and release** — browser fixtures, ranking fixtures, privacy/security checks, performance envelope, accessibility, and release gates.

Each slice must preserve the complete target boundary. Delivery order may reduce risk, but temporary behavior must not silently become a new product contract.
