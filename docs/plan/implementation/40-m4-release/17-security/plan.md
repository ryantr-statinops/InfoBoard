# 17 — Security and privacy

**Plan status:** `ready`  
**Delivery status:** `not_started`  
**Baseline coverage:** `partial`  
**Milestone:** M4  
**Dependencies:** 10, 13, 18, 19

## Threat boundary

The MVP binds to `127.0.0.1`, is single-user, and has no authentication. URL/file/text inputs are untrusted. Chroma, RocksDB, and DuckDB are local derived stores. Any network or cloud provider requires a new threat model.

## Controls

- Validate Host/Origin on write requests; never enable wildcard CORS.
- Accept only HTTP(S) URLs and block loopback, private, link-local, reserved, and metadata destinations after DNS, socket, and redirect checks.
- Enforce 20 MB file, 5 MB HTML, and 1,000,000-character/10,000-chunk text limits.
- Escape Jinja/HTMX output; never render user HTML or Markdown directly.
- Read secrets only from the environment; never write provider keys to the database, logs, or unencrypted backups.
- Redact bodies, content, sensitive queries, and credential-bearing URLs from logs.

## Failure and acceptance

Reject unsafe input before creating a row/job, use stable error codes, and avoid revealing filesystem/network details. Security failures must not retry indefinitely.

- Initial and redirected SSRF fixtures are blocked.
- Oversized input returns `413`.
- XSS strings render as text.
- Secrets are absent from captured logs.
- Invalid write Origin/Host is rejected.

## Review gate

Inspect URL clients, template `safe` usage, logging, subprocesses, and the integration security suite.
