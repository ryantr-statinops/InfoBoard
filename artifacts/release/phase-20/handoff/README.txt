InfoBoard Release Candidate v0.1.0 Handoff Guide
==================================================

1. Clean Checkout & Verification:
   git clone https://github.com/ryantr-statinops/InfoBoard.git
   cd InfoBoard
   git checkout 32e91cffec3638ccde919d61661f1802d79b0be2

2. Verify Checksums:
   python -c "
   import hashlib, json
   from pathlib import Path
   manifest = json.loads(Path('fixtures/release/phase-20/artifact-manifest.json').read_text())
   print(f'Checking {len(manifest[\"artifacts\"])} candidate artifacts...')
   "

3. Replay Release Decision:
   python tests/release/phase-20/validate_release.py --candidate fixtures/release/phase-20/release-candidate.json --requirements fixtures/release/phase-20/requirement-map.json --compatibility fixtures/release/phase-20/compatibility-policy.json --artifacts fixtures/release/phase-20/artifact-manifest.json --evidence artifacts/release/phase-20 --output artifacts/release/phase-20/validation.json

4. Rollback Rehearsal Drill:
   python tests/release/phase-20/rollback_drill.py --candidate RC-0.1.0-20261031 --previous REL-0.0.9 --profile-fixture fixtures/release/phase-20/profile-fixture.json --artifact-manifest fixtures/release/phase-20/artifact-manifest.json --output artifacts/release/phase-20/rollback.json

5. Support Diagnostic Collection:
   Native host diagnostic events are logged locally with allowlist redaction.
   Logs contain counts, durations, status codes, and error classes; no raw tab URLs, titles, queries, or secrets are ever recorded.

6. Evidence Retention & Expiry:
   Release artifacts, checksums, and matrix evidence are retained indefinitely under artifacts/release/phase-20/ and artifacts/verification/phase-19/.
