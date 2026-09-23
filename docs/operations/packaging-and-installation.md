# Packaging and installation

## Product artifacts

A release consists of the browser extension package, the signed or integrity-checked Go Native Messaging host binary for each supported operating system and architecture, Native Messaging manifests, and the installer or registration instructions needed to bind the extension to the host.

Supported browser families are Chrome and Edge desktop. Supported operating systems are Linux, macOS, and Windows. The release matrix must identify browser versions, OS versions, CPU architecture, extension version, host version, protocol version, and ranking-model version.

## Installation responsibilities

Installation must:

1. place the host artifact in the platform-appropriate application location;
2. register the Native Messaging manifest with the supported browser family;
3. restrict allowed extension origins to the released extension identity;
4. install the extension package and required command registration;
5. verify host discovery, handshake, profile separation, and a full tab snapshot;
6. report a repairable error when registration, permissions, or compatibility checks fail.

Installation must not request history, cookies, page-content, broad host, or unrelated storage permissions. It must not expose a loopback listening port.

## Idempotence and ownership

Repeated installation must converge on one known version without duplicating manifests, registrations, data directories, or startup entries. The installer owns only InfoBoard artifacts. It must not alter browser tabs, unrelated browser data, or user content.

## Release evidence

A release is not installable by declaration alone. Verification must cover clean installation, existing installation upgrade, missing host, wrong extension origin, incompatible protocol, permission denial, profile separation, and first full-snapshot convergence.
