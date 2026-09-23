# Native Messaging runtime protocol

## Transport

The extension connects to the registered Go Native Messaging host. Messages use the browser's framed stdin/stdout transport and JSON payloads. The protocol adds its own version and request identity; it does not expose a loopback port.

## Envelope

Every application message uses:

```json
{
  "protocol": 1,
  "type": "query",
  "request_id": "r-123",
  "profile_id": "profile-local-id",
  "projection_revision": 42,
  "payload": {}
}
```

Required rules:

- `protocol`, `type`, and bounded payload validation are mandatory.
- `request_id` is required for request/response operations and is unique within a connection.
- `profile_id` is an opaque per-profile value; it is not a browser account identifier.
- `projection_revision` prevents activation or result use against an unknown snapshot.
- Unknown message types and protocol versions fail closed with a typed error.
- Maximum payload size, field lengths, query length, tab count, and result count are enforced before allocation or indexing.

## Handshake

The client sends `hello` with extension version, browser family/version, profile identity, supported capabilities, and protocol version. The host returns `hello_ack` with the accepted protocol, host version, limits, ranking model version, and persistence health.

The extension does not enter `Ready` until the handshake succeeds. A mismatch yields a repairable incompatibility state and does not send tab data.

## Synchronization messages

- `snapshot`: complete eligible-tab projection and revision; replaces the host projection atomically.
- `delta`: ordered create/update/move/group/pin/activate/remove operations with sequence numbers.
- `sync_ack`: accepted revision and sequence.
- `resync_required`: host or client detected a sequence/revision gap; extension sends a new snapshot.

A snapshot is authoritative. Deltas are an optimization and never override a newer snapshot.

## Query messages

`query` contains normalized-by-contract user input, result limit, current window, and the projection revision known by the client. The host returns `query_result` with result references, bounded display metadata, ranking model version, projection revision, and timing counters.

The host MUST reject a query against an unknown or rebuilding revision rather than returning silently stale results. The extension may render a clearly marked stale state while requesting resync.

## Activation messages

The extension owns activation because browser APIs are not delegated to the host. The host only returns a result reference. Before activation, the extension verifies that the tab ID and projection revision are still valid. After activation it reports `activation_observed` or `activation_failed` for diagnostics and recency updates.

## Health and errors

Supported messages include `health`, `health_result`, and typed `error` responses. Error classes include:

```text
INVALID_FRAME, PROTOCOL_MISMATCH, PROFILE_MISMATCH,
PAYLOAD_LIMIT, SNAPSHOT_REQUIRED, REVISION_MISMATCH,
INDEX_REBUILDING, QUERY_TIMEOUT, PERSISTENCE_DEGRADED,
HOST_SHUTDOWN, INTERNAL_FAILURE
```

Errors include a safe user-facing code, retryability, and request identity. They do not include raw URLs, titles, or secrets by default.

## Compatibility

Protocol changes are additive when possible. A breaking change increments the major protocol version and requires host/extension compatibility checks. The host reports supported versions; the extension chooses only a version it understands. Tests cover malformed frames, disconnects, duplicate requests, out-of-order deltas, oversized payloads, and interrupted snapshots.
