# Target architecture

## System shape

```text
Chrome/Edge extension
  ├─ command and focused search surface
  ├─ browser event adapter
  ├─ profile-scoped tab projection
  ├─ activation controller
  └─ Native Messaging client
          │ framed local messages
          ▼
Go Native Messaging host
  ├─ protocol/session manager
  ├─ projection reconciler
  ├─ in-memory lexical index
  ├─ deterministic ranking engine
  ├─ bounded SQLite repository
  └─ diagnostics and health
```

The browser extension owns browser authority. The Go host owns local computation and bounded local persistence. Neither component owns data outside the product contract.

## Component ownership

| Component | Owns | Must not own |
| --- | --- | --- |
| Command/search surface | Focus, query input, selection, status, accessible presentation. | Ranking policy, persistent tab data, browser activation assumptions. |
| Browser event adapter | API permissions, event conversion, full snapshot, activation calls. | SQL, process startup policy, ranking weights. |
| Tab projection | Profile-scoped current records and revisions. | Whole history or page contents. |
| Native Messaging client | Connection, framing, request IDs, reconnect, protocol compatibility. | Direct browser state mutation beyond extension APIs. |
| Go protocol/session manager | Handshake, limits, timeouts, synchronization state. | Trusting unvalidated fields or executing field data. |
| Go index/ranker | Normalization, lexical index, scores, deterministic ordering. | Browser API calls or network access. |
| SQLite repository | Configuration, migrations, bounded activation metadata, installation state. | Source of truth for live tabs. |
| Diagnostics | Health, counts, durations, error classes, redacted traces. | Raw tab content by default. |

## Data flow

1. The extension starts and obtains a full eligible-tab snapshot.
2. The client connects to the registered host and performs a versioned hello.
3. The extension sends the snapshot with profile and projection revision.
4. The host builds or replaces the in-memory index and acknowledges the revision.
5. Browser events become ordered deltas; overflow or mismatch requests a full snapshot.
6. Search queries stay in the local connection and return bounded result references.
7. The extension validates the projection revision, activates the selected browser tab, and confirms outcome.
8. The host persists only allowed configuration and bounded activation metadata.

## Lifecycle states

```text
Disconnected -> Connecting -> Handshaking -> Synchronizing -> Ready
      ▲             │              │              │          │
      └─────────────┴──────────────┴──────────────┴──────────┘
                         recoverable failure
```

- `Disconnected`: no host connection; UI can show repair/retry.
- `Connecting`: startup timeout applies.
- `Handshaking`: protocol, extension, browser, and profile identity are checked.
- `Synchronizing`: a snapshot or resync is in progress; stale results are not activated.
- `Ready`: queries and deltas are accepted.
- `Recovering`: reconnect backoff and snapshot rebuild; no silent data loss.

## Runtime decision

The product target uses a Go Native Messaging host rather than an independent loopback daemon. This gives Chrome and Edge a documented local-process boundary and avoids an application-owned listening port. The host may keep in-memory state while connected, but the design does not rely on permanent process residency. A future standalone daemon requires a new decision covering discovery, authentication, binding, installer behavior, and threat model.

## Failure isolation

- Host failure affects search availability, not browser tabs or browser navigation.
- SQLite failure affects persistence, not current lexical search.
- Missing optional fields affect only their ranking/display signals.
- Service-worker restart triggers reconnect and snapshot reconciliation.
- Semantic or embedding experiments are outside this architecture and cannot block the lexical path.

## Scaling envelope

The first supported envelope is 1,000 open tabs per browser profile and a bounded result list. The architecture must remain correct above that envelope, but performance targets and memory limits are measured at the declared envelope rather than optimized for hypothetical history-scale datasets.
