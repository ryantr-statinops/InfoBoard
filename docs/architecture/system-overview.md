# System overview

## Product shape

InfoBoard is a Manifest V3 Chrome/Edge desktop extension connected to a local Go Native Messaging host.

```text
Chrome or Edge profile
  ├─ configurable command
  ├─ focused search surface
  ├─ browser event adapter
  ├─ profile-scoped tab projection
  ├─ activation controller
  └─ Native Messaging client
          │ framed JSON over stdin/stdout
          ▼
Go Native Messaging host
  ├─ protocol and session manager
  ├─ projection reconciler
  ├─ in-memory lexical index
  ├─ deterministic ranking engine
  ├─ bounded SQLite repository
  └─ diagnostics and health
```

The extension owns browser authority. The host owns local indexing, ranking, protocol state, diagnostics, and the bounded persistence allowed by the product contract.

## Authority boundaries

| Boundary | Owns | Does not own |
| --- | --- | --- |
| Browser extension | Browser permissions, tab snapshots, browser events, UI, activation calls | Ranking weights, SQL, process residency policy |
| Native Messaging client | Framing, request IDs, reconnect, compatibility, send/receive state | Direct browser mutation outside extension APIs |
| Go host | Protocol validation, synchronization, index, ranking, health | Browser APIs, network access, page content |
| SQLite repository | Configuration, installation state, migrations, bounded activation metadata | Current open tabs as durable truth |
| Search surface | Input, selection, status, accessible presentation | Hidden ranking or stale activation decisions |

## Product boundary

Only the current profile's eligible open tabs enter the projection. The architecture does not require cookies, browser history, page contents, network interception, cloud indexing, or a loopback listening port.

The current tab projection is rebuildable. Persistence improves configuration and bounded activation context; it is not a second source of truth for currently open tabs.

## Design invariants

- Every runtime request has a bounded payload, timeout, and typed error path.
- Every result is tied to a known profile and projection revision.
- Browser state changes are performed by the extension, never by the host.
- A failed optional component cannot remove the normal lexical search path.
- The architecture is measured at 1,000 open tabs per profile and remains correct above that envelope.

See the [canonical target architecture](../plan/refactor/architecture.md) for the binding contract.
