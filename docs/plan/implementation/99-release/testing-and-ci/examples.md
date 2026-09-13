# Testing and CI: examples

Examples are synthetic and non-authoritative.

## Contract or workflow example

CI uses fake providers for deterministic failures; release evidence separately records one configured compatible-provider smoke without retaining query or content.

## Verification pattern

```bash
uv sync --frozen
uv run ruff check .
uv run pytest -q
```

Add exact migration, browser, provider, recovery, and packaging commands during execution.

## Evidence example

```text
Task: RELEASE-TEST-001
Commit: <git-sha>
Environment: <clean platform>
Command: <exact command>
Result: PASS
Artifacts: <redacted paths>
Reviewer: <name or handle>
```
