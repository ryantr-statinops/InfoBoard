# Discovery tasks — Browser data portability

- [ ] <a id="d34-001"></a>**D34-001 — Select supported browser and data-class matrix**
  - Product source: [Next Plan](../../../../product/next-plan/00-browser-portability.md)
  - Verify: matrix separating portable content, credentials, session state, and storage
  - Evidence: [D34-001](evidence.md#d34-001)

- [ ] <a id="d34-002"></a>**D34-002 — Run read-only profile-copy portability prototype**
  - Product source: [Next Plan](../../../../product/next-plan/00-browser-portability.md)
  - Depends on: `D34-001`
  - Verify: import/export fixture with unsupported and corrupted records
  - Evidence: [D34-002](evidence.md#d34-002)

- [ ] <a id="d34-003"></a>**D34-003 — Complete consent, threat, and rollback gate**
  - Product source: [Next Plan](../../../../product/next-plan/00-browser-portability.md)
  - Depends on: `D34-002`
  - Verify: per-class consent UX, threat model, backup/restore, and acceptance proposal
  - Evidence: [D34-003](evidence.md#d34-003)
