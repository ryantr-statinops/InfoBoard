# Acceptance matrix

The product is accepted only when the observable contract passes across all relevant layers.

| Area | Must prove |
| --- | --- |
| Invocation and UX | Shortcut opens the focused surface, input receives focus, keyboard flow works, dismissal has no side effect. |
| Search | Current-profile fields are searchable, results update live, ranking is deterministic, duplicate tabs are distinguishable. |
| Activation | `Enter` activates the selected tab/window; stale or failed activation is reported without silent substitution. |
| Projection | Create/update/move/group/pin/activate/window/remove events converge without duplicate records. |
| Recovery | Service-worker restart, browser restart, host reconnect, missed event, and full rebuild recover without reinstall in the normal case. |
| Runtime | Handshake, framing, request IDs, limits, timeouts, revisions, malformed messages, and unknown versions fail safely. |
| Privacy | No excluded permission, network path, page-content read, history access, cookie access, or sensitive diagnostic export occurs. |
| Persistence | Configuration, migrations, reset, uninstall, and bounded activation retention respect ownership and profile scope. |
| Accessibility | Keyboard operation, visible focus, assistive labels, contrast, reduced motion, and text scaling meet the supported baseline. |
| Compatibility | Chrome and Edge operate on Linux, macOS, and Windows for the declared release matrix. |
| Performance | Declared p95 latency and memory targets hold at 1,000 indexed tabs and bounded query/result sizes. |

A release gate must point to evidence for each applicable row. A passing happy path cannot waive a failure, privacy, compatibility, or recovery row.
