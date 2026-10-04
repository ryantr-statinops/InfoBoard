#!/usr/bin/env python3
"""Validate the shared IP-05 projection corpora.

The runner proves the fixture contract only. For the IP-05-T01 boundary corpus it
checks that every mandatory partition and fence field is declared, that every case
states the boundary cause the declared requirement sets derive, and that the declared
evidence matches the declared outcomes. For the IP-05-T02 snapshot-authority corpus it
checks that the reserved fixture ID is claimed exactly once, that each case's declared
acquisition follows from the data it declares, that no non-authoritative case could be
read as an empty authority, and that the declared evidence matches the cases. Runtime
proof lives in the Go and Node consumers that read these artifacts: the Go consumer
drives the real host boundary in host/internal/projection and the Node consumer drives
the real producer acquisition in extension/src/projection/reconciler.ts.
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
AUTHORITY_ARTIFACT = FIXTURE_DIRECTORY + '/snapshot-authority.json'
EXPECTED = {
    'FX-PROJECTION-BOUNDARY-HANDOFF-IDENTITY',
    'FX-PROJECTION-BOUNDARY-REQUIRED-FIELDS',
    'FX-PROJECTION-BOUNDARY-SYNTHETIC-IDENTITY',
}
AUTHORITY_FIXTURE_ID = 'FX-PROJECTION-SNAPSHOT-AUTHORITY'
AUTHORITY_OWNER_TASK = 'IP-05-T02'
AUTHORITY_ARTIFACT_NAME = 'projection-snapshot-authority'
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
AUTHORITY_CASE_FIELDS = {
    'case_id', 'acquisition_signal', 'partition_context', 'handoff', 'acquisition',
    'expected', 'note', 'record_expansion',
}
AUTHORITY_CASE_REQUIRED = {
    'case_id', 'acquisition_signal', 'partition_context', 'handoff', 'acquisition',
    'expected', 'note',
}
AUTHORITY_EXPANSION_FIELDS = {'template', 'count', 'identity_path', 'identity_start', 'identity_step', 'expansion_note'}
AUTHORITY_PRODUCER_FIELDS = {
    'outcome', 'reason', 'authoritative', 'resync_required', 'record_count',
    'explicitly_empty', 'retained',
}
AUTHORITY_HOST_FIELDS = {'call', 'validation', 'cause', 'code', 'resync', 'index', 'records'}
AUTHORITY_OBSERVATIONS = {
    'final_outcome', 'authoritative_cases', 'snapshot_required_cases',
    'profile_mismatch_cases', 'explicitly_empty_cases', 'authoritative_record_count',
    'non_authoritative_record_count', 'partial_outputs', 'committed_queryable_partitions',
}
AUTHORITY_OUTCOME_SIGNALS = {
    'AUTHORITATIVE': 'authoritative_cases',
    'SNAPSHOT_REQUIRED': 'snapshot_required_cases',
    'PROFILE_MISMATCH': 'profile_mismatch_cases',
}
AUTHORITY_SIGNALS = ('completed', 'denied', 'failed', 'unsupported', 'incomplete', 'disconnect')
AUTHORITY_REASONS = (
    'missing_records', 'missing_context_kind', 'missing_projection_epoch',
    'snapshot_resync_required', 'non_snapshot_handoff', 'unpartitioned_handoff',
    'invalid_record', 'partition_profile_mismatch', 'snapshot_bounds_exceeded',
)
AUTHORITY_DEFERRED_OWNERS = ('IP-05-T03', 'IP-05-T04', 'IP-05-T05', 'IP-05-T07', 'IP-05-T09', 'IP-05-T12')
# Every field the producer fence requires on any handoff kind, plus the optional
# fields the boundary artifact declares, is what a case may state on its handoff.
AUTHORITY_FENCE_FIELDS = (
    'previous_revision', 'event_sequence', 'kind',
)
AUTHORITY_COMMITTED_FIELDS = {'status', 'epoch', 'revision', 'count', 'queryable'}
AUTHORITY_RETAINED_FIELDS = {'authoritative_case', 'after_first_refusal'}


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


def authority_handoff_fields(boundary: dict) -> set[str]:
    """Return the fields one declared handoff may state, whatever its kind.

    The boundary artifact requires only the kind of an unallocated status handoff,
    while the producer fence requires the allocated partition and lineage on every
    handoff it admits. A case therefore declares the kind, the boundary artifact's
    optional fields, and the allocated fence fields.
    """
    handoff = boundary['ip04_handoff_boundary']
    allowed = set(handoff['optional_handoff_fields']) | set(AUTHORITY_FENCE_FIELDS)
    allowed |= set(boundary['mandatory_partition_fields'])
    allowed |= set(boundary['mandatory_fence_fields'])
    return allowed


def declared_records(handoff: dict) -> list[dict] | None:
    """Return the record list a handoff declares, or None when it declares none."""
    records = handoff.get('records')
    return records if isinstance(records, list) else None


def authority_record_errors(boundary: dict, handoff: dict, record: dict, label: str) -> list[str]:
    """Report what a declared record would let a consumer infer about authority.

    The checks are the declared record preconditions of the corpus, not a second copy
    of the IP-02 record contract: a record that cannot state the snapshot identity is
    evidence that the snapshot is not a clean read of the current partition.
    """
    errors: list[str] = []
    allowed = set(boundary['snapshot_record_required_fields']) | set(boundary['optional_record_fields'])
    errors += [f'{label} carries undeclared field {field}' for field in record if field not in allowed]
    identity = record.get('tab_identity')
    if not isinstance(identity, dict):
        errors.append(f'{label} states no tab identity')
        return errors
    errors += [
        f'{label} tab identity carries undeclared field {field}'
        for field in identity
        if field not in boundary['tab_identity_required_fields']
    ]
    for field in boundary['tab_identity_required_fields']:
        if field not in identity:
            errors.append(f'{label} cannot state the mandatory tab identity field {field}')
    for field in boundary['mandatory_partition_fields']:
        if field not in record:
            errors.append(f'{label} cannot state the mandatory partition field {field}')
    for field in boundary['mandatory_fence_fields']:
        if field not in record:
            errors.append(f'{label} cannot state the mandatory fence field {field}')
    if record.get('projection_epoch') != handoff.get('projection_epoch'):
        errors.append(f'{label} does not repeat the declared snapshot epoch')
    for field in boundary['mandatory_partition_fields']:
        if record.get(field) != handoff.get(field):
            errors.append(f'{label} names another {field}')
        if identity.get(field) != handoff.get(field):
            errors.append(f'{label} tab identity names another {field}')
    tab_id = identity.get('tab_id')
    if isinstance(tab_id, bool) or not isinstance(tab_id, int) or tab_id < 0 or tab_id > 9007199254740991:
        errors.append(f'{label} states no bounded browser tab id')
    if record.get('projection_revision') != handoff.get('projection_revision'):
        errors.append(f'{label} does not repeat the declared snapshot revision')
    if record.get('eligible') is not True:
        errors.append(f'{label} is not an eligible record')
    return errors


def producer_reason(item: dict) -> str | None:
    """Return the bounded reason one case declares for the real producer acquisition."""
    return item.get('expected', {}).get('producer', {}).get('reason')


def authority_expansion_errors(boundary: dict, authority: dict, handoff: dict, expansion: dict, label: str) -> list[str]:
    """Validate a bounded record-list expansion instead of a literal record body.

    An over-limit read cannot be written as fixture text, so the artifact declares the
    single record template and the count that materializes it. The expansion must be
    deterministic, must substitute only the browser tab id, and must ask for more
    records than the declared bound so the size gate is the only thing that can refuse it.
    """
    errors: list[str] = []
    if set(expansion) != AUTHORITY_EXPANSION_FIELDS:
        errors.append(f'{label} has undeclared record_expansion fields')
    if not isinstance(expansion.get('count'), int) or isinstance(expansion.get('count'), bool):
        errors.append(f'{label} declares a non-integer record_expansion count')
        return errors
    bound = authority.get('max_snapshot_records')
    if not isinstance(bound, int) or isinstance(bound, bool) or bound <= 0:
        errors.append(f'{label} relies on an undeclared snapshot record bound')
        return errors
    if expansion['count'] <= bound:
        errors.append(f'{label} expands {expansion["count"]} records, which does not exceed the declared bound {bound}')
    if expansion.get('identity_path') != 'tab_identity.tab_id':
        errors.append(f'{label} substitutes identity field {expansion.get("identity_path")}, which is not the browser tab id')
    for field in ('identity_start', 'identity_step'):
        value = expansion.get(field)
        if not isinstance(value, int) or isinstance(value, bool) or value < 0:
            errors.append(f'{label} declares an unusable identity_{field.removeprefix("identity_")}')
    if not str(expansion.get('expansion_note', '')).strip():
        errors.append(f'{label} states no expansion note')
    template = expansion.get('template')
    if not isinstance(template, dict):
        errors.append(f'{label} declares no record template')
        return errors
    errors += authority_record_errors(boundary, handoff, template, f'{label}/template')
    identity = template.get('tab_identity')
    start, step = expansion.get('identity_start'), expansion.get('identity_step')
    if isinstance(identity, dict) and isinstance(start, int) and isinstance(step, int) and step > 0:
        highest = start + step * (expansion['count'] - 1)
        if highest > 9007199254740991:
            errors.append(f'{label} expands a tab id past the IP-02 projection bound')
    return errors


def derived_authority(item: dict, boundary: dict, profile_id: str) -> bool:
    """Derive from the declared data alone whether the case could be authority.

    The derivation mirrors the declared acquisition conditions and nothing more: the
    runtime decision belongs to the real producer acquisition, which the Node consumer
    compares with this declaration. A case whose own data cannot satisfy every
    condition must not declare authority, and one whose data does satisfy them must,
    so a declared outcome can never disagree with the fixture it belongs to.
    """
    handoff = item['handoff']
    fence = set(boundary['mandatory_partition_fields']) | set(boundary['mandatory_fence_fields'])
    fence |= set(AUTHORITY_FENCE_FIELDS) - {'kind'}
    if any(field not in handoff for field in fence):
        return False
    if handoff.get('profile_id') != profile_id:
        return False
    if handoff.get('kind') != 'snapshot' or handoff.get('resync_required') is not False:
        return False
    if item.get('record_expansion') is not None:
        # A generated list past the declared bound exists only to be refused by the
        # size gate, so it can never stand as an authoritative read.
        return False
    records = declared_records(handoff)
    if records is None:
        return False
    label = f'{item["case_id"]}'
    return not any(authority_record_errors(boundary, handoff, record, f'{label}/record[{index}]') for index, record in enumerate(records))


def authority_case_errors(item: dict, document: dict, boundary: dict) -> list[str]:
    """Report one snapshot-authority case that disagrees with the declared contract."""
    errors: list[str] = []
    authority = document['authority']
    label = item['case_id']
    handoff = item['handoff']
    if not set(item) <= AUTHORITY_CASE_FIELDS:
        errors.append(f'{label} has undeclared case fields')
    if not AUTHORITY_CASE_REQUIRED <= set(item):
        errors.append(f'{label} omits a required case field')
    if not str(item.get('note', '')).strip():
        errors.append(f'{label} declares no note')
    if item['acquisition'] not in authority['outcome_vocabulary']:
        errors.append(f'{label} declares acquisition {item["acquisition"]} outside the declared vocabulary')
    if item['partition_context'] not in CONTEXTS:
        errors.append(f'{label} binds context {item["partition_context"]} outside the IP-02 vocabulary')
    if item['acquisition'] == authority['authoritative_outcome'] and item['partition_context'] != document['input']['context_kind']:
        errors.append(f'{label} declares authority for a context the corpus does not bind')
    handoff_boundary = boundary['ip04_handoff_boundary']
    if handoff.get('kind') not in handoff_boundary['handoff_kinds']:
        errors.append(f'{label} declares handoff kind {handoff.get("kind")} outside the IP-04 vocabulary')
    allowed = authority_handoff_fields(boundary)
    errors += [f'{label} carries undeclared handoff field {field}' for field in handoff if field not in allowed]
    records = declared_records(handoff)
    record_errors = [
        message
        for index, record in enumerate(records or [])
        for message in authority_record_errors(boundary, handoff, record, f'{label}/record[{index}]')
    ]
    expansion = item.get('record_expansion')
    if expansion is not None:
        if not isinstance(expansion, dict):
            errors.append(f'{label} declares a record_expansion that is not an object')
            expansion = None
        else:
            errors += authority_expansion_errors(boundary, authority, handoff, expansion, label)
            if item['acquisition'] == authority['authoritative_outcome']:
                errors.append(f'{label} declares authority for a read past the declared record bound')
            if producer_reason(item) != 'snapshot_bounds_exceeded':
                errors.append(f'{label} declares a size-gate case that does not prove the size gate')
    expected = item['expected']
    producer, host = expected['producer'], expected['host']
    if item['acquisition'] == authority['authoritative_outcome']:
        # An authoritative read must declare records that satisfy every precondition.
        errors += record_errors
    elif producer['reason'] == 'invalid_record' and not record_errors:
        errors.append(f'{label} declares an invalid-record refusal with no defective record')
    if set(producer) != AUTHORITY_PRODUCER_FIELDS:
        errors.append(f'{label} has undeclared producer fields')
    if set(host) != AUTHORITY_HOST_FIELDS:
        errors.append(f'{label} has undeclared host fields')
    if producer['outcome'] != item['acquisition'] or producer['authoritative'] != (item['acquisition'] == authority['authoritative_outcome']):
        errors.append(f'{label} producer expectation disagrees with the declared acquisition')
    if item['acquisition'] == authority['authoritative_outcome']:
        if producer['record_count'] != len(records or []):
            errors.append(f'{label} producer record count disagrees with its declared record list')
        if producer['explicitly_empty'] != (producer['record_count'] == 0):
            errors.append(f'{label} declares an empty flag that disagrees with its record count')
    elif producer['explicitly_empty']:
        errors.append(f'{label} declares an empty read it never completed')
    if producer['resync_required'] != (item['acquisition'] != authority['authoritative_outcome']):
        errors.append(f'{label} declares a resync requirement that disagrees with its acquisition')
    if producer['retained'] != (item['acquisition'] == authority['authoritative_outcome']):
        errors.append(f'{label} declares retention that disagrees with its acquisition')
    if item['acquisition'] == authority['authoritative_outcome']:
        if producer['reason'] is not None:
            errors.append(f'{label} declares a reason for an authoritative acquisition')
        if records is None:
            errors.append(f'{label} must declare its record list explicitly to be authority')
    else:
        # A failed, denied, partial, or unavailable read must publish nothing, so it
        # can never be read as a converged empty projection.
        if producer['reason'] not in authority['reject_reason_vocabulary']:
            errors.append(f'{label} declares reason {producer["reason"]} outside the declared vocabulary')
        if producer['record_count'] != 0 or producer['explicitly_empty'] or producer['retained']:
            errors.append(f'{label} declares records in a non-authoritative acquisition')
    if host['call'] not in ('snapshot', 'handoff'):
        errors.append(f'{label} declares unknown host call {host["call"]}')
    if host['call'] == 'handoff' and handoff.get('kind') == 'snapshot':
        errors.append(f'{label} snapshot handoff must bind the snapshot call')
    if host['validation'] == 'REFUSED':
        if host['cause'] is None or host['code'] is None or host['index'] is None:
            errors.append(f'{label} refused call declares no cause, code, or position')
        if host['records'] != 0:
            errors.append(f'{label} refused call declares records')
    elif host['validation'] == 'ADMITTED':
        if host['cause'] is not None or host['code'] is not None or host['index'] is not None:
            errors.append(f'{label} admitted call declares a cause, code, or position')
        if host['records'] != len(records or []):
            errors.append(f'{label} admitted call declares {host["records"]} records for {len(records or [])} declared')
    else:
        errors.append(f'{label} declares host validation {host["validation"]}')
    return errors


def authority_errors(document: dict, boundary_document: dict) -> tuple[list[str], dict]:
    """Validate the IP-05-T02 snapshot-authority corpus against its own contract."""
    errors: list[str] = []
    boundary = boundary_document['boundary']
    authority = document['authority']
    if document['schema_version'] != 1 or document['phase'] != 'IP-05' or document['artifact'] != AUTHORITY_ARTIFACT_NAME:
        errors.append('authority artifact header disagrees with the phase corpus contract')
    if document['owner_task'] != AUTHORITY_OWNER_TASK:
        errors.append('the authority corpus is not owned by the IP-05-T02 slice')
    if document['fixture_id'] != AUTHORITY_FIXTURE_ID:
        errors.append('the authority corpus does not carry the reserved fixture ID')
    if document['shared_consumers'] != ['go', 'node']:
        errors.append('the authority corpus must be shared by the Go and Node consumers')
    if not document['requirement_ids'] or not str(document['note']).strip() or not str(document['outcome_note']).strip():
        errors.append('the authority corpus declares no requirement, note, or outcome note')
    for source in document['contract_source']:
        if not (ROOT / source).is_file():
            errors.append(f'authority contract source {source} is absent')
    if not document['expected']['privacy_assertions']:
        errors.append('the authority corpus declares no privacy assertion')

    claims = document['claims']
    if claims['boundary_artifact'] != ARTIFACT:
        errors.append('the authority claim names another boundary artifact')
    if claims['reserved_fixture_ids'] != [AUTHORITY_FIXTURE_ID]:
        errors.append('the authority corpus must claim exactly the reserved fixture ID')
    if not str(claims.get('claim_note', '')).strip():
        errors.append('the authority corpus states no claim note')
    if AUTHORITY_FIXTURE_ID not in boundary_document['reserved_fixture_ids']:
        errors.append(f'{AUTHORITY_FIXTURE_ID} is not reserved by the boundary corpus')
    implemented = {fixture['fixture_id'] for fixture in boundary_document['fixtures']}
    if AUTHORITY_FIXTURE_ID in implemented:
        errors.append(f'{AUTHORITY_FIXTURE_ID} is already implemented by the boundary corpus')
    if implemented & set(claims['reserved_fixture_ids']):
        errors.append('the authority corpus claims a fixture the boundary corpus already implements')

    vocabulary = authority['outcome_vocabulary']
    if authority['authoritative_outcome'] not in vocabulary or set(vocabulary) != set(AUTHORITY_OUTCOME_SIGNALS):
        errors.append('the authority outcome vocabulary is not the bounded acquisition vocabulary')
    for field in ('required_conditions', 'record_preconditions', 'reject_reason_vocabulary', 'resync_signals', 'deferred_clauses'):
        if not authority[field]:
            errors.append(f'the authority corpus declares no {field}')
    if not isinstance(authority.get('max_snapshot_records'), int) or authority.get('max_snapshot_records', 0) <= 0:
        errors.append('the authority corpus declares no snapshot record bound')
    if not any(condition.get('id') == 'bounded_record_count' for condition in authority['required_conditions']):
        errors.append('the authority corpus declares no record-count condition')
    if not str(authority.get('host_limit_note', '')).strip():
        errors.append('the authority corpus states no host limit note')
    for condition in authority['required_conditions']:
        if not condition.get('id') or not str(condition.get('rule', '')).strip():
            errors.append(f'authority condition {condition} declares no id or rule')
    for precondition in authority['record_preconditions']:
        if not precondition.get('field') or not precondition.get('owner') or not str(precondition.get('rule', '')).strip():
            errors.append(f'authority record precondition {precondition} declares no field, owner, or rule')
    for signal, effect in authority['resync_signals'].items():
        if not signal.strip() or not str(effect).strip():
            errors.append(f'authority resync signal {signal} declares no effect')
    for field in ('producer_signal_note', 'host_field_note'):
        if not str(authority.get(field, '')).strip():
            errors.append(f'the authority corpus states no {field}')
    for reason in authority['reject_reason_vocabulary']:
        if reason not in {f'missing_{field}' for field in boundary['mandatory_partition_fields'] + boundary['mandatory_fence_fields']}:
            # A reason outside the missing-field set must still be a bounded reject
            # reason the producer declares, which the Node consumer checks against the
            # real vocabulary; here only its spelling is validated.
            if not re.fullmatch(r'[a-z][a-z_]*', reason):
                errors.append(f'authority reject reason {reason} is not a bounded code')
    owners = {clause['owner_task'] for clause in authority['deferred_clauses']}
    for clause in authority['deferred_clauses']:
        if not re.fullmatch(r'IP-05-T\d{2}', str(clause.get('owner_task', ''))) or clause['owner_task'] == AUTHORITY_OWNER_TASK:
            errors.append(f'deferred clause {clause.get("clause")} names owner {clause.get("owner_task")}, want a later IP-05 task')
        if not str(clause.get('reason', '')).strip():
            errors.append(f'deferred clause {clause.get("clause")} declares no reason')
    for owner in AUTHORITY_DEFERRED_OWNERS:
        if owner not in owners:
            errors.append(f'the authority corpus defers no clause to {owner}')

    for field in ('profile_id', 'projection_epoch', 'unbound_profile_id', 'foreign_epoch'):
        if not UUID_V4.fullmatch(str(document['input'][field])):
            errors.append(f'authority input {field} is not an allocated identity')
    if document['input']['context_kind'] not in CONTEXTS:
        errors.append('authority input context_kind is outside the IP-02 context vocabulary')
    if document['input']['unbound_profile_id'] == document['input']['profile_id']:
        errors.append('the authority corpus declares no foreign profile to refuse')
    if document['input']['foreign_epoch'] == document['input']['projection_epoch']:
        errors.append('the authority corpus declares no foreign epoch to refuse')

    cases = document['cases']
    if not cases:
        errors.append('the authority corpus declares no case')
    seen: set[str] = set()
    for item in cases:
        if item['case_id'] in seen:
            errors.append(f'{item["case_id"]} appears more than once')
        seen.add(item['case_id'])
        errors.extend(authority_case_errors(item, document, boundary))
        declared = derived_authority(item, boundary, document['input']['profile_id'])
        if declared != (item['acquisition'] == authority['authoritative_outcome']):
            errors.append(f'{item["case_id"]} acquisition {item["acquisition"]} disagrees with its own declared data')

    observations = {observation['signal']: observation['value'] for observation in document['expected']['observations']}
    if set(observations) != AUTHORITY_OBSERVATIONS:
        errors.append('the authority corpus declares an undeclared observation set')
    tallies: dict[str, int] = {}
    for outcome, signal in AUTHORITY_OUTCOME_SIGNALS.items():
        tallies[outcome] = sum(1 for item in cases if item['acquisition'] == outcome)
        if tallies[outcome] == 0:
            errors.append(f'the authority corpus declares no {outcome} case')
        if observations.get(signal) != tallies[outcome]:
            errors.append(f'authority {signal} disagrees with its cases')
    authority_records = sum(item['expected']['producer']['record_count'] for item in cases if item['acquisition'] == authority['authoritative_outcome'])
    withheld_records = sum(item['expected']['producer']['record_count'] for item in cases if item['acquisition'] != authority['authoritative_outcome'])
    empty_reads = sum(1 for item in cases if item['expected']['producer']['explicitly_empty'])
    if observations.get('authoritative_record_count') != authority_records:
        errors.append('authority authoritative_record_count disagrees with its cases')
    if observations.get('non_authoritative_record_count') != withheld_records:
        errors.append('authority non_authoritative_record_count disagrees with its cases')
    if observations.get('explicitly_empty_cases') != empty_reads:
        errors.append('authority explicitly_empty_cases disagrees with its cases')
    if observations.get('final_outcome') != cases[-1]['acquisition']:
        errors.append('authority final_outcome disagrees with its final case')
    # IP-05-T07 owns commit, so no case may publish a committed lineage or a partial
    # record set; the runtime proof lives in the Go and Node consumers.
    if observations.get('partial_outputs') != 0 or observations.get('committed_queryable_partitions') != 0:
        errors.append('the authority corpus claims a partial output or a queryable partition before IP-05-T07')
    invariants = document['expected']['invariants']
    if set(invariants['host_committed_state']) != AUTHORITY_COMMITTED_FIELDS or invariants['host_committed_state'] != {
        'status': 'Uninitialized', 'epoch': '', 'revision': 0, 'count': 0, 'queryable': False,
    }:
        errors.append('the authority corpus declares a committed state instead of the uninitialized invariant')
    if set(invariants['producer_retained_records']) != AUTHORITY_RETAINED_FIELDS:
        errors.append('the authority corpus declares an undeclared retained-record invariant')
    if invariants['producer_retained_records']['authoritative_case'] != authority_records:
        errors.append('the authoritative retained-record invariant disagrees with the corpus')
    if not str(invariants.get('note', '')).strip():
        errors.append('the authority corpus states no invariant note')

    signals = {item['acquisition_signal'] for item in cases}
    for signal in AUTHORITY_SIGNALS:
        if signal not in signals:
            errors.append(f'the authority corpus declares no {signal} acquisition signal')
    reasons = {item['expected']['producer']['reason'] for item in cases}
    for reason in AUTHORITY_REASONS:
        if reason not in reasons:
            errors.append(f'the authority corpus proves no refusal with reason {reason}')
    over_limit = sum(1 for item in cases if item.get('record_expansion') is not None)
    if over_limit == 0:
        errors.append('the authority corpus proves no size-gate refusal')
    evidence = {
        'over_limit': over_limit,
        'fixture_id': AUTHORITY_FIXTURE_ID,
        'owner_task': AUTHORITY_OWNER_TASK,
        'cases': len(cases),
        'authoritative': tallies[authority['authoritative_outcome']],
        'withheld': len(cases) - tallies[authority['authoritative_outcome']],
        'records': authority_records,
        'empty_reads': empty_reads,
        'reasons': len(reasons - {None}),
    }
    return errors, evidence


def validate() -> tuple[list[str], list[tuple[str, str, str, int, int]], dict]:
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
    authority_document = json.loads((ROOT / AUTHORITY_ARTIFACT).read_text(encoding='utf-8'))
    authority_failures, authority_evidence = authority_errors(authority_document, document)
    errors.extend(authority_failures)
    return errors, evidence, authority_evidence


def main() -> int:
    arguments = sys.argv[1:]
    if arguments[:1] == ['--help']:
        print(f'usage: {pathlib.Path(__file__).name} --suite {SUITE} --fixtures {FIXTURE_DIRECTORY} --strict')
        return 0
    if arguments != ['--suite', SUITE, '--fixtures', FIXTURE_DIRECTORY, '--strict']:
        print(f'FAIL: invalid arguments; run --suite {SUITE} --fixtures {FIXTURE_DIRECTORY} --strict')
        return 2
    errors, evidence, authority_evidence = validate()
    for message in errors:
        print(f'FAIL: {message}')
    if errors:
        return 1
    for fixture_id, outcome, epoch, revision, identities in evidence:
        print(f'PASS: {fixture_id} outcome={outcome} epoch={epoch} revision={revision} identities={identities}')
    print(
        f'PASS: {authority_evidence["fixture_id"]} owner={authority_evidence["owner_task"]}'
        f' cases={authority_evidence["cases"]} authoritative={authority_evidence["authoritative"]}'
        f' withheld={authority_evidence["withheld"]} records={authority_evidence["records"]}'
        f' explicitly_empty={authority_evidence["empty_reads"]} over_limit={authority_evidence["over_limit"]}'
        f' refusal_reasons={authority_evidence["reasons"]}'
        ' partial_outputs=0 committed_queryable=0'
    )
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
