# URL validation and normalization: examples

Examples are synthetic and non-authoritative; the linked canonical contracts win on conflict.

## Contract or workflow example

`HTTPS://Example.COM:443/a/../b?utm_source=x&z=2&z=1#frag` normalizes deterministically to the policy-v1 identity while retaining both `z` values.

## Verification pattern

```bash
uv run pytest -q
uv run ruff check .
```

Add package-specific test selectors and failure-injection commands during implementation.

## Evidence example

```text
Task: CAP-URL-001
Commit: <git-sha>
Command: <exact command>
Result: PASS (<count and duration>)
Artifact: <path or report link>
Deviation: none
```
