# Refactor decision log

These are provisional decisions for discovery. They are not final product requirements until the idea is accepted and specified.

| ID | Status | Decision or question | Current position |
| --- | --- | --- | --- |
| `REF-001` | proposed | What is the first product direction? | Chromium-first instant tab search |
| `REF-002` | proposed | Which browser family is supported first? | Chrome and Edge desktop |
| `REF-003` | proposed | Which browser data is in the first scope? | Open tabs and lightweight tab metadata |
| `REF-004` | open | How does the extension reach Go? | Compare warm daemon IPC with Native Messaging |
| `REF-005` | proposed | What ranks normal searches? | In-memory lexical/fuzzy matching plus recency/context signals |
| `REF-006` | proposed | Is ChromaDB on the hot path? | No; optional asynchronous semantic layer |
| `REF-007` | open | Is a background Go process acceptable? | Must be explicitly accepted because instant response requires a warm runtime |
| `REF-008` | open | What is the UI surface? | Popup, side panel, or small focused window |
| `REF-009` | open | Are full-page contents indexed? | Not in the first scope |
| `REF-010` | open | Which OSes are packaged first? | Decide before native host/daemon packaging |

## Decision rule

A decision becomes binding only after it records user value, privacy impact, architecture impact, failure behavior, and an observable acceptance signal.
