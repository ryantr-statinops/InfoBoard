# Data and failure boundaries

## Data flow

1. Browser APIs expose eligible open tabs for the active profile.
2. The extension creates a normalized projection containing title, URL/domain, window, group, pinned state, and allowed activation metadata.
3. A full snapshot establishes the host projection and revision.
4. Subsequent browser events become ordered deltas.
5. The host indexes searchable fields and applies deterministic ranking.
6. Query results return references and bounded display metadata.
7. The extension performs activation and reports the confirmed outcome.
8. Only allowed configuration and bounded activation metadata reach SQLite.

## Failure isolation

| Failure | User-visible effect | Must remain intact |
| --- | --- | --- |
| Host unavailable | Search shows retry/repair state | Browser tabs and navigation |
| Host crash | Reconnect and rebuild | Browser state; no false activation |
| SQLite unavailable | Persistence health warning | Current lexical search when safe |
| Missed browser event | Rebuild from full snapshot | Projection convergence |
| Stale result | Refresh before activation | No silent activation of another tab |
| Optional component failure | Optional capability unavailable | Normal lexical search |
| Missing optional field | Reduced context/ranking signal | Other fields and deterministic order |
| Protocol mismatch | Repairable incompatibility state | No tab data sent to incompatible host |

## Privacy boundary

No network path, cookies, browser history, page contents, cloud index, or arbitrary page injection is required. Diagnostics use metadata such as counts, durations, versions, and error classes; sensitive tab content is redacted or disabled by default.

## Scale envelope

The declared performance envelope is 1,000 open tabs per browser profile. The implementation must bound payload sizes, field lengths, token counts, query length, result count, and memory growth before allocation or indexing. Correctness above the envelope is required; performance commitments are measured at the declared envelope.

## Review rule

A proposed data source, process boundary, persistence responsibility, or failure fallback is incomplete until it names its owner, privacy impact, retention behavior, failure behavior, and measurable acceptance signal.
