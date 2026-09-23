# Updates, rollback, and repair

## Update flow

An update replaces extension and host artifacts as a compatible unit. The package records extension version, host version, protocol version, schema version, and ranking-model version. The Native Messaging handshake refuses incompatible combinations rather than exchanging tab data under an unknown contract.

The updater must:

- preserve allowed profile-scoped configuration;
- migrate SQLite before advertising healthy persistence;
- retain enough metadata to explain the previous version and failure reason;
- validate manifest origin and executable ownership;
- restart or reconnect the host through the browser-managed boundary;
- verify a fresh snapshot before returning to `Ready`.

## Rollback

Rollback restores a previously known compatible extension/host pair. It must not silently reuse an incompatible schema or ranking model. If rollback cannot validate the pair, the product remains in a repairable unavailable state and gives the user a concrete next action.

## Repair

Repair is idempotent. It may recreate manifests, restore executable permissions, remove stale registrations, repair owned directories, and re-run the handshake. Repair must not delete browser tabs or unrelated browser data. Destructive data reset is a separate, explicit action.

## Failure presentation

Every update, rollback, or repair failure exposes an error class, affected component, whether browser state changed, and the next safe action. Logs contain versions, counts, durations, and error classes by default—not URLs, titles, queries, or tokens.
