# 03 — Delivery workflow

**Status:** `ready`
**Main branch:** `main`
**Working branch:** `dev`

## Feature cycle

1. Fetch `origin` and update local `main`.
2. Fast-forward `dev` from `main` and confirm the worktree is clean except for acknowledged user changes.
3. Select a `ready` feature and review its README, plan, tasks, references, and acceptance.
4. Implement one task/commit boundary at a time.
5. Run the narrow test and `uv run ruff check .` after each commit.
6. Push each commit to `origin/dev`; never force-push or merge directly to `main`.
7. Run full checks, update execution evidence, and report the result.
8. Stop at `review`; the user reviews and merges the PR.
9. After merge, fetch `origin/main`, fast-forward `dev`, and start the next feature.

## Commit rules

- One commit has one coherent behavior or documentation purpose.
- Use Conventional Commits: `feat`, `fix`, `test`, `docs`, `chore`, `security`, `ci`.
- Do not mix broad formatting with behavior changes.
- A migration is paired with upgrade tests or an explicitly dependent follow-up commit.
- Commit messages must be suitable for a changelog.

## Review report

Include feature/epic, acceptance result, ordered commit hashes, affected files/interfaces/schema, tests/lint/benchmark/manual smoke, known limitations, rollback command, `git status`, remote branch, and PR link.

## Rollback and ready definition

Rollback code through a revert PR and data through the backup/migration procedure in epic 18. Never delete local data without resolving the exact path and backup first.

A task is ready only when it has an owner, input/output contract, dependencies, test cases, failure behavior, acceptance, and commit boundary.
