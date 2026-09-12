# M4 release acceptance checklist

- [ ] Security limits, SSRF, escaping, Origin/Host, and log redaction pass.
- [ ] Backup/restore/rebuild of derived indexes is verified on a database copy.
- [ ] Health/degraded/diagnostics accurately represent each component.
- [ ] Core-mode clean checkout follows the README.
- [ ] Core tests, Ruff, migration tests, and benchmark report pass; full-mode smoke/evaluation is additionally required only when full mode is in release scope.
- [ ] Release, upgrade, rollback, and runbooks are reviewed.

**Evidence:** release checklist, CI run, benchmark report, and restore transcript.
