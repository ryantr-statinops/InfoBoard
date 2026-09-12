# M1 — Local dashboard

**Outcome:** A user can run InfoBoard locally, capture and organize items, and use the dashboard after restart.
**Gate:** All M1 acceptance items pass with evidence; post-MVP portability remains out of scope.
**Dependencies:** program contracts, current SQLite schema

## Epic packages

- [10 — Application foundation](10-application-foundation/README.md)
- [11 — Data model and migrations](11-data-and-migrations/README.md)
- [12 — Dashboard and item workspace](12-dashboard/README.md)

## Delivery order

`10 → 11 → 12`; application boundaries must be agreed before dashboard behavior is verified.

## Acceptance gate

Fresh install, text capture, list/filter, detail, notes/collections, status edit, soft-delete, restart persistence, and desktop/mobile layout must pass with evidence.
