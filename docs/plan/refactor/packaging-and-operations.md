# Packaging and operations

## Distribution artifacts

The complete product ships as:

1. a Chrome Web Store extension package;
2. an Edge Add-ons extension package; and
3. a platform-specific Native Messaging host installer for Linux, macOS, and Windows.

The extension and host declare compatible product, protocol, and ranking versions. A mismatched pair fails with a repairable status rather than accepting tab data under unknown semantics.

## Installation

The installer MUST:

- verify package integrity before installation;
- install the host binary and browser-specific manifest in the documented location;
- register only the exact supported extension IDs in `allowed_origins`;
- set restrictive file permissions;
- perform a hello/health smoke check;
- explain the local host, data boundary, and required browser permissions;
- leave a rollback path if registration or health validation fails.

Installation does not require a cloud account and does not start a loopback server.

## Update

Extension and host updates are versioned independently but compatibility checked together. An update MUST preserve configuration when the schema remains compatible, run transactional migrations otherwise, and keep the previous package available until health verification succeeds.

## Repair

The settings view reports one of: host missing, host registration invalid, protocol mismatch, permission unavailable, host unavailable, persistence degraded, or healthy. Repair re-registers or reinstalls only InfoBoard-owned artifacts and never changes browser tabs.

## Diagnostics

Diagnostics include:

- extension, host, browser, OS, protocol, and ranking versions;
- connection and synchronization state;
- projection count and revision;
- query/activation latency distributions;
- error class and retryability;
- persistence migration state.

Diagnostics exclude raw URLs, titles, query strings, page content, cookies, and tokens unless the user explicitly enables a short-lived local debug capture. Export redacts those fields by default.

## Logging and crash behavior

Logs are local, bounded, and rotatable. Host crashes produce a recoverable unavailable state, bounded reconnect attempts, and a diagnostic marker. A crash must not close tabs, alter browser history, or block browser use.

## Uninstall

Uninstall removes the extension, registered host manifests, host binaries, InfoBoard-owned database, activation metadata, logs, and cache. It does not remove browser tabs, other profiles, unrelated files, or browser data. A post-uninstall check verifies owned paths are absent.

## Operational security

- Release artifacts are signed or checksum-verified.
- Host manifests are generated from a single release configuration and reviewed for origin allowlists.
- Package installation and update paths reject path traversal and unexpected symlinks.
- Recovery tools use explicit ownership markers before deleting anything.
- No external telemetry is enabled by default.
