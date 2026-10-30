# IP-17 evidence - Privacy, permissions, and profile isolation

Status: implementation and executable phase checks passed on `dev` at `cbcb4eb`.
The branded Chrome/Edge matrix is recorded as **blocked** with an exact reason and
is not counted as a pass; the real matrix remains IP-19 release-gate work.

## Delivered contract

- `extension/src/privacy/` now owns the field-class/ownership matrix, approved
  permission and capability matrix, scope guard, private-context lifetime, shared
  redaction boundary, and bounded input/secret policy.
- `host/privacy/` now owns the matching Go policy for field classes, permissions,
  frame scope validation, private write rejection, redaction, secret rejection,
  and input limits.
- Every message, result, activation, setting, diagnostic, and reset path is bound
  to `(browser_family, profile_id, context_kind)`. Foreign profile/context data is
  rejected before projection, index, SQLite, or browser activation work.
- Private projection records and query input are ephemeral, never persisted, and
  cleared at private-context teardown. A delayed private response is rejected
  rather than falling back to normal state.
- Redaction is deny-by-default. Only bounded enums, counts, durations, sizes,
  versions, revisions, retryability, and safe status/error literals survive.
  Unknown fields are dropped; secret-shaped fields and values are rejected before
  persistence or export.
- Input limits reuse the IP-07 constants and enforce result limit `1..50` and the
  declared `1,000`-tab operating envelope before allocation or indexing.
- The audit covers permission set, API-call set, storage classes, retention limits,
  fixture hashes, profile-scoped activation retention, diagnostics age/byte
  retention, private-row absence, reset, uninstall, and excluded surfaces.
- Eleven IP-17 fixtures are registered in `fixtures/catalog.json`; the catalog
  grew from 82 to 93 fixtures and kept 19 phase seams.

## Commands and results

```sh
go test ./host/privacy/... -run 'Test(Profile|Private|Redaction|Secret|Input|Retention)' -count=1
go vet ./host/privacy/...
gofmt -l host/privacy
npm run build:extension
node --test tests/privacy/phase-17/*.test.mjs tests/diagnostics/diagnostics-contract.test.mjs
python tests/implementation/validate_corpus.py
python tests/privacy/phase-17/validate_permissions.py --manifest extension/manifest.json --fixtures fixtures/privacy/phase-17
python tests/privacy/phase-17/validate_redaction.py --fixtures fixtures/privacy/phase-17 --diagnostics fixtures/privacy/phase-17/output
python tests/privacy/phase-17/audit_retention.py --db fixtures/privacy/phase-17/retention.sqlite --max-age-days 30 --max-diagnostics-bytes 10000000
python tests/privacy/phase-17/run_browser_matrix.py --browsers chrome,edge --contexts normal,private --fixtures fixtures/privacy/phase-17
git diff --check
```

Observed:

- Targeted Go privacy suite: PASS (`ok github.com/ryantr-statinops/InfoBoard/host/privacy`).
- `go vet ./host/privacy/...`: PASS.
- `gofmt -l host/privacy`: no output.
- Extension build: PASS.
- Privacy and diagnostics Node suites: 29/29 PASS.
- Corpus validator: PASS (`phases=20, tasks=236, requirements=25, fixtures=93, invariants=7, phase_seams=19`).
- Permission audit: PASS; approved permissions are exactly
  `nativeMessaging`, `storage`, `tabGroups`, and `tabs`; observed API-call set is
  limited to `storage`; denied surfaces are absent.
- Redaction audit: PASS; 9 synthetic sentinels absent from 3 committed output files.
- Retention audit: PASS; activation counts `500` and `10`, diagnostics
  `100` rows / `10,000,000` payload bytes, private rows absent, reset
  profile-scoped, uninstall owned-tables-only.
- Browser matrix: **BLOCKED** 4/4 entries (`chrome`/`edge` × `normal`/`private`);
  reason `branded_extension_sideload_requires_managed_profile_or_manual_ui`;
  `summary.pass == 0` and no blocked entry is counted as a pass.
- `git diff --check`: PASS (line-ending warnings only).

## Permission matrix

| Capability | Permission / surface | Requirement | Denial behavior |
|---|---|---|---|
| Search-surface command | `commands: open-search`, `close-search` | FR-001, FR-003 | Surface stays closed; no collection. |
| Tab metadata | `tabs` | FR-003, FR-014 | Unavailable state; no fallback or broad host access. |
| Tab groups | `tabGroups` | FR-003, FR-014 | Group fields omitted; tab eligibility remains bounded. |
| Activation / window focus | `tabs` | FR-003, FR-014 | Reject stale or foreign target; no fallback activation. |
| Private context | Browser-granted private capability only | FR-014, NFR-009 | Report unavailable; never request broader access. |
| Native Messaging | `nativeMessaging` | NFR-007, NFR-009 | Local host unavailable; no remote transport. |
| Extension-owned profile key | `storage.local` only | FR-014, NFR-009 | No `storage.sync`; no account-derived identity. |
| History / bookmarks / downloads / cookies / page content / webRequest / external network | Denied | NFR-007, NFR-009 | Static and runtime audit fail closed. |

## Trust boundaries

| Boundary | Validation before use |
|---|---|
| Browser API → extension | Capability is approved; context is current; tab/group fields are bounded and typed. |
| Extension → Native Messaging | Frame/payload/query/result/snapshot/delta limits, scope identity, and projection revision. |
| Host → in-memory projection/index | `(browser_family, profile_id, context_kind)` match; foreign records and stale revisions rejected. |
| Host → SQLite | Normal scope only; private writes rejected at the repository seam; values parameterized. |
| Diagnostics/export → UI | Allowlist redaction; unknown fields dropped; record size and retention bounded. |
| Installer/manifest → host registration | Approved permission/command set only; excluded surfaces absent. |

## Abuse cases

| Case | Expected | Observed evidence |
|---|---|---|
| Cross-profile confused deputy | `PROFILE_MISMATCH`; no result/settings/row access; no activation | `profile-cross-read.json`, `profile-cross-activate.json`; Go/Node scope tests PASS. |
| Private-to-normal leakage | Zero private fields outside private scope | `private-separation.json`; private arena tests PASS. |
| Delayed private response | `PRIVATE_CONTEXT_ENDED`; no normal fallback | `private-teardown.json`; teardown tests PASS. |
| Permission denial | Unavailable state; no escalation | `permission-denied.json`; denied-manifest audit PASS. |
| Diagnostic exfiltration | Sentinels absent byte-for-byte | `redaction-sentinels.json`; redaction audit PASS. |
| Secret-shaped input | Rejected or dropped; no echo | `secret-rejection.json`; Go/Node secret tests PASS. |
| Oversized / malformed input | Typed limit or `INVALID_FRAME` before work | `input-limits.json`; Go/Node limit tests PASS. |
| Retention bypass | Age/count/byte bounds hold; private rows absent | `retention-audit.json`; Python audit PASS. |
| Excluded surface use | No excluded API, page read, network request, or transmission | `excluded-surface.json`; permission audit and browser matrix report PASS/BLOCKED as above. |

## Retention and ownership notes

- Activation metadata is profile-scoped, bounded to 500 rows and 30 days, and
  disabled for private context.
- Diagnostics apply the seven-day age rule and a global oldest-first payload-byte
  budget. The IP-17 audit applies the stricter `10,000,000`-byte contract cap.
- IP-08 owns the storage constant and currently defines it as `10 * 1024 * 1024`
  bytes (10 MiB). No storage change was made because that constant is an IP-08
  contract with existing tests. Residual risk: if the release contract requires
  exactly `10,000,000` bytes at runtime, align the IP-08 constant before release.
- Reset removes only the requested profile's owned rows; the audit leaves the
  other profile and installation state intact. Uninstall removes only owned
  tables; unrelated browser data remains.
- The audit reports fixture hashes only, not raw titles, URLs, queries, tokens, or
  private identifiers.

## Fixtures registered

`FX-PRIVACY-PERMISSION-MATRIX`, `FX-PRIVACY-PROFILE-CROSS-READ`,
`FX-PRIVACY-PROFILE-CROSS-ACTIVATE`, `FX-PRIVACY-PRIVATE-SEPARATION`,
`FX-PRIVACY-PRIVATE-TEARDOWN`, `FX-PRIVACY-PERMISSION-DENIED`,
`FX-PRIVACY-REDACTION`, `FX-PRIVACY-SECRET-REJECTION`,
`FX-PRIVACY-INPUT-LIMITS`, `FX-PRIVACY-RETENTION-AUDIT`, and
`FX-PRIVACY-EXCLUDED-SURFACE`.

## Commits

```text
7e5726e feat(privacy): implement ip-17-t01
0d5bc0f test(privacy): implement ip-17-t02
35a5e1c feat(privacy): implement ip-17-t03
53e5698 feat(privacy): implement ip-17-t04
a16f394 feat(privacy): implement ip-17-t05
accef92 feat(privacy): implement ip-17-t06
00679d6 test(privacy): implement ip-17-t07
cbcb4eb feat(privacy): implement ip-17-t08
```

## Known environment limits and non-goals

- Windows Application Control intermittently blocks Go test binaries. The targeted
  IP-17 suite passed; a broad `go test ./host/... ./tests/...` run remains
  unreliable for that environment reason.
- Docker Desktop's Linux engine is not running, so a full Linux-container Go suite
  was not available for this phase.
- Branded Chrome/Edge sideload and private-context UI journeys require a managed
  profile or manual UI. The matrix records `blocked` with the exact reason; real
  branded verification remains IP-19 work.
- IP-17 delivers policy modules and evidence without wiring them into the existing
  runtime composition. Full `host/cmd` composition remains IP-18/IP-20 work.
- No release PR was opened. `dev` remains the integration branch at `cbcb4eb`.
