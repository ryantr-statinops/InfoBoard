# Agent integration

This is repository workflow documentation, not product documentation. It explains how coding agents discover and route reusable skills; it does not define InfoBoard capabilities or implementation behavior.

## Purpose

`.agent/skills/` is the canonical portable skill root. A skill provides task-specific workflow guidance, references, or templates. Installation makes skills discoverable; it does not mean every task should load every skill.

## Routing policy

1. Match the task to the smallest skill set whose descriptions and boundaries apply.
2. Prefer repository conventions and the canonical product documents over generic skill defaults.
3. Keep product decisions in [`docs/plan/refactor/`](../plan/refactor/README.md).
4. Treat implementation planning as separate from product specification.
5. Do not activate unrelated domain skills merely because they are installed.

## Product boundary

Agent notes must not introduce alternate browser, privacy, runtime, storage, ranking, packaging, or release behavior. A proposed capability first belongs in the product contract, requirements, decision log, and acceptance criteria.

## Implementation boundary

This document does not authorize source changes or changes under [`docs/plan/implementation/`](../plan/implementation/README.md). Implementation work requires an explicit task and its own review scope.

## Maintenance

Update this file only when skill discovery, routing, or repository-agent boundaries change. Product changes belong in the refactor specification, not here.
