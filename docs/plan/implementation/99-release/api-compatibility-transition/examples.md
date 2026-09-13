# API compatibility transition: examples

Examples are synthetic and non-authoritative.

## Contract or workflow example

During the compatibility release, `/api/search` and `/api/v1/search` call the same service; only the old route adds deprecation headers.

## Verification pattern

```bash
uv sync --frozen
uv run ruff check .
uv run pytest -q
```

Add exact migration, browser, provider, recovery, and packaging commands during execution.

## Evidence example

```text
Task: RELEASE-COMPAT-001
Commit: <git-sha>
Environment: <clean platform>
Command: <exact command>
Result: PASS
Artifacts: <redacted paths>
Reviewer: <name or handle>
```
