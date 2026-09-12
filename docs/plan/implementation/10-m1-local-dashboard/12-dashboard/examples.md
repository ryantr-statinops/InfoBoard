# Examples — Dashboard and item workspace

## Target contract: item creation response

```json
{
  "id": 42,
  "status": "inbox",
  "source_type": "text",
  "content_version": 1,
  "job_id": "job-42"
}
```

The response is a target contract example. It must be reconciled with the shared API contract and verified by [T12-002](tasks.md#t12-002).

## Current contract: local smoke command

```bash
uv run uvicorn app.main:app --reload
```

Use a temporary data path during testing; never treat the committed example as evidence of a completed UI flow.
