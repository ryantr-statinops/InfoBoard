# InfoBoard implementation

This section translates accepted product and architecture contracts into delivery order and actionable work. It does not redefine product behavior, schema, API payloads, or quality criteria.

- [Roadmap](roadmap.md) defines dependency order and milestone outcomes.
- [Tasks](tasks.md) maps delivery units to requirements, contracts, and verification.

## Status model

Each task uses one state: `not_started`, `in_progress`, `blocked`, `review`, or `verified`. A task becomes `verified` only when implementation and required evidence exist. The current documentation refactor establishes plans only; it does not verify runtime capabilities.

## Change rule

When implementation reveals a product or architecture decision gap, stop that task and update the upstream decision/specification first. Do not resolve contract ambiguity only inside code or this implementation section.
