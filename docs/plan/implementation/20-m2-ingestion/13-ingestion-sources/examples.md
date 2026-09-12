# Examples — Ingestion sources

## Target contract: ExtractedDocument

```python
class ExtractedDocument:
    title: str
    text: str
    source_type: str
    source_url: str | None
    original_filename: str | None
    metadata: dict[str, str]
```

The contract is target documentation. [T13-001](tasks.md#t13-001) must define the concrete runtime type and tests.

## Target contract: bounded extraction

```text
validate source → resolve destination → enforce timeout/size → extract → normalize → hash → enqueue
```

Any failed validation stops before item creation.
