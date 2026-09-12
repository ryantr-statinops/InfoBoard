# Architecture — Decisions

## Current state

The foundational decisions appear in product/implementation docs but did not previously have a canonical architecture index.

## Target state — locked decisions

| Decision | Rationale | Consequence |
| --- | --- | --- |
| SQLite is the system of record | Local, transactional, recoverable | Every derived store hydrates/rebuilds from SQLite |
| No ORM in the MVP | Transparent SQL/schema ownership | Migrations and data-access SQL must be version-controlled |
| Server-rendered Jinja2 + HTMX | No SPA/Node build required | Route/partial contracts need integration tests |
| Core/full modes | Native/ML dependencies are not stable on every machine | Keyword/core flow always has a fallback |
| Sequential durable worker | Simplifies local consistency | Requires leases, checkpoints, retry, and restart recovery |
| Local embedding by default | Privacy and offline operation | Explicit model preparation and revision metadata are required |

## Decision lifecycle

- Product outcome changes are recorded in the Internal PRD first.
- A new architecture decision records context, alternatives, decision, consequence, and supersession.
- Implementation governance keeps execution-specific decisions/evidence and links here.
- A target decision is not marked implemented until current-state/evidence confirms it.
