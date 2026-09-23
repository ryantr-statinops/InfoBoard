# Domain, data, privacy, and security

## Domain entities

### Browser profile

The isolation boundary for configuration, permissions, runtime connection, tab projection, and local data. No profile may query or activate another profile's tabs.

### Tab projection

A rebuildable local record for one eligible open tab:

```text
profile_id, tab_id, window_id, group_id,
title, url, domain, window_label, group_label,
pinned, active, eligible, observed_at, projection_revision
```

`tab_id` and browser identifiers are operational keys, not user-facing identifiers. The projection is not a browser-history database.

### Activation record

A bounded local record used for recency ranking:

```text
profile_id, tab_identity, domain, activated_at, source
```

It stores no page body, cookies, session token, query string, or network payload. It is limited by count and age and is removed by reset/uninstall.

### Search query

An in-memory user input with normalized tokens, optional phrase boundaries, query revision, and result limit. Queries are never transmitted outside the local extension/host boundary.

### Search result

A projection reference plus score explanation fields needed to render and debug the ranking. It expires when the projection revision changes materially.

### Runtime session

A connection between one profile's extension and its Go Native Messaging host, with protocol version, health state, projection revision, and last successful synchronization.

## Data lifecycle

1. Browser events enter the extension's profile-scoped projection.
2. The extension sends a snapshot or ordered delta to the local host.
3. The host normalizes fields and builds the in-memory lexical index.
4. A query returns bounded result references and score reasons.
5. Activation is confirmed through the extension's browser API.
6. Recent activation metadata is optionally persisted in bounded SQLite storage.
7. Reset/uninstall disconnects sessions before deleting InfoBoard-owned local data, including the identity-only extension key after the profile session is closed.

## Trust boundaries

| Boundary | Trusted responsibility | Required protection |
| --- | --- | --- |
| Browser APIs -> extension | Browser supplies tab/window events and performs activation. | Validate IDs, event ordering, profile identity, and unavailable fields. |
| Extension -> Native Messaging host | Local process receives projection and query messages. | Registered host, allowed origin, framed/versioned protocol, bounded payloads. |
| Host -> SQLite | Local runtime persists configuration and bounded recency metadata. | File permissions, migrations, atomic writes, no arbitrary SQL from extension input. |
| Product -> user UI | UI presents status, results, and failures. | Redact sensitive URL parts and never claim unconfirmed actions. |

## Privacy invariants

- No cloud service is required for any product behavior.
- No cookies, local storage, session tokens, network interception, browser history, bookmarks, downloads, or page contents are collected.
- Full URLs may be used transiently for local matching, but query strings and fragments are not displayed by default and are not persisted in activation records.
- Logs contain counts, durations, versions, revisions, and error classes by default. Titles, domains, and URLs are opt-in diagnostics and redacted before export.
- Incognito/private records remain isolated and are not persisted after the private context ends.
- Reset and uninstall remove owned configuration, projection cache, activation records, logs, host registration, and generated diagnostic artifacts.

## Security invariants

- The Native Messaging manifest allows only the exact extension IDs for the supported browser packages.
- The host accepts messages only after a valid hello handshake and profile/session validation.
- Message sizes, query lengths, result limits, synchronization counts, and timeouts are bounded.
- Unknown protocol versions and malformed frames fail closed and trigger a recoverable resync path.
- The host never executes data from a tab field as a command, path, URL to fetch, or SQL fragment.
- No loopback listener is part of the target architecture; a future change would require a new security decision.
