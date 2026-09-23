# Interaction states

Every state must tell the user what happened, whether browser state changed, and what action is safe next.

| State | Presentation | Allowed actions |
| --- | --- | --- |
| Opening | Focus moves to the query field; stale rows do not flash. | Type or dismiss. |
| Empty query | Bounded recent/context list with an explanation. | Type, navigate, activate, dismiss. |
| Results | Ordered rows with visible selection and context. | Navigate or activate. |
| No results | Explain searchable fields and invite query editing. | Edit or dismiss. |
| Rebuilding | Show freshness status; display previous rows only when clearly marked. | Wait, retry, dismiss. |
| Runtime unavailable | Explain local search is unavailable and offer repair/retry. | Retry, diagnostics, dismiss. |
| Stale selection | Refresh the projection before activation. | Retry or select a refreshed row. |
| Activation failure | State that activation was not confirmed. Keep surface open. | Retry, select another row, dismiss. |
| Private context unavailable | Explain the permission/context boundary. | Switch context or dismiss. |
| Reset/uninstall | Confirm owned local data removal without claiming browser data removal. | Confirm, cancel, or return. |

## Trust rules

- Never claim activation before the browser API confirms it.
- Never hide a recoverable runtime or synchronization failure.
- Never silently activate a different tab because the original result disappeared.
- Never expose sensitive URL query strings by default.
- Never make network access or page-content permission appear necessary for normal search.

## Accessibility states

Focus order, selected-result state, result labels, runtime status, and action outcome must be exposed to assistive technology. Reduced-motion settings disable nonessential animation. Text scaling and narrow windows must preserve the query field and selected result.
