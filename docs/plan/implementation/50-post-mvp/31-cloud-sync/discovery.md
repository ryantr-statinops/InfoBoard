# Discovery — Cloud sync and multi-device

## Questions and experiments

Define replica/change-log semantics, offline conflicts, key recovery, quota/cost, retention, export/delete guarantees, and service-unavailable behavior. Prototype at least two local replicas with offline edits and delete-vs-update conflicts.

## Implementation-ready gate

Require threat/privacy review, conflict acceptance, identity/key recovery, quota model, export/delete guarantee, compatibility/versioning, and degraded offline behavior.

Collaboration is a dependent discovery package and cannot be enabled by this package alone.
