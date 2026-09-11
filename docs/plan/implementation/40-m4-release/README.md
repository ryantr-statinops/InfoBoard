# M4 — Hardening and release

**Gate:** MVP có thể backup, chẩn đoán, kiểm thử và phát hành lặp lại
**Dependencies:** M1–M3

## Epic packages

- [17 — Security](17-security/plan.md)
- [18 — Reliability](18-reliability/plan.md)
- [19 — Observability](19-observability/plan.md)
- [20 — Testing và CI](20-testing-and-ci/plan.md)
- [21 — Packaging và release](21-packaging/plan.md)

## Thứ tự

Security boundary và recovery contract trước; observability hỗ trợ test; packaging chỉ release sau quality gate `20`.

## Acceptance gate

Test/lint/benchmark pass, backup/restore/rebuild verified, security limits pass, health/degraded states rõ và clean checkout chạy được core/full mode.
