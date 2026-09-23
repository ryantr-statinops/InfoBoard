# Diagnostics, reset, and uninstall

## Diagnostics

Diagnostics describe the local state without exporting browsing content. Useful fields include extension/host/protocol/schema/ranking versions, lifecycle state, projection count, synchronization revision, query and activation durations, reconnect count, and typed error classes.

URLs, titles, query strings, tokens, cookies, page content, and full tab projections are redacted or disabled by default. A diagnostic export must state its retention and destination before leaving the local machine.

## Reset

Reset removes InfoBoard-owned configuration, installation state, bounded activation metadata, and rebuildable local index state. It then reconnects and requests a fresh browser snapshot. Reset must not close tabs, change tab positions, delete browser history, or remove unrelated browser data.

## Uninstall

Uninstall removes the extension, Native Messaging registration, host artifacts, InfoBoard-owned local data, and diagnostics according to the platform package contract. It must be safe to run repeatedly and must clearly distinguish owned data from browser-owned data.

The current tab projection is always rebuildable from browser APIs. Uninstall therefore must not claim to delete browser tabs or browsing history because those are outside InfoBoard ownership.

## Acceptance

The [verification contract](../plan/refactor/verification-and-acceptance.md) must cover host absence, host crash, malformed messages, persistence failure, reset ownership boundaries, private-context cleanup, uninstall idempotence, and redaction defaults.
