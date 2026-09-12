# Examples — Application foundation

## Current contract: public entrypoint

```bash
uv run uvicorn app.main:app --reload
```

This is a contract example, not proof that the target factory refactor is complete.

## Target contract: lifecycle shape

```python
@asynccontextmanager
async def lifespan(app: FastAPI):
    init_db(app.state.settings.database_path)
    worker = start_worker(app.state.settings)
    try:
        yield
    finally:
        worker.stop()
```

The snippet becomes evidence only through [T10-003](tasks.md#t10-003) and its lifecycle tests.
