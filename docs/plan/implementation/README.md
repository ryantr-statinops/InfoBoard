# InfoBoard implementation preparation

This workspace translates accepted product and architecture contracts into delivery planning. It does not redefine product behavior, schema, API payloads, or quality criteria.

Read in order:

1. [Current runtime and gap analysis](current-state.md)
2. [Roadmap](roadmap.md)
3. [Task map](tasks.md)

## Status model

Each task uses `not_started`, `in_progress`, `blocked`, `review`, or `verified`. A task becomes `verified` only when implementation and required evidence exist. These files are preparation inputs for the forthcoming full implementation-workspace rewrite; they do not verify runtime capabilities.

## Change rule

When implementation reveals a product or architecture gap, stop the affected task and update the upstream decision/specification first. Runtime evidence, target contract, and delivery state remain separate.
