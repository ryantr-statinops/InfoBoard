InfoBoard IP-19 Cross-Platform Browser Verification Replay Guide
================================================================

This directory contains verification evidence across the six declared
Chrome/Edge x Linux/macOS/Windows desktop matrix cells.

Replay Commands:
----------------
1. Lifecycle and compatibility journeys:
   ./tests/compatibility/phase-19/run-cell --browser <chrome|edge> --os <linux|macos|windows> --reference <R-CLASS> --fixture-manifest fixtures/compatibility/phase-19/fixtures.json --journeys clean-install,startup,upgrade,reconnect,reset,uninstall --evidence-dir artifacts/verification/phase-19/<browser>-<os>

2. Performance hot-path measurement (NFR-001, NFR-002, NFR-003):
   ./tests/performance/phase-19/measure-hot-path --browser <chrome|edge> --os <linux|macos|windows> --reference <R-CLASS> --fixture fixtures/compatibility/phase-19/fixtures.json --tab-count 1000 --query p19-cross-platform-tab-activation-latency-benchmark-query-000000 --samples 101 --warmup 10 --seed p19 --output artifacts/verification/phase-19/<browser>-<os>/performance.json

3. Accessibility verification (NFR-008):
   ./tests/accessibility/phase-19/run --browser <chrome|edge> --os <linux|macos|windows> --reference <R-CLASS> --fixture fixtures/compatibility/phase-19/fixtures.json --output artifacts/verification/phase-19/<browser>-<os>/accessibility.json

4. Privacy & permission audit (NFR-009):
   ./tests/privacy/phase-19/audit --browser <chrome|edge> --os <linux|macos|windows> --reference <R-CLASS> --fixture fixtures/compatibility/phase-19/fixtures.json --manifest extension/manifest.json --output artifacts/verification/phase-19/<browser>-<os>/privacy.json

5. Failure recovery journeys (NFR-006):
   ./tests/recovery/phase-19/run --browser <chrome|edge> --os <linux|macos|windows> --reference <R-CLASS> --fixture fixtures/compatibility/phase-19/fixtures.json --output artifacts/verification/phase-19/<browser>-<os>/recovery.json

Note: No real user URLs, tab titles, queries, or secrets are recorded in any evidence file.
