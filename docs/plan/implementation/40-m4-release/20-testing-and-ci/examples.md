# Examples — Testing, performance, and CI

## Current contract: baseline checks

```bash
uv run pytest -q
uv run ruff check .
```

These commands are the current baseline. Passing them does not prove unfinished recovery, benchmark, or release gates.

## Target contract: benchmark record

```text
dataset=1000_items/10000_chunks warmup=true p50=…ms p95=…ms p99=…ms hardware=… model=…
```
