# Implementation conventions

## Authority

Canonical product, design, architecture, quality, and operations documents own behavior and contracts. Implementation documents own sequencing, source impact, task status, and execution evidence. Contract ambiguity blocks the affected task until the upstream document is corrected.

## Package contract

Every leaf package contains `README.md`, `plan.md`, `tasks.md`, `examples.md`, `references.md`, and `execution.md`. The package task file is the sole authoritative checklist.

## Task record

Each required task has:

- stable package-prefixed ID and exactly one checkbox;
- outcome, prerequisites, canonical references, and source areas;
- ordered actions, migration/compatibility impact, and failure recovery;
- required tests, completion criteria, one-to-three-commit boundary;
- evidence anchor in the package execution log.

A task is checked only after its completion criteria pass and real evidence is recorded.

## Status and rollup

Package status is `not_started`, `in_progress`, `blocked`, or `done`. A transition updates the package README, milestone README, root index, and execution log in the same commit. Rollups never duplicate task checkboxes.

## Evidence

Execution logs begin as `not_started` templates. Record exact commit IDs, commands, exit results, test artifacts, deviations, follow-up decisions, and reviewer notes. Never pre-fill successful evidence. Redact secrets, raw content, and sensitive provider responses.

## Examples

Examples clarify expected integration and verification but are non-authoritative. They link to the canonical contract and use synthetic, non-sensitive data.

## Git discipline

Work on `dev`; keep commits scoped; push every commit to `origin/dev`; never merge `main`. Runtime implementation commits should normally align with one task and a package should span one to three commits per task.
