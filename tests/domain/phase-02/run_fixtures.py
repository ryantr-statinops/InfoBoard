#!/usr/bin/env python3
import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[3]
EXPECTED = {
    'FX-PROFILE-ISOLATION', 'FX-IDENTITY-TIEBREAK', 'FX-PROJECTION-LIFECYCLE',
    'FX-REVISION-FENCE', 'FX-ACTIVATION-RETENTION', 'FX-OWNERSHIP-BOUNDARY',
}
UUID_V4 = re.compile(r'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$')

def main():
    if sys.argv[1:] != ['--suite', 'phase-02']:
        print('FAIL: invalid arguments')
        return 2
    document = json.loads((ROOT / 'fixtures/domain/phase-02/phase-02.json').read_text())
    fixtures = document.get('fixtures', [])
    by_id = {fixture.get('fixture_id'): fixture for fixture in fixtures}
    if document.get('schema_version') != 1 or set(by_id) != EXPECTED or len(fixtures) != len(EXPECTED):
        print('FAIL: fixture catalog contract')
        return 1
    checks = {
        'FX-PROFILE-ISOLATION': lambda x: len(x['input']['profile_ids']) == 2 and all(UUID_V4.fullmatch(value) for value in x['input']['profile_ids']) and x['input']['profile_ids'][0] != x['input']['profile_ids'][1],
        'FX-IDENTITY-TIEBREAK': lambda x: x['input']['same_display'] is True and len(set(x['input']['tab_ids'])) == len(x['input']['tab_ids']),
        'FX-PROJECTION-LIFECYCLE': lambda x: x['input']['add_tab_id'] not in x['input']['initial_tab_ids'] and x['input']['remove_tab_id'] in x['input']['initial_tab_ids'],
        'FX-REVISION-FENCE': lambda x: x['input']['gap_sequence'] > x['input']['initial_revision'] + 1 and x['input']['stale_epoch'] and x['input']['conflicting_duplicate'],
        'FX-ACTIVATION-RETENTION': lambda x: x['input']['max_records'] == 500 and x['input']['age_days'] == 30 and x['input']['private_records'] > 0,
        'FX-OWNERSHIP-BOUNDARY': lambda x: all(x['input'][key] is False for key in ('host_may_activate', 'ui_may_mutate_projection', 'sqlite_is_live_truth')),
    }
    for fixture_id in sorted(EXPECTED):
        if not checks[fixture_id](by_id[fixture_id]):
            print(f'FAIL: {fixture_id}')
            return 1
        print(f'PASS: {fixture_id}')
    return 0

if __name__ == '__main__':
    raise SystemExit(main())
