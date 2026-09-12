# M4 — Hardening and release

**Outcome:** The MVP can be secured, diagnosed, recovered, tested, and released repeatedly.
**Gate:** Security, recovery, observability, quality, and clean-checkout evidence is complete.
**Dependencies:** M1–M3

## Epic packages

- [17 — Security](17-security/README.md)
- [18 — Reliability](18-reliability/README.md)
- [19 — Observability](19-observability/README.md)
- [20 — Testing and CI](20-testing-and-ci/README.md)
- [21 — Packaging and release](21-packaging/README.md)

## Delivery order

Security boundaries and recovery contracts come first; observability supports testing; packaging releases only after quality gate `20`.

## Acceptance gate

Tests, lint, and benchmark pass; backup/restore/rebuild is verified; security limits pass; health/degraded states are truthful; and a clean checkout runs core mode. Full-mode installation and semantic evidence are required only when full mode is included in the release scope; otherwise missing full-mode dependencies must be reported clearly.
