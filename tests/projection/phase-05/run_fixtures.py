#!/usr/bin/env python3
"""Validate the shared IP-05-T01 projection boundary corpus.

The runner proves the fixture contract only: every mandatory partition and fence
field is declared, every case states the boundary cause the declared requirement
sets derive, and the declared evidence matches the declared outcomes. Runtime
proof of the boundary lives in the Go and Node consumers that read this artifact.
"""

from __future__ import annotations

import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[3]
SUITE = 'phase-05'
FIXTURE_DIRECTORY = f'fixtures/projection/{SUITE}'
ARTIFACT = FIXTURE_DIRECTORY + '/phase-05.json'
EXPECTED = {
    'FX-PROJECTION-BOUNDARY-HANDOFF-IDENTITY',
    'FX-PROJECTION-BOUNDARY-REQUIRED-FIELDS',
    'FX-PROJECTION-BOUNDARY-SYNTHETIC-IDENTITY',
}
STAGE_BOUNDARY = 'boundary'
STAGE_PROJECTION = 'projection'
APPLY_NONE = 'none'
APPLY_RECORD = 'record'
UUID_V4 = re.compile(r'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$')
CASE_FIELDS = {
    'case_id', 'stage', 'cause', 'apply', 'payload', 'handoff', 'message_kind',
    'boundary_check', 'synthetic_fallback_probe', 'expected_outcome', 'expected_state',
}
STATE_FIELDS = {'epoch', 'revision', 'identity_count'}
CONTEXTS = {'normal', 'private'}
OBSERVATION_FIELDS = {
    'final_outcome', 'final_epoch', 'final_revision', 'final_identity_count',
    'accepted_cases', 'rejected_cases',
}


def boundary_errors(boundary: dict) -> list[str]:
    errors: list[str] = []
    mandatory = boundary['mandatory_partition_fields'] + boundary['mandatory_fence_fields']
    if len(set(mandatory)) != len(mandatory):
        errors.append('a mandatory field is declared twice')
    for field in mandatory:
        if field not in boundary['snapshot_record_required_fields']:
            errors.append(f'mandatory field {field} is absent from the record requirement set')
        if field not in boundary['identity_sources']:
            errors.append(f'mandatory field {field} declares no identity source')
    if len(boundary['identity_sources']) != len(mandatory):
        errors.append('an identity source is declared for a field that is not mandatory')
    for field in boundary['mandatory_partition_fields']:
        if field not in boundary['tab_identity_required_fields']:
            errors.append(f'tab identity does not require the mandatory field {field}')
    if boundary['synthetic_identity_fallback_allowed']:
        errors.append('the corpus permits a synthetic identity fallback')
    handoff = boundary['ip04_handoff_boundary']
    if (handoff['owner'], handoff['consumer']) != ('IP-04', 'IP-05'):
        errors.append('the handoff boundary is not bound to IP-04 and IP-05')
    for kind in ('snapshot', 'delta'):
        for field in mandatory:
            if field not in handoff['required_handoff_fields'][kind]:
                errors.append(f'{kind} handoff does not require the mandatory field {field}')
    if handoff['required_handoff_fields']['status'] != ['kind']:
        errors.append('a status handoff must require only its kind')
    for kind in handoff['handoff_kinds']:
        if kind not in handoff['handoff_kind_map']:
            errors.append(f'handoff kind {kind} has no declared message kind')
    return errors


def derived_cause(boundary: dict, item: dict) -> str | None:
    """Derive the boundary cause a case must declare from the requirement sets."""
    handoff = item.get('handoff')
    if handoff:
        check = item.get('boundary_check') or item.get('message_kind')
        for field in boundary['ip04_handoff_boundary']['required_handoff_fields'][check]:
            if field not in handoff:
                return f'missing_{field}'
        return None
    payload = item['payload']
    for field in boundary['snapshot_record_required_fields']:
        if field not in payload:
            return f'missing_{field}'
    identity = payload['tab_identity']
    for field in boundary['tab_identity_required_fields']:
        if field not in identity:
            return f'missing_tab_identity_{field}'
    return None


def undeclared_fields(boundary: dict, item: dict) -> list[str]:
    """Report fields the declared contract does not permit on a case payload."""
    handoff = item.get('handoff')
    if handoff:
        declared = boundary['ip04_handoff_boundary']
        check = item.get('boundary_check') or item.get('message_kind')
        allowed = declared['required_handoff_fields'][check] + declared['optional_handoff_fields']
        return [field for field in handoff if field not in allowed]
    allowed = boundary['snapshot_record_required_fields'] + boundary['optional_record_fields']
    extra = [field for field in item['payload'] if field not in allowed]
    identity = item['payload'].get('tab_identity')
    if isinstance(identity, dict):
        extra += [field for field in identity if field not in boundary['tab_identity_required_fields']]
    return extra


def fixture_errors(fixture: dict, boundary: dict) -> list[str]:
    errors: list[str] = []
    accepted_outcome = boundary['accepted_outcomes'][0]
    allowed_causes = {f'missing_{field}' for field in boundary['mandatory_partition_fields']}
    allowed_causes |= {f'missing_{field}' for field in boundary['mandatory_fence_fields']}
    allowed_causes |= {f'missing_tab_identity_{field}' for field in boundary['tab_identity_required_fields']}
    profile = fixture['input']['profile_id']
    if not UUID_V4.fullmatch(profile):
        errors.append(f'{fixture["fixture_id"]} binds a profile_id that is not an allocated profile identity')
    if not UUID_V4.fullmatch(fixture['input']['projection_epoch']):
        errors.append(f'{fixture["fixture_id"]} binds a projection_epoch that is not an allocated epoch')
    if fixture['input']['context_kind'] not in CONTEXTS:
        errors.append(f'{fixture["fixture_id"]} binds a context_kind outside the IP-02 context vocabulary')
    identity_fields = boundary['mandatory_partition_fields'] + boundary['mandatory_fence_fields']
    accepted_case = next((entry for entry in fixture['input']['cases'] if entry['expected_outcome'] == accepted_outcome and entry.get('payload')), None)
    previous: tuple[int, int] | None = None
    for index, item in enumerate(fixture['input']['cases']):
        label = f'{fixture["fixture_id"]}/{item["case_id"]}'
        if not set(item) <= CASE_FIELDS:
            errors.append(f'{label} has undeclared case fields')
        if set(item['expected_state']) != STATE_FIELDS:
            errors.append(f'{label} has an undeclared expected state')
        for field in undeclared_fields(boundary, item):
            errors.append(f'{label} carries undeclared field {field}')
        if item['stage'] == STAGE_BOUNDARY:
            cause = derived_cause(boundary, item)
            if item['cause'] != cause:
                errors.append(f'{label} cause {item["cause"]} disagrees with the derived cause {cause}')
            if item['cause'] not in allowed_causes:
                errors.append(f'{label} cause is not a missing mandatory identity field')
            if item['expected_outcome'] != boundary['fail_closed_outcome']:
                errors.append(f'{label} does not fail closed')
            if item['apply'] != APPLY_NONE:
                errors.append(f'{label} must not reach the projection stage')
            if not item['synthetic_fallback_probe']:
                errors.append(f'{label} must probe the synthetic fallback')
        elif item['stage'] == STAGE_PROJECTION:
            if derived_cause(boundary, item) is not None:
                errors.append(f'{label} does not satisfy the boundary it claims to satisfy')
            if item['cause'] is not None:
                errors.append(f'{label} must not declare a boundary cause')
            if item['apply'] != APPLY_RECORD:
                errors.append(f'{label} does not apply a record')
            if item['expected_outcome'] not in (accepted_outcome, boundary['fail_closed_outcome']):
                errors.append(f'{label} declares an outcome outside the declared vocabularies')
            if item['expected_outcome'] == boundary['fail_closed_outcome'] and accepted_case is not None:
                payload, reference = item['payload'], accepted_case['payload']
                differs = any(payload.get(field) != reference.get(field) for field in identity_fields)
                if not differs and payload['tab_identity'] != reference['tab_identity']:
                    differs = True
                if not differs:
                    errors.append(f'{label} declares the identity the accepted record carries, so its rejection is unprovable')
        else:
            errors.append(f'{label} declares an undeclared stage {item["stage"]}')
        state = item['expected_state']
        observed = (state['revision'], state['identity_count'])
        if index == 0:
            if previous is None:
                previous = observed
            if observed != (0, 0) and item['expected_outcome'] != accepted_outcome:
                errors.append(f'{label} must open on an uninitialized projection')
        elif observed != previous:
            errors.append(f'{label} declares a state transition no accepted case produced')
        if item['expected_outcome'] == accepted_outcome:
            previous = observed
    cases = fixture['input']['cases']
    observations = {observation['signal']: observation['value'] for observation in fixture['expected']['observations']}
    if set(observations) != OBSERVATION_FIELDS:
        errors.append(f'{fixture["fixture_id"]} declares an undeclared observation set')
    accepted = sum(1 for item in cases if item['expected_outcome'] == accepted_outcome)
    if observations.get('accepted_cases') != accepted:
        errors.append(f'{fixture["fixture_id"]} accepted_cases disagrees with its cases')
    if observations.get('rejected_cases') != len(cases) - accepted:
        errors.append(f'{fixture["fixture_id"]} rejected_cases disagrees with its cases')
    final = cases[-1]['expected_state']
    declared_final = {
        'final_outcome': cases[-1]['expected_outcome'],
        'final_epoch': final['epoch'],
        'final_revision': final['revision'],
        'final_identity_count': final['identity_count'],
    }
    for signal, value in declared_final.items():
        if observations.get(signal) != value:
            errors.append(f'{fixture["fixture_id"]} {signal} disagrees with its final case')
    return errors


def validate() -> tuple[list[str], list[tuple[str, str, str, int, int]]]:
    errors: list[str] = []
    document = json.loads((ROOT / ARTIFACT).read_text(encoding='utf-8'))
    fixtures = {fixture['fixture_id']: fixture for fixture in document['fixtures']}
    if document['schema_version'] != 1 or document['phase'] != 'IP-05' or document['artifact'] != 'projection-boundary':
        errors.append('artifact header disagrees with the phase corpus contract')
    if document['owner_task'] != 'IP-05-T01':
        errors.append('the corpus is not owned by the IP-05-T01 slice')
    if document['shared_consumers'] != ['go', 'node']:
        errors.append('the corpus must be shared by the Go and Node consumers')
    if len(fixtures) != len(document['fixtures']) or set(fixtures) != EXPECTED:
        errors.append('the corpus must contain exactly the IP-05-T01 boundary fixtures')
    for source in document['contract_source']:
        if not (ROOT / source).is_file():
            errors.append(f'contract source {source} is absent')
    for fixture_id in fixtures:
        if fixture_id in document['reserved_fixture_ids']:
            errors.append(f'{fixture_id} is reserved for a later IP-05 task')
    errors.extend(boundary_errors(document['boundary']))
    evidence: list[tuple[str, str, str, int, int]] = []
    for fixture_id in sorted(EXPECTED):
        fixture = fixtures[fixture_id]
        errors.extend(fixture_errors(fixture, document['boundary']))
        observations = {observation['signal']: observation['value'] for observation in fixture['expected']['observations']}
        evidence.append((
            fixture_id,
            str(observations['final_outcome']),
            str(observations['final_epoch']),
            observations['final_revision'],
            observations['final_identity_count'],
        ))
    return errors, evidence


def main() -> int:
    arguments = sys.argv[1:]
    if arguments[:1] == ['--help']:
        print(f'usage: {pathlib.Path(__file__).name} --suite {SUITE} --fixtures {FIXTURE_DIRECTORY} --strict')
        return 0
    if arguments != ['--suite', SUITE, '--fixtures', FIXTURE_DIRECTORY, '--strict']:
        print(f'FAIL: invalid arguments; run --suite {SUITE} --fixtures {FIXTURE_DIRECTORY} --strict')
        return 2
    errors, evidence = validate()
    for message in errors:
        print(f'FAIL: {message}')
    if errors:
        return 1
    for fixture_id, outcome, epoch, revision, identities in evidence:
        print(f'PASS: {fixture_id} outcome={outcome} epoch={epoch} revision={revision} identities={identities}')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())