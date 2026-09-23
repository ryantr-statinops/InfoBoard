# Persistence and lifecycle

## Persistence boundary

SQLite is local implementation storage, not the source of truth for live tabs. The live tab projection is rebuilt from browser APIs and kept in host memory. If SQLite is unavailable, the product continues lexical search with defaults and reports degraded persistence.

## Stored data

| Store | Data | Retention |
| --- | --- | --- |
| Configuration | Result limit, density, theme, context-label preference, protocol/ranking versions. | Until reset or uninstall. |
| Installation state | Schema version, host registration state, last successful migration. | Until reset or uninstall. |
| Activation metadata | Bounded local tab identity, domain, timestamp, and source. | Maximum 500 records and 30 days, whichever comes first. |
| Diagnostics | Counts, durations, versions, error classes, redacted lifecycle events. | Maximum 7 days or 10 MB. |
| Live projection | Current tab metadata used by the index. | Memory only; rebuilt after restart. |

No full URL query strings, page content, cookies, session tokens, or raw query history are persisted.

## Schema and migrations

- Every schema has a monotonic version.
- Migrations run in a transaction and record completion only after commit.
- A failed migration keeps the previous database recoverable and places persistence in degraded mode.
- Unknown future schema versions are not rewritten; the host reports an upgrade-required state.
- Database corruption triggers a safe rename/quarantine and rebuild from defaults; it must not prevent tab activation through the browser.

## Startup

1. Validate install paths and host registration.
2. Open SQLite with bounded timeout.
3. Load configuration and ranking/protocol compatibility.
4. Start the Native Messaging handshake.
5. Receive and index the full tab snapshot.
6. Report `Ready` only after projection and ranking model are consistent.

## Shutdown and restart

- Host shutdown flushes bounded activation metadata atomically and closes the database.
- Browser or service-worker restart reconnects and sends a new snapshot.
- Unexpected host exit is visible as unavailable/recovering; reconnect uses bounded backoff.
- Repeated failure offers diagnostics and repair without changing browser state.

## Reset and uninstall

Reset deletes configuration, activation metadata, diagnostics, cached projections, and generated host state, then returns to first-run defaults. Uninstall removes the Native Messaging registration and all InfoBoard-owned files. Neither operation closes tabs, changes browser history, or deletes unrelated browser data.

## Retention and user control

The settings view explains local retention and offers reset. Users can disable recent-activation persistence; when disabled, recency ranking is session-only. The product never silently expands retention to compensate for ranking quality.
