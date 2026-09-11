# Discovery tasks — Cloud sync and multi-device

- [ ] <a id="d31-001"></a>**D31-001 — Define sync ownership and replica model**
  - Verify: decision record for ownership, change log, and local-mode independence
  - Evidence: [D31-001](evidence.md#d31-001)

- [ ] <a id="d31-002"></a>**D31-002 — Prototype offline conflict and recovery flows**
  - Depends on: `D31-001`
  - Verify: two-replica conflict transcript and restore scenario
  - Evidence: [D31-002](evidence.md#d31-002)

- [ ] <a id="d31-003"></a>**D31-003 — Complete security, cost, and readiness review**
  - Depends on: `D31-002`
  - Verify: threat model, key recovery, quota, export/delete, SLO, and acceptance proposal
  - Evidence: [D31-003](evidence.md#d31-003)
