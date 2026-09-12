# 12 — Dashboard and item workspace

**Plan status:** `ready`  
**Delivery status:** `not_started`  
**Baseline coverage:** `partial`  
**Milestone:** M1  
**Dependencies:** 10, 11

## Outcome

Users can capture, list, filter, open, edit, organize, annotate, and soft-delete items from a responsive local dashboard.

## UI contract

- Simple mode is the default: clear capture, list, filter, and detail actions with safe defaults.
- Item detail exposes title, status, source metadata, snapshot content, notes, and collections.
- Loading, empty, validation, conflict, unavailable, and degraded states are explicit.
- HTMX fragments preserve the current server-rendered interaction model and public URLs.
- The interface remains usable at 1440 px and 390 px with keyboard-visible focus and accessible labels.

## Feature slices

1. Dashboard shell, navigation, and item list.
2. Text capture, item detail, edit, and soft delete.
3. Collection assignment and filtering.
4. Notes and related detail actions.
5. Search entry point and health/degraded messaging.

## Non-goals

Browser extension, browser profile migration, cloud sync, credential/session transfer, and AI-assisted workflows remain post-MVP discovery.

## Acceptance and review

- M1 acceptance flows work after reload and restart.
- Deleted items disappear from public lists without deleting required relations.
- Empty/loading/error states are testable and readable.
- No change is made to the public entrypoint or to target capabilities not yet implemented.
