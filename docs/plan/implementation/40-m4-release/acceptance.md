# M4 release acceptance checklist

- [ ] Security limits, SSRF, escaping, Origin/Host, and log redaction pass.
- [ ] Backup/restore/rebuild of derived indexes is verified on a database copy.
- [ ] Health/degraded/diagnostics accurately represent each component.
- [ ] Core/full clean checkout follows the README.
- [ ] Full tests, Ruff, migration tests, and benchmark report pass.
- [ ] Release, upgrade, rollback, and runbooks are reviewed.

**Evidence:** release checklist, CI run, benchmark report, and restore transcript.
