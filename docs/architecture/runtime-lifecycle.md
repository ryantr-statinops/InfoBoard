# Runtime lifecycle

## State machine

```text
Disconnected -> Connecting -> Handshaking -> Synchronizing -> Ready
      ▲             │              │              │          │
      └─────────────┴──────────────┴──────────────┴──────────┘
                         recoverable failure
```

- **Disconnected** — no host connection; the surface can explain unavailable search and offer retry or repair.
- **Connecting** — the client starts or reaches the registered host and applies a startup timeout.
- **Handshaking** — protocol version, extension version, browser family/version, profile identity, capabilities, limits, ranking model, and persistence health are checked.
- **Synchronizing** — a complete snapshot or resync is being applied; results tied to an unknown revision cannot activate.
- **Ready** — bounded queries and ordered deltas are accepted.
- **Recovering** — reconnect backoff and snapshot rebuild occur after a crash, restart, timeout, or sequence gap.

## Startup sequence

1. The extension opens its profile-scoped browser adapter.
2. It obtains a complete eligible-tab snapshot and projection revision.
3. The Native Messaging client connects to the registered Go host.
4. The client and host complete the versioned handshake.
5. The extension sends the snapshot with profile identity and revision.
6. The host atomically replaces its projection, builds the index, and acknowledges readiness.
7. The UI exposes search only against the acknowledged revision.

## Steady state

Browser events become ordered deltas. Each query carries the projection revision known by the client. The host returns result references, bounded display metadata, ranking-model version, revision, and timing counters. The extension validates the revision before rendering or activating.

## Recovery rules

- Missing or out-of-order deltas trigger a full snapshot.
- Service-worker restart reconnects and reconciles instead of assuming the old projection is valid.
- Host failure affects search availability, not browser tabs or browser navigation.
- SQLite failure affects persistence health, not the in-memory lexical path when the host can continue safely.
- An incompatible handshake stops tab-data transmission and exposes repair guidance.

The [runtime protocol](../plan/refactor/runtime-protocol.md) defines the wire-level messages and failure classes.
