# Security and privacy

## Trust boundaries

Untrusted inputs include submitted URLs, DNS answers, redirects, remote headers/bodies, extracted markup/text, search queries, names/notes, provider responses, and restored archives. The local browser is not permission to expose the service beyond loopback.

## Local HTTP boundary

- Bind to `127.0.0.1` by default.
- Validate Host and Origin for state-changing browser requests and apply CSRF protection appropriate to server-rendered/JSON flows.
- Use secure response headers and no runtime CDN dependencies.
- Bound request body, field length, pagination, filter counts, and query complexity.
- Escape all user/remote content by default; any sanitized rich rendering uses an explicit allowlist.

## Public URL and SSRF boundary

- Accept only HTTP(S), with no URL credentials.
- Reject loopback, private, link-local, multicast, reserved, unspecified, and otherwise non-public destinations for IPv4 and IPv6.
- Validate DNS answers, actual connected address, and every redirect hop.
- Disable ambient credentials/cookies and unsafe proxy inheritance.
- Bound redirects, time, bytes, decompression, parser work, and temporary-file lifetime.
- Do not render remote scripts/styles or trust upstream MIME/title/markup.

## Provider configuration and consent

Endpoint/model/key configuration and consent are separate actions. Verification may send only the minimal compatibility request documented by the UI; it cannot index a snapshot or semantic query and cannot grant consent.

Before consent, the disclosure names:

- Snapshot text sent for document embeddings.
- Search query text sent for query embeddings.
- Configured endpoint/model.
- Local retention in RocksDB and ChromaDB.
- How to revoke consent and rebuild/delete derived semantic data.

Consent records disclosure version and time. Revocation takes effect before the API responds: new provider calls stop, queued semantic jobs pause, and cleanup is queued. Canonical snapshots remain local.

## Secrets

- API keys are write-only and stored through a dedicated local secret-storage boundary; SQLite stores only an opaque reference and key-presence metadata.
- If no acceptable OS/keyring/file-permission-backed mechanism is implemented, setup must fail closed rather than store plaintext in SQLite or logs.
- Responses, exceptions, diagnostics, command output, and backups redact keys, authorization headers, raw provider responses, URL credentials, and sensitive local paths.
- Key replacement is atomic; failed verification does not destroy the last valid reference unless explicitly removed.

## Data minimization

- Provider requests contain only bounded text and required model/input fields.
- Provider request identifiers and timings may be logged; snapshot/query text may not.
- DuckDB excludes raw snapshots and notes.
- Diagnostics use aggregate counts, component versions, safe error codes, timings, and hashed identifiers.

## Component health semantics

- `unconfigured`: required setup or consent has never completed; not an outage.
- `ready`: contract checks pass for the current configuration/revision.
- `degraded`: a configured component fails while a safe fallback exists.
- `unavailable`: the component and required behavior have no safe fallback.

SQLite/FTS unavailability affects core health. RocksDB, ChromaDB, DuckDB, or provider failure affects only owned capabilities as documented, but cannot be hidden from health or the relevant UI response.

## Security verification

Required cases include private/redirect/rebinding SSRF, oversized/compressed payloads, parser timeouts, unsafe markup, Host/Origin/CSRF rejection, API-key and content redaction, consent before transfer, immediate revocation, malicious provider responses, archive traversal/checksum failure, and explicit derived-path safeguards.
