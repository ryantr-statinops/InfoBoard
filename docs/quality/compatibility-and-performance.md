# Compatibility and performance

## Compatibility matrix

| Dimension | Supported boundary |
| --- | --- |
| Browser | Chrome desktop and Edge desktop |
| Operating system | Linux, macOS, Windows |
| Context | Normal profile; private/incognito contexts only where browser permissions and isolation allow |
| Runtime | Manifest V3 extension with registered Go Native Messaging host |
| Protocol | Versioned framed JSON with fail-closed incompatibility handling |
| Dataset | Current eligible open tabs; measured envelope is 1,000 tabs per profile |

The release matrix must name concrete browser versions, OS versions, architectures, extension/host versions, protocol version, schema version, and ranking-model version.

## Performance targets

- Shortcut to focused input: p95 <= 100 ms on supported reference hardware when healthy.
- Query to rendered result: p95 <= 50 ms for 1,000 indexed tabs and a 64-character query.
- Selection to activation: p95 <= 100 ms excluding browser scheduling delays.
- Ranking determinism: same projection, query, and ranking timestamp produce identical order and scores.
- Projection convergence: after a full snapshot, indexed IDs equal eligible browser-visible IDs.

## Resource boundaries

Payload size, field length, token count, query length, tab count, result count, and diagnostic output must be bounded before allocation or indexing. The hot path performs no network access, page-content fetch, semantic inference, or history-scale scan.

Performance measurements must include cold start, warm connection, reconnect, full snapshot, incremental delta, empty query, long query, and 1,000-tab search.
