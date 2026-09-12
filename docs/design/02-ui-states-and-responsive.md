# Design — UI states and responsive behavior

## User-facing states

| State | Required behavior | Next action |
| --- | --- | --- |
| Empty | Explain the value of saving information and lead to Add | Add the first item |
| Loading | Preserve layout shape and identify the loading area | Wait or retry |
| Queued | Confirm the item was saved and is waiting for processing | Continue browsing |
| Processing | Show that extraction/indexing is in progress without claiming indexed content | Continue browsing or open detail |
| Indexed | Show that the item is available for normal retrieval | Read, organize, or search |
| Failed | Show a safe reason, affected capability, and Retry when recoverable | Retry or keep the item |
| Duplicate/conflict | Explain that an existing item was found and avoid silent overwrite | Open existing item or cancel |
| Deleted | Hide the item from user-facing lists and close its detail view | Return to the preserved context |
| Degraded | Explain the configured optional capability that is unavailable while keeping core flow usable | Repair, rebuild, or continue with keyword/read |

## Job-to-UI state mapping

| Technical job state | UI state | Product meaning |
| --- | --- | --- |
| `queued` | `queued` | Saved and waiting to be processed |
| `extracting` | `processing` | Source content is being extracted |
| `chunking` | `processing` | Content is being prepared for retrieval |
| `embedding` | `processing` | Optional/full-mode representation is being created |
| `indexed` | `indexed` | Current content is available to retrieval |
| `failed` | `failed` | Processing stopped with an actionable error |

The UI may show a more detailed step in advanced mode, but it must not change
the durable job state or expose internal identifiers as user-facing concepts.

## Core and full-mode behavior

- Core mode always provides keyword/read behavior when SQLite and FTS5 are
  available.
- Full mode controls are hidden or clearly marked as unavailable until the
  user explicitly configures full mode.
- An unconfigured full mode is a normal core-mode state, not degraded.
- A configured full mode with an unavailable model, vector store, or cache is
  degraded; keyword results and saved snapshots remain available.
- Analytics uses its core fallback when the optional analytics engine is not
  available. The UI should identify a fallback only when it affects the
  meaning or freshness of the displayed metric.

## Responsive behavior

### Desktop — 1440 px

- Keep navigation, filter/search controls, item list, and detail panel visible
  without horizontal scrolling.
- Reserve enough width for title, source, status, and primary actions.
- Keep analytics secondary to the item workspace.

### Mobile — 390 px

- Collapse navigation behind an explicit control.
- Stack filters and primary actions vertically.
- Open detail as a full-width panel or page-like surface.
- Keep title, source, status, and the next action visible without horizontal
  scrolling.
- Preserve query and filter state when returning from detail.

## Accessibility

- Every primary action has an accessible name and visible focus state.
- Status is conveyed by text and not color alone.
- Validation and failure messages are associated with the affected control.
- Dynamic list/detail updates announce meaningful changes without stealing
  focus.
- Keyboard users can open, close, and return from detail without losing
  context.
- Contrast, readable line length, and touch target size are reviewed at both
  target viewports.
