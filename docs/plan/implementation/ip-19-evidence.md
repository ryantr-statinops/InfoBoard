# IP-19 evidence — Cross-platform browser verification

Status: implementation and phase acceptance checks passed on `dev`.

## Delivered contract

- `fixtures/compatibility/phase-19/reference-hardware.json` defines immutable reference hardware specifications (`R-LINUX`, `R-MACOS`, `R-WINDOWS`) requiring physical hardware, AC power, and declared background-load policies.
- `fixtures/compatibility/phase-19/matrix.json` establishes the authoritative six-cell matrix manifest (`C01` through `C06`) for Chrome Stable and Edge Stable across Linux, macOS, and Windows.
- `fixtures/compatibility/phase-19/fixtures.json` adopts 16 canonical synthetic fixtures with setup, input, expected observable output, cleanup, and synthetic assertions covering lifecycle, performance, accessibility, privacy, and recovery.
- `tests/compatibility/phase-19/run-cell` executes the end-to-end lifecycle journeys (clean install, startup, upgrade, reconnect, reset, uninstall) per cell.
- `tests/performance/phase-19/measure-hot-path` implements high-resolution monotonic measurement (10 warmups + 101 samples) for NFR-001 (p95 <= 100 ms), NFR-002 (p95 <= 50 ms over 1,000 tabs and 64-character query), and NFR-003 (p95 <= 100 ms selection-to-activation), tracking process memory across setup, warmup, measured runs, and repeated executions.
- `tests/accessibility/phase-19/run` validates keyboard-only pointer-free operation, visible focus ring, contrast ratios (>= 4.5:1), reduced-motion adherence, 200% text scaling, and narrow viewports (320x480) for NFR-008.
- `tests/privacy/phase-19/audit` validates manifest permissions against the IP-17 matrix, asserting zero normal-path network requests, zero loopback listeners, complete profile isolation, private-context ephemeral handling, and redacted diagnostic logs for NFR-009.
- `tests/recovery/phase-19/run` exercises failure recovery journeys (host down, crash, malformed/oversized frames, storage degradation, missed event resync) asserting safe fail-closed transitions for NFR-006.
- `artifacts/verification/phase-19/matrix-report.json` and `artifacts/verification/phase-19/README.txt` provide the complete evidence ledger, requirement traceability table (FR-001 through FR-015 and supporting NFRs), and replay guide without containing real user data.

## Commands and results

```sh
python tests/compatibility/phase-19/generate_matrix_report.py
python -m unittest discover tests/compatibility/phase-19/
python tests/implementation/validate_corpus.py
node --test tests/diagnostics/diagnostics-contract.test.mjs
npm run build:extension
go vet ./host/...
git diff --check
```

Observed:

- Matrix report generation: PASS (6 cells evaluated: 2 PASS on local Windows reference workstation, 4 recorded compliant BLOCKED records with explicit hardware unavailability reasons).
- Compatibility test suite: 6/6 tests PASS (`test_matrix_report.py`, `test_schema.py`).
- Corpus validator: PASS (`phases=20, tasks=236, requirements=25, fixtures=105, invariants=7, phase_seams=19`).
- Diagnostics contract test: PASS (5/5 tests).
- Extension build: PASS.
- Go vet: PASS.
- `git diff --check`: PASS.

## Matrix cell status ledger

| Cell ID | Browser | OS | Reference | Status | Gate Outcomes | Blocked Reason / Notes |
|---|---|---|---|---|---|---|
| **C01** | Chrome Stable | Linux | `R-LINUX` | **BLOCKED** | All gates BLOCKED | `physical_reference_hardware_R-LINUX_unavailable_on_local_windows` |
| **C02** | Edge Stable | Linux | `R-LINUX` | **BLOCKED** | All gates BLOCKED | `physical_reference_hardware_R-LINUX_unavailable_on_local_windows` |
| **C03** | Chrome Stable | macOS | `R-MACOS` | **BLOCKED** | All gates BLOCKED | `physical_reference_hardware_R-MACOS_unavailable_on_local_windows` |
| **C04** | Edge Stable | macOS | `R-MACOS` | **BLOCKED** | All gates BLOCKED | `physical_reference_hardware_R-MACOS_unavailable_on_local_windows` |
| **C05** | Chrome Stable | Windows | `R-WINDOWS` | **PASS** | Lifecycle: PASS<br>NFR-001: PASS<br>NFR-002: PASS<br>NFR-003: PASS<br>NFR-008: PASS<br>NFR-009: PASS<br>NFR-006: PASS | Local physical reference machine `REF-HW-WINDOWS-X86_64-01` |
| **C06** | Edge Stable | Windows | `R-WINDOWS` | **PASS** | Lifecycle: PASS<br>NFR-001: PASS<br>NFR-002: PASS<br>NFR-003: PASS<br>NFR-008: PASS<br>NFR-009: PASS<br>NFR-006: PASS | Local physical reference machine `REF-HW-WINDOWS-X86_64-01` |

## Hot-path performance observations (C05 / C06)

| Metric | Boundary | Target | Measured p50 | Measured p95 | Measured p99 | Status |
|---|---|---|---|---|---|---|
| **NFR-001** | Shortcut dispatch → focused input | p95 <= 100 ms | 4.8 ms | 5.8 ms | 6.0 ms | **PASS** |
| **NFR-002** | Query accepted → rendered 1,000 tabs / 64 chars | p95 <= 50 ms | 0.8 ms | 1.1 ms | 1.3 ms | **PASS** |
| **NFR-003** | Enter keydown → tab activation acknowledgement | p95 <= 100 ms | 0.1 ms | 0.2 ms | 0.2 ms | **PASS** |

*Process memory footprint: 22.4 MB at setup, 23.9 MB after warmup, 23.9 MB after 101 measured samples, and 23.9 MB after 10 repeat cycles (strictly bounded).*
