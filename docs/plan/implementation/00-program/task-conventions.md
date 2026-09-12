# Implementation task conventions

## Purpose

Tasks are the smallest reviewable delivery units in the implementation workspace. A task must be specific enough for an engineer to implement, test, review, and record without inventing missing product or architecture decisions.

## Task IDs and states

- MVP tasks use `T<epic>-<number>`, for example `T10-001`.
- Post-MVP discovery tasks use `D<package>-<number>`, for example `D30-001`.
- IDs are never reused after merge. Cancelled work remains recorded with its reason.
- Checklist state is `[ ]` until the task is verified; execution status carries the detailed delivery state.

The task-level state vocabulary is:

`planned → ready → in_progress → review → verified`

Use `blocked` when a concrete dependency prevents progress, and record the unblock condition in the task and execution log.

## Required task fields

Each task must include:

1. Outcome and scope.
2. Product requirement and implementation evidence requirement references. List every
   applicable ID explicitly; do not use wildcards or ranges such as `PR-SEC-*` or
   `RQ-007…009` in task fields.
3. Dependencies on earlier task IDs or approved decisions.
4. Code, interface, schema, or documentation areas affected.
5. Verification command, test case, or manual scenario.
6. Acceptance condition and expected commit boundary.
7. Evidence anchor to the task's entry in `execution.md` for M1–M4 or `evidence.md` for M5+ discovery packages.

## Checklist format

```markdown
- [ ] T10-001 Introduce the application settings boundary
  - Product: PR-UX-01
  - Evidence: RQ-001
  - Depends on: —
  - Code: `app/main.py`, target `app/config.py`
  - Verify: `uv run pytest tests/test_core.py -q`
  - Accept when: the app factory uses temporary settings without changing the public entrypoint.
  - Commit: `refactor: introduce settings and application factory`
  - Evidence log: [T10-001](../10-m1-local-dashboard/10-application-foundation/execution.md#t10-001)
```

Planned paths must be labelled `target` until they exist. Do not turn a code example or target path into evidence. A link to an execution/evidence entry must resolve to an explicit task anchor in that file.

## Commit and review rules

- One commit has one coherent behavior or documentation purpose.
- A task may have one or more commits only when the plan explains the boundary.
- Run the narrowest relevant test after each commit and the required lint/check suite before review.
- Do not mark a task verified from a plan, target diagram, or unmerged branch.
- Record command, result, commit hash, reviewer, and known limitation in `execution.md` or `evidence.md`, according to the package type.
