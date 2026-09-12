# Design — UI states and responsive behavior

## UI states

| State | Required behavior |
| --- | --- |
| Empty | Explain the value and guide the user to Add |
| Loading | Keep the layout stable with a placeholder |
| Processing | Show the step/job state without assuming indexing is complete |
| Failed | Show a safe reason, affected area, and Retry when appropriate |
| Degraded | Identify unavailable semantic/analytics capabilities; keep the core flow usable |
| Duplicate/conflict | Explain the existing item and next action |
| Deleted | Close detail/refresh the list while preserving filter context |

## Job-to-UI state mapping

| Job state | UI state |
| --- | --- |
| `queued` | `queued` |
| `extracting`, `chunking`, `embedding` | `processing` |
| `indexed` | `indexed` |
| `failed` | `failed` |

The UI may show a detailed step label in advanced/detail view but does not create another organization status for the item.

## Responsive behavior

- Desktop target 1440 px: approximately 240 px sidebar, two-column dashboard, right-side detail panel.
- Mobile target 390 px: one column, collapsed menu, full-width detail surface.
- No horizontal overflow in add, list, detail, note, or search flows.
- Tables/charts provide a summary or suitable mobile alternative.

## Accessibility

- Every action is keyboard accessible and has visible focus.
- Form fields have labels; errors are linked to inputs and preserve unsuccessful submission data.
- Status is not conveyed by color alone; loading/retry/degraded states have text.
- Dialogs/panels manage focus and define Escape/close behavior.
- Imported content is rendered as untrusted text and cannot create active UI controls.
