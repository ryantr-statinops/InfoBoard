# IP-18 evidence — Packaging and host registration

Status: released via PR #30 (merge commit `18b7be3`); implementation and phase acceptance checks passed.

## Delivered contract

- `packaging/config/` owns the single authoritative release source of truth (`release-manifest.json` and `extension-ids.json`). A release cannot be built when an ID, version, target, or checksum input is missing or contains an unresolved placeholder.
- `packaging/extension/` and `packaging/release/extension_package.py` build deterministic MV3 extension packages for Chrome Web Store and Edge Add-ons with exactly `nativeMessaging`, `storage`, `tabGroups`, and `tabs` permissions, `open-search` and `close-search` commands, and zero `host_permissions` or external network endpoints.
- `packaging/host/` and `packaging/release/host_manifest.py` render and validate browser-specific Native Messaging host manifests (`com.infoboard.host.json`). Manifests use fixed host name `com.infoboard.host`, `type: "stdio"`, an absolute platform executable path, and exactly one matching `chrome-extension://<id>/*` origin. Extra origins, broad origins, and unresolved tokens are rejected fail-closed.
- `packaging/installers/` and `packaging/release/lifecycle.py` implement safe, idempotent, per-user install, upgrade, repair, rollback, and uninstall operations for Linux (XDG), macOS (`Application Support`), and Windows (`HKCU`).
- Preflight integrity verification validates SHA-256 manifests before mutating user paths. Archives with path traversal (`..`), absolute paths, or symlinks are rejected before extraction.
- Upgrade preserves last-known-good backup and atomically publishes versioned binaries and manifests. If smoke checks fail, rollback restores the previous verified installation.
- Repair verifies and restores missing or corrupted InfoBoard-owned files. Uninstall cleans up only InfoBoard-owned paths recorded in `owned-paths.json` and treats already-absent resources as success without touching browser tabs, browser history, or user data.
- Compatibility negotiation enforces protocol version `1`, ranking model `rank-1`, and storage schema version `1`. Version mismatches fail closed before processing data.
- Deterministic archive assembly pins `SOURCE_DATE_EPOCH=0`, normalizes file order and permissions, and produces byte-identical artifacts and sorted `SHA256SUMS`.
- Twelve packaging fixtures (`FX-PKG-*`) are registered in `fixtures/catalog.json`; the catalog grew from 93 to 105 fixtures and retained 19 phase seams.

## Commands and results

```sh
SOURCE_DATE_EPOCH=0 python packaging/release/build.py --config packaging/config/release-manifest.json --output /tmp/infoboard-ip18-a
python packaging/release/verify.py --artifacts /tmp/infoboard-ip18-a --fixture-root fixtures/packaging/phase-18
python packaging/release/lifecycle_test.py --fixture-root fixtures/packaging/phase-18 --browsers chrome,edge --platforms linux,macos,windows --repeat 2
python -m unittest discover tests/packaging/phase-18/
python tests/implementation/validate_corpus.py
node --test tests/diagnostics/diagnostics-contract.test.mjs
npm run build:extension
go vet ./host/...
git diff --check
```

Observed:

- Clean build and verify: PASS; emitted Chrome/Edge extension archives, host payloads for linux/macos/windows, installer archives, `build-input.json`, and sorted `SHA256SUMS`.
- Verification runner (`verify.py`): PASS; checked extension packages, host manifests, installer archives, checksums, and verified that 9 negative fixtures (missing ID, unresolved placeholder, duplicate version, unsupported target, permission drift, extra origin, relative path) are rejected correctly.
- Lifecycle convergence runner (`lifecycle_test.py`): PASS; 6 matrix cells (Chrome/Edge × Linux/macOS/Windows) executed install (v1), repeat install (no-op), upgrade (v2), rollback (v1), repair, uninstall, and repeat uninstall with full convergence.
- Packaging unit suite: 56 tests (55 passed, 1 skipped).
- Corpus validator: PASS (`phases=20, tasks=236, requirements=25, fixtures=105, invariants=7, phase_seams=19`).
- Diagnostics contract test: PASS (5/5 tests).
- Extension build: PASS (`tsc` and asset copy clean).
- Go vet: PASS (no issues).
- `git diff --check`: PASS.

## Artifact verification summary

| Artifact | Channels / Targets | Exact permissions / Origins | Integrity |
|---|---|---|---|
| `infoboard-extension-chrome-v0.1.0.zip` | Chrome Web Store | `nativeMessaging`, `storage`, `tabGroups`, `tabs` | SHA-256 in `SHA256SUMS`, reproducible |
| `infoboard-extension-edge-v0.1.0.zip` | Edge Add-ons | `nativeMessaging`, `storage`, `tabGroups`, `tabs` | SHA-256 in `SHA256SUMS`, reproducible |
| `infoboard-host` / `infoboard-host.exe` | Linux, macOS, Windows (amd64) | Absolute binary path, stdio | SHA-256 in `SHA256SUMS`, reproducible |
| `com.infoboard.host.json` (Chrome) | Chrome NativeMessagingHosts | `["chrome-extension://abcdefghijklmnopabcdefghijklmnop/*"]` | Literal ID, absolute path |
| `com.infoboard.host.json` (Edge) | Edge NativeMessagingHosts | `["chrome-extension://bcdefghijklmnopabcdefghijklmnopa/*"]` | Literal ID, absolute path |
| Host installer archives | Linux, macOS, Windows | Per-user only, owned-paths cleanup | SHA-256 in `SHA256SUMS`, reproducible |

## Acceptance checklist

- [x] Release source of truth (`packaging/config/release-manifest.json`): validated against schema with literal IDs.
- [x] Chrome Web Store package: assembled MV3 archive with exact permissions and commands.
- [x] Edge Add-ons package: assembled MV3 archive matching Chrome contract without host permissions.
- [x] Host manifests: exact allowed origins, absolute executable paths, no broad origins.
- [x] Deterministic builds: byte-identical zip archives and sorted `SHA256SUMS`.
- [x] Preflight checks: checksum and archive path safety checks prevent unsafe extractions.
- [x] Install idempotency: repeated install is a verified no-op.
- [x] Upgrade & Rollback: transactional backup, rollback verification on health check failure.
- [x] Repair & Uninstall: restores missing owned files, clean uninstallation limited to owned paths.
- [x] Compatibility fail-closed: version mismatch halts execution before data transfer.
- [x] Fixture catalog seam: 12 new packaging fixtures registered in `catalog.json`, 105 total fixtures, 19 seams intact.

## Release status

- Released via PR #30; `main` contains merge commit `18b7be3`.
