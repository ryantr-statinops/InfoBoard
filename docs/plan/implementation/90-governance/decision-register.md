# 91 — Decision register

**Status:** `active`

## Convention

Every decision has an ID, date, status, context, decision, alternatives, impact, and affected epic. History is not rewritten; a new record supersedes an old one.

## Decisions

| ID | Status | Decision | Impact |
| --- | --- | --- | --- |
| ADR-001 | accepted | SQLite is the source of truth; derived stores are rebuildable. | 11, 14, 18 |
| ADR-002 | accepted | MVP is single-user/local-first and binds to `127.0.0.1`. | 17, 31, 32 |
| ADR-003 | accepted | UI uses Jinja2 + HTMX; no SPA/Node build. | 12, 21 |
| ADR-004 | accepted | Core mode does not depend on semantic search; full mode is optional. | 15, 21 |
| ADR-005 | accepted | Work on `dev`, use small commits, and stop for user review/merge. | 03, all epics |
| ADR-006 | proposed | Normalize list responses to `{items,total,limit,offset}`. | 04, 12, 20 |
| ADR-007 | proposed | MVP worker is sequential in-process with SQLite checkpoint/lease. | 14, 19 |
| ADR-008 | accepted | Each implementation epic uses README/plan/tasks/references/execution; examples and guides are conditional. | all MVP epics |
| ADR-009 | accepted | M5+ packages remain discovery-only and use `D*` IDs until implementation-ready. | 30–35 |

## Record template

```text
ID / date / status
Context and problem
Decision
Alternatives rejected and why
Consequences / migration
Owners and affected epics
Evidence and supersedes
```

## Review policy

Decisions affecting public API, schema, security boundaries, or dependencies are recorded before code. If user preference is required, keep the epic in `draft` until review resolves it.
