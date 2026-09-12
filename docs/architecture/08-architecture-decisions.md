# Architecture — Decisions

## Current state

The foundational decisions have a canonical Architecture index. Runtime delivery remains partial, and implementation evidence determines whether each target decision is operational.

## Target contract — locked decisions

| Decision | Rationale | Consequence |
| --- | --- | --- |
| SQLite is the system of record | Local, transactional, recoverable | Every derived store hydrates/rebuilds from SQLite |
| No ORM in the MVP | Transparent SQL/schema ownership | Migrations and data-access SQL must be version-controlled |
| Server-rendered Jinja2 + HTMX | No SPA/Node build required | Route/partial contracts need integration tests |
| Core/full modes | Native/ML dependencies are not stable on every machine | Missing unconfigured full-mode components is normal; failure after explicit enablement is degraded |
| Sequential durable worker | Simplifies local consistency | Requires leases, checkpoints, retry, and restart recovery |
| Local embedding by default | Privacy and offline operation | Explicit model preparation and revision metadata are required |
| Architecture owns interface contracts | Product and implementation need one target contract | Implementation maps tasks/evidence to the Architecture contract instead of copying it |

## Decision lifecycle

- Product outcome changes are recorded in the Internal PRD first.
- A new architecture decision records context, alternatives, decision, consequence, and supersession.
- Implementation governance keeps execution-specific decisions/evidence and links here.
- A target decision is not marked implemented until current-state/evidence confirms it.

## Implementation gap

- The API/interface target is currently duplicated in the implementation workspace and has not been converged on by runtime evidence.
- Proposed implementation decisions remain unimplemented until their owners record evidence.

## Owning work

Architecture maintainers own canonical decisions; affected implementation epics own delivery evidence and governance records supersession/history.

## Evidence required

Accepted decision records for public API/schema/security/dependency changes, linked task evidence, and a review proving that no competing canonical contract remains.
