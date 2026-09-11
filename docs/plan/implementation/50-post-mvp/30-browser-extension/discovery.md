# Discovery — Browser extension and capture

## Questions

- Which browsers and store policies are in scope?
- Does capture send URL/selection only or snapshot HTML?
- Is the local handshake native messaging, localhost, or another mechanism?
- What happens when the app is closed, the URL is private, or the network fails?

## Prototype gate

Send selection plus URL to a local endpoint, show success/error, avoid secrets, and measure permission surface, duplicate behavior, and offline handling.

## Implementation-ready gate

Approve browser matrix, permissions/consent, payload version, handshake authentication, retry/offline behavior, security review, and migration from the current URL endpoint.
