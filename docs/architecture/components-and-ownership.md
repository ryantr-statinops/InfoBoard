# Component ownership

Ownership is explicit so failure handling and review boundaries remain testable.

## Extension components

### Command and search surface

Owns shortcut invocation, query focus, keyboard navigation, result presentation, runtime status, and accessible state. It renders bounded result metadata but does not calculate ranking policy or persist tab data.

### Browser event adapter

Owns browser API permissions, full eligible-tab snapshots, tab create/update/move/group/pin/activate/remove events, profile separation, and activation calls. It converts browser events into ordered projection operations and requests a full snapshot after an event gap.

### Profile-scoped tab projection

Owns the current in-memory representation sent for one browser profile. Each projection has a revision and stable tab/window identity. It never expands into whole browsing history or page-content indexing.

### Native Messaging client

Owns host discovery through the browser's registered Native Messaging boundary, framed transport, request IDs, reconnect backoff, protocol compatibility, and synchronization status. It rejects incompatible or unbounded messages before forwarding them.

## Host components

### Protocol and session manager

Owns hello/hello-ack, profile identity, protocol version, size limits, request timeouts, revision checks, and typed errors. Unknown versions and malformed messages fail closed.

### Projection reconciler

Owns atomic snapshot replacement, ordered delta application, sequence checks, and `resync_required` decisions. A newer authoritative snapshot always supersedes pending deltas.

### Index and ranking engine

Owns Unicode normalization, field tokenization, lexical indexing, named score components, deterministic tie-breaking, bounded results, and ranking-model versioning. It does not call browser APIs or the network.

### SQLite repository

Owns schema migrations, profile-scoped configuration, installation state, and bounded activation metadata. It never becomes authoritative for live tabs.

### Diagnostics and health

Owns counts, durations, versions, state transitions, error classes, and redacted traces. URLs, titles, queries, and tokens are not logged by default.

## Forbidden coupling

- UI code must not embed ranking weights.
- Host code must not activate tabs or access browser APIs.
- Persistence code must not reconstruct a supposedly current browser state without a fresh snapshot.
- Optional semantic or durable-storage experiments must not block lexical search.
- No component may add a data source without updating product, privacy, retention, and acceptance contracts.
