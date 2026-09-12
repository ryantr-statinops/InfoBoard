# Examples — Security and privacy

## Target contract: rejected input

```json
{
  "error": {
    "code": "ssrf_destination_blocked",
    "message": "The requested source is not available."
  }
}
```

The error must not reveal resolved IPs, filesystem paths, or credentials. Verify through [T17-002](tasks.md#t17-002).
