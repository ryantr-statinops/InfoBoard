# InfoBoard implementation workspace

This workspace translates accepted product and architecture contracts into executable delivery packages. It owns sequencing, implementation checklists, status, and evidence; it never redefines canonical behavior.

Read in order:

1. Open the [implementation index](index.md) for delivery and review status.
2. Read [program control](00-program/README.md), especially the [conventions](00-program/conventions.md).
3. Follow the [roadmap](00-program/roadmap.md) and [dependency map](00-program/dependency-map.md).
4. Enter a milestone through its README, then use each package in the order README → plan → tasks → examples → references → execution.

## Document authority

Package `tasks.md` files contain the only authoritative checkboxes. Root and milestone tables are review rollups. Product, design, architecture, quality, and operations documentation remain canonical when implementation examples or plans differ.

Package status is `not_started`, `in_progress`, `blocked`, or `done`. No task is complete without matching, non-fabricated evidence.

## Change rule

When implementation exposes a contract gap, stop the affected task and update the upstream decision or specification first. Keep runtime evidence, target contract, and delivery state separate. Work on `dev`, push scoped commits to `origin/dev`, and do not merge `main`.
