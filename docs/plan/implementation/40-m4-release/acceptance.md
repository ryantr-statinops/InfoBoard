# M4 release acceptance checklist

- [ ] Security limits, SSRF, escaping, Origin/Host và log redaction pass.
- [ ] Backup/restore/rebuild derived indexes verified trên DB copy.
- [ ] Health/degraded/diagnostics phản ánh đúng component.
- [ ] Core/full clean checkout chạy theo README.
- [ ] Full test, Ruff, migration test và benchmark report pass.
- [ ] Release/upgrade/rollback/runbook được review.

**Evidence:** release checklist, CI run, benchmark report và restore transcript.
