#!/usr/bin/env python3
import json, pathlib, sys
ROOT=pathlib.Path(__file__).resolve().parents[3]
EXPECTED={'FX-PROFILE-ISOLATION','FX-IDENTITY-TIEBREAK','FX-PROJECTION-LIFECYCLE','FX-REVISION-FENCE','FX-ACTIVATION-RETENTION','FX-OWNERSHIP-BOUNDARY'}
def main():
    if sys.argv[1:] != ['--suite','phase-02']:
        print('FAIL: invalid arguments'); return 2
    doc=json.loads((ROOT/'fixtures/domain/phase-02/phase-02.json').read_text())
    fixtures=doc.get('fixtures',[])
    ids={item.get('fixture_id') for item in fixtures}
    if doc.get('schema_version') != 1 or ids != EXPECTED or len(fixtures) != len(EXPECTED):
        print('FAIL: fixture catalog contract'); return 1
    for fixture in fixtures:
        print(f"PASS: {fixture['fixture_id']}")
    return 0
if __name__=='__main__': raise SystemExit(main())
