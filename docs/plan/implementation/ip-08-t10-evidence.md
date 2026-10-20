# IP-08 T10 — Diagnostics redaction and acceptance evidence

T10 verifies that excluded content never reaches stored diagnostics or
diagnostic exports, and that health output exposes only bounded metadata.

## Excluded content classes (NFR-009)

| Class | Sentinel used in tests |
|---|---|
| Tab title | `My Secret Tab Title` |
| Full URL / query | `https://example.test/search?q=private+query` |
| SQL text | `SELECT * FROM ...` |
| Cookie / token | `session_token=abc123-secret` |
| File path | OS config path / Windows path |

## Enforcement points

| Layer | Mechanism | Test |
|---|---|---|
| Storage payload | `RedactDiagnosticPayload` replaces every byte outside `[a-zA-Z0-9_.-]` with `#`; over-long or invalid UTF-8 input redacts to empty | `TestRedactDiagnosticPayloadEliminatesExcludedContent` |
| Storage vocabulary | `ValidDiagnosticEventKind` (lowercase dotted), `ValidDiagnosticErrorClass` (closed snake_case), `ValidDiagnosticVersionLabel` (opaque label) reject all sentinels | `TestDiagnosticVocabularyRejectsExcludedContent` |
| Storage write path | `AppendDiagnostic` rejects unredacted or off-vocabulary records with `ErrorInvalidConfiguration`; nothing is silently rewritten | `TestAppendDiagnosticRejectsUnredactedContent` |
| Stored rows | `redacted_payload` re-validates as a redaction fixed point; JSON export contains no sentinel | `TestStoredDiagnosticsContainNoExcludedContent`, `TestStoredDiagnosticsHaveBoundedShape` |
| Runtime export | `Diagnostic` JSON keys pinned to the 16-key bounded allowlist; marshalled export contains no sentinel | `TestDiagnosticExportContainsOnlyBoundedKeys` |

## Requirement linkage

| Requirement | Evidence |
|---|---|
| FR-013 (lexical search available when storage fails) | Storage failures return `Degraded` + closed class; `TestSQL009UnavailableRepositoryReturnsDegradedDefaults` |
| FR-014 (profile-scoped configuration) | `TestSQL002ProfileIsolation`, profile-reset isolation in T07 |
| FR-015 (reset/uninstall removes only owned data) | `TestSQL010ResetAndUninstallPreserveUnownedFiles`, uninstall sentinel checks in T07 |
| NFR-006 (recovery without reinstall) | `TestSQL006CorruptDatabaseIsPreservedAndRebuilt`, quarantine retention in T05 |
| NFR-009 (no excluded data persisted/transmitted) | All T10 redaction tests above |

## Commands

```sh
go test ./host/storage/... ./host/internal/runtime/ ./tests/storage/... -count=1
```
