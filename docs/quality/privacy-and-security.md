# Privacy and security

## Permission review

The extension requests only permissions required to observe eligible open tabs, invoke the configured command, present the search surface, and activate the selected tab. History, cookies, bookmarks, downloads, page content, broad host access, and unrelated storage are outside the contract.

## Trust boundaries

- Browser APIs are the authority for current tabs and activation.
- Native Messaging is the local process boundary.
- The host validates protocol, profile identity, revisions, limits, and payloads before indexing.
- SQLite contains only declared configuration, installation state, and bounded activation metadata.
- Diagnostics are metadata-first and redact URLs, titles, queries, and tokens by default.

## Required checks

- No network requests occur in the normal search path.
- No page-content reads, history API calls, cookie reads, or shell execution from user-controlled fields occur.
- Native Messaging manifests allow only the released extension origin.
- Unknown protocol versions and malformed/oversized messages fail closed.
- Private contexts remain separated from normal profiles.
- Reset and uninstall remove only InfoBoard-owned data.
- Logs and diagnostic exports do not become an accidental browsing-data export.

Any exception requires a new product decision, updated privacy documentation, retention/removal rules, and explicit acceptance tests.
