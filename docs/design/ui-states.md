# UI states and responsive behavior

## State presentation

| State | User message | Required action |
| --- | --- | --- |
| Empty library | No bookmarks have been saved. | Offer URL capture. |
| Loading | Existing layout remains stable while data loads. | No destructive action. |
| Snapshot pending | Bookmark saved; content capture has not started. | Allow organization and navigation. |
| Snapshot processing | Content is being captured. | Show progress without blocking edits. |
| Snapshot failed | Bookmark saved; snapshot could not be captured. | Explain safe reason and offer retry. |
| Indexing | Current snapshot is being indexed. | Keyword metadata retrieval may remain available. |
| Ready | Current snapshot and required indexes are current. | Offer normal browse/search actions. |
| Semantic unconfigured | Provider or consent is missing. | Link to setup; do not label as an outage. |
| Semantic degraded | A configured component is unavailable or incompatible. | Use keyword fallback and offer diagnostics/retry. |
| Analytics degraded | Current projection cannot be used safely. | Show bounded fallback or analytics-only error. |
| Validation error | Submitted input is invalid or unsafe. | Preserve input when safe and explain correction. |

Color is never the only state signal. Status has text, icon/shape, accessible name, and programmatic semantics.

## Responsive contract

- At 1440 px, navigation, result list, and detail may form adjacent regions.
- At 390 px, navigation collapses, filters use a modal/drawer, and detail becomes a full-width route or sheet.
- Tables become cards or allow deliberate contained scrolling; the page itself has no horizontal overflow.
- Primary actions remain reachable by keyboard and touch with visible focus.
- Opening/closing mobile detail preserves query, filters, and list position.
- Long URLs, titles, tags, provider errors, and excerpts wrap or truncate without hiding required actions.

## Confirmation and destructive actions

Soft delete requires confirmation naming the bookmark. Provider-consent revocation explains that external calls stop and semantic retrieval becomes unavailable until re-enabled. Backup restore and derived rebuild actions live in maintenance settings and show exact target/status before execution.
