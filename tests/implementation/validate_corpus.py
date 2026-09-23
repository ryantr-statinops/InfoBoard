#!/usr/bin/env python3
"""Validate InfoBoard implementation-plan and fixture-catalog contracts."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
ALLOWED_STATUS = {
    "Not started",
    "In progress",
    "Blocked",
    "Ready for dev integration",
    "Integrated in dev",
    "Complete",
}
EXPECTED_FIXTURES = {
    "FX-SHORTCUT-FOCUS",
    "FX-QUERY-LIVE",
    "FX-RANKING-DETERMINISTIC",
    "FX-TAB-EVENTS",
    "FX-RECONNECT",
    "FX-STALE-ACTIVATION",
    "FX-HOST-DOWN",
    "FX-RESET-OWNED-DATA",
    "FX-PERMISSION-DENIED",
}


def task_owner_errors(task_ids: list[str], commit_ids: list[str]) -> list[str]:
    errors: list[str] = []
    if len(task_ids) != len(set(task_ids)):
        errors.append("duplicate task ID")
    if len(commit_ids) != len(set(commit_ids)):
        errors.append("duplicate commit owner")
    for task_id in task_ids:
        count = commit_ids.count(task_id)
        if count != 1:
            errors.append(f"{task_id} has {count} commit owners")
    for commit_id in commit_ids:
        if commit_id not in task_ids:
            errors.append(f"commit owns unknown task {commit_id}")
    return errors


def tracker_errors(rows: list[tuple[str, str]]) -> list[str]:
    errors: list[str] = []
    ids = [phase_id for phase_id, _ in rows]
    if len(ids) != 20 or len(ids) != len(set(ids)):
        errors.append("tracker must contain 20 unique phase rows")
    if set(ids) != {f"IP-{n:02d}" for n in range(1, 21)}:
        errors.append("tracker phase IDs must be IP-01 through IP-20")
    for phase_id, status in rows:
        if status not in ALLOWED_STATUS:
            errors.append(f"{phase_id} has invalid status {status!r}")
    return errors


def _json_schema_errors(schema: dict, value: object, root_schema: dict, path: str = "$") -> list[str]:
    errors: list[str] = []
    if "$ref" in schema:
        target = root_schema
        for part in schema["$ref"].split("/")[1:]:
            part = part.replace("~1", "/").replace("~0", "~")
            target = target[part]
        return _json_schema_errors(target, value, root_schema, path)

    allowed_types = schema.get("type")
    if allowed_types is not None:
        if not isinstance(allowed_types, list):
            allowed_types = [allowed_types]
        def matches_type(kind: str) -> bool:
            if kind == "object":
                return isinstance(value, dict)
            if kind == "array":
                return isinstance(value, list)
            if kind == "string":
                return isinstance(value, str)
            if kind == "integer":
                return isinstance(value, int) and not isinstance(value, bool)
            if kind == "number":
                return isinstance(value, (int, float)) and not isinstance(value, bool)
            if kind == "boolean":
                return isinstance(value, bool)
            if kind == "null":
                return value is None
            return True
        if not any(matches_type(kind) for kind in allowed_types):
            return [f"{path}: expected type {allowed_types}"]

    if "const" in schema and value != schema["const"]:
        errors.append(f"{path}: does not match const")
    if "enum" in schema and value not in schema["enum"]:
        errors.append(f"{path}: value is outside enum")
    if isinstance(value, str):
        if len(value) < schema.get("minLength", 0):
            errors.append(f"{path}: shorter than minLength")
        if "pattern" in schema and re.search(schema["pattern"], value) is None:
            errors.append(f"{path}: does not match pattern")
    if isinstance(value, list):
        if len(value) < schema.get("minItems", 0):
            errors.append(f"{path}: fewer items than minItems")
        if schema.get("uniqueItems") and len({json.dumps(item, sort_keys=True) for item in value}) != len(value):
            errors.append(f"{path}: items are not unique")
        item_schema = schema.get("items")
        if isinstance(item_schema, dict):
            for index, item in enumerate(value):
                errors.extend(_json_schema_errors(item_schema, item, root_schema, f"{path}[{index}]"))
    if isinstance(value, dict):
        for key in schema.get("required", []):
            if key not in value:
                errors.append(f"{path}: missing required property {key}")
        properties = schema.get("properties", {})
        for key, item in value.items():
            if key in properties:
                errors.extend(_json_schema_errors(properties[key], item, root_schema, f"{path}.{key}"))
            elif schema.get("additionalProperties") is False:
                errors.append(f"{path}: additional property {key}")
    return errors


def _read(path: Path, errors: list[str]) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except OSError as exc:
        errors.append(f"cannot read {path.relative_to(ROOT)}: {exc}")
        return ""


def commit_entry_errors(entry: str) -> list[str]:
    errors: list[str] = []
    message = re.match(r"\d+\. `([^`]+)`", entry)
    if not message or not re.match(r"(?:feat|fix|docs|test|perf|refactor)\([^)]+\): .+", message.group(1)):
        errors.append("non-conventional commit message")
    for field in ("Owned target paths:", "Behavior:", "Fixture and command:", "Observable result before commit:", "Dependency gate:"):
        if field not in entry:
            errors.append(f"commit entry misses {field}")
    if not re.search(r"- Fixture and command:.*run `[^`]+`", entry):
        errors.append("commit entry has no exact verification command")
    return errors


def _phase_dependencies(index_rows: dict[str, str], phase_id: str) -> set[str]:
    field = index_rows.get(phase_id, "")
    result: set[str] = set()
    for first, last in re.findall(r"IP-(\d{2})(?:\.\.IP-(\d{2}))?", field):
        start = int(first)
        end = int(last) if last else start
        result.update(f"IP-{number:02d}" for number in range(start, end + 1))
    return result


def _validate_catalog(root: Path, errors: list[str]) -> tuple[int, int, int]:
    schema_path = root / "fixtures/catalog.schema.json"
    catalog_path = root / "fixtures/catalog.json"
    try:
        schema = json.loads(schema_path.read_text(encoding="utf-8"))
        catalog = json.loads(catalog_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        errors.append(f"fixture schema/catalog is unreadable or invalid JSON: {exc}")
        return 0, 0, 0

    if schema.get("$schema") != "https://json-schema.org/draft/2020-12/schema":
        errors.append("fixture schema must declare JSON Schema 2020-12")
    errors.extend(f"catalog schema: {message}" for message in _json_schema_errors(schema, catalog, schema))
    if "$defs" not in schema or "fixture" not in schema["$defs"]:
        errors.append("fixture schema is missing the fixture definition")
    else:
        fixture_schema = schema["$defs"]["fixture"]
        required_fixture_fields = {"fixture_id", "owner_phase", "requirement_ids", "category", "input", "expected"}
        if not required_fixture_fields.issubset(set(fixture_schema.get("required", []))):
            errors.append("fixture schema does not require every contract field")
        expected_schema = fixture_schema.get("properties", {}).get("expected", {})
        if not {"observations", "privacy_assertions"}.issubset(set(expected_schema.get("required", []))):
            errors.append("fixture schema does not require observable and privacy outcomes")
    if catalog.get("schema_version") != 1:
        errors.append("fixture catalog schema_version must be 1")

    fixtures = catalog.get("fixtures")
    if not isinstance(fixtures, list):
        errors.append("fixture catalog fixtures must be an array")
        fixtures = []
    fixture_ids = [item.get("fixture_id") for item in fixtures if isinstance(item, dict)]
    if set(fixture_ids) != EXPECTED_FIXTURES or len(fixture_ids) != len(EXPECTED_FIXTURES):
        errors.append("fixture catalog must contain exactly the nine stable IP-01 fixture IDs")

    index = _read(root / "docs/plan/implementation/index.md", errors)
    owner_rows = re.findall(
        r"^\| ((?:FR|NFR)-\d{3}) \| (IP-\d{2}) \|$",
        index[index.find("## Requirement ownership") : index.find("## Phase authoring contract")],
        re.MULTILINE,
    )
    owners: dict[str, str] = {}
    for requirement_id, owner in owner_rows:
        if requirement_id in owners:
            errors.append(f"duplicate primary owner row for {requirement_id}")
        owners[requirement_id] = owner

    requirements = _read(root / "docs/plan/refactor/requirements.md", errors)
    expected_requirements = set(re.findall(r"\b(?:FR|NFR)-\d{3}\b", requirements))
    if len(owners) != 25 or set(owners) != expected_requirements:
        errors.append("primary-owner matrix must cover each canonical FR/NFR ID exactly once")

    for fixture in fixtures:
        if not isinstance(fixture, dict):
            errors.append("fixture entries must be objects")
            continue
        fixture_id = fixture.get("fixture_id", "<missing>")
        required = {"fixture_id", "owner_phase", "requirement_ids", "category", "input", "expected"}
        if not required.issubset(fixture):
            errors.append(f"{fixture_id} is missing required fixture fields")
            continue
        requirement_ids = fixture["requirement_ids"]
        if not isinstance(requirement_ids, list) or not requirement_ids:
            errors.append(f"{fixture_id} must reference at least one requirement")
            continue
        if any(requirement_id not in owners for requirement_id in requirement_ids):
            errors.append(f"{fixture_id} references an unowned requirement")
        elif owners.get(requirement_ids[0]) != fixture["owner_phase"]:
            errors.append(f"{fixture_id} primary fixture owner disagrees with its first requirement")
        expected = fixture["expected"]
        if not isinstance(fixture.get("input"), dict):
            errors.append(f"{fixture_id} input must be an object")
        if not isinstance(expected, dict) or not isinstance(expected.get("observations"), list) or not expected.get("observations"):
            errors.append(f"{fixture_id} must declare observable expected results")
        elif any(not isinstance(item, dict) or not item.get("signal") or "value" not in item for item in expected["observations"]):
            errors.append(f"{fixture_id} observations must have a signal and expected value")
        if not isinstance(expected, dict) or not isinstance(expected.get("privacy_assertions"), list) or not expected["privacy_assertions"]:
            errors.append(f"{fixture_id} must declare privacy assertions")

    invariants = catalog.get("global_invariants", [])
    invariant_ids = [item.get("id") for item in invariants if isinstance(item, dict)]
    if len(invariant_ids) != 7 or len(set(invariant_ids)) != 7:
        errors.append("catalog must contain seven uniquely identified global invariants")
    check_contracts = catalog.get("future_check_contracts", [])
    check_ids = [item.get("check_id") for item in check_contracts if isinstance(item, dict)]
    if len(check_ids) != 8 or len(set(check_ids)) != 8:
        errors.append("catalog must contain eight uniquely identified future check contracts")
    for item in check_contracts:
        if not isinstance(item, dict) or not all(item.get(key) for key in ("check_id", "scope", "selection_rule", "pass_signal")):
            errors.append("future check contracts require scope, selection rule, and pass signal")
    for item in invariants:
        if not isinstance(item, dict) or not all(item.get(key) for key in ("id", "statement", "observable_guard")):
            errors.append("global invariants require a statement and observable guard")

    seams = catalog.get("phase_seams", [])
    seam_phases = [item.get("phase") for item in seams if isinstance(item, dict)]
    expected_phases = {f"IP-{number:02d}" for number in range(2, 21)}
    if set(seam_phases) != expected_phases or len(seam_phases) != len(expected_phases):
        errors.append("phase seams must cover IP-02 through IP-20 exactly once")
    by_fixture = {item.get("fixture_id"): item for item in fixtures if isinstance(item, dict)}
    for seam in seams:
        if not isinstance(seam, dict):
            errors.append("phase seams must be objects")
            continue
        fixture_ids = seam.get("fixture_ids", [])
        signal = seam.get("fixture_signal")
        if not fixture_ids or fixture_ids[0] not in by_fixture:
            errors.append(f"{seam.get('phase', '<unknown>')} seam references an unknown fixture")
            continue
        observations = by_fixture[fixture_ids[0]].get("expected", {}).get("observations", [])
        if not any(item.get("signal") == signal for item in observations if isinstance(item, dict)):
            errors.append(f"{seam['phase']} seam references an unobserved signal")
        requirement_ids = seam.get("requirement_ids", [])
        if not requirement_ids or requirement_ids[0] not in owners:
            errors.append(f"{seam['phase']} seam references an unowned requirement")
        elif seam.get("ownership_relation") == "primary" and owners[requirement_ids[0]] != seam["phase"]:
            errors.append(f"{seam['phase']} incorrectly claims primary requirement ownership")

    return len(fixtures), len(invariants), len(seams)


def validate(root: Path) -> tuple[list[str], dict[str, int]]:
    errors: list[str] = []
    implementation = root / "docs/plan/implementation"
    index_path = implementation / "index.md"
    index = _read(index_path, errors)
    target_layout_start = index.find("## Logical target layout")
    target_layout_end = index.find("## Phase inventory and dependency DAG")
    target_layout = index[target_layout_start:target_layout_end] if target_layout_start >= 0 and target_layout_end > target_layout_start else ""
    for observed in ("fixtures/catalog.schema.json", "fixtures/catalog.json", "tests/implementation/validate_corpus.py"):
        if not (root / observed).exists() or observed not in target_layout or "to-create" in next((line for line in target_layout.splitlines() if observed in line), ""):
            errors.append(f"observed target path is missing or misclassified: {observed}")
    for absent in ("extension/", "host/", "packaging/"):
        if absent in target_layout and "to-create" not in next((line for line in target_layout.splitlines() if absent in line), ""):
            errors.append(f"absent runtime root is not marked to-create: {absent}")

    phase_paths = sorted(implementation.glob("phase-*.md"))
    if len(phase_paths) != 20:
        errors.append(f"expected 20 phase files, found {len(phase_paths)}")

    index_rows: dict[str, str] = {}
    for row in index.splitlines():
        cells = [cell.strip() for cell in row.split("|")]
        if len(cells) >= 6 and re.fullmatch(r"IP-\d{2}", cells[1]):
            index_rows[cells[1]] = cells[4]
    ownership_line = next((line for line in index.splitlines() if line.startswith("Operational ownership:")), "")
    operational_owners = {"IP-07", "IP-08", "IP-16", "IP-17", "IP-18"}
    found_owners = set(re.findall(r"IP-\d{2}", ownership_line))
    if found_owners != operational_owners:
        errors.append("operational ownership must name IP-07, IP-08, IP-16, IP-17, and IP-18 exactly")
    for owner in operational_owners:
        if not _phase_dependencies(index_rows, owner):
            errors.append(f"{owner} operational work has no dependency path in the index")
        matches = list(implementation.glob(f"phase-{owner[3:]}-*.md"))
        if len(matches) != 1:
            errors.append(f"{owner} operational owner has no unique phase file")
            continue
        owner_text = _read(matches[0], errors)
        section8_start = owner_text.find("## 8.")
        section9_start = owner_text.find("## 9.")
        section8 = owner_text[section8_start:section9_start] if section8_start >= 0 and section9_start > section8_start else ""
        if not section8 or not re.search(r"(?m)^- \[[ xX]\]", section8) or "`" not in section8:
            errors.append(f"{owner} operational ownership lacks a concrete acceptance evidence/check in Section 8")

    phase_count = 0
    task_count = 0
    phase_links = set(re.findall(r"\]\((phase-\d{2}-[^)]+\.md)\)", _read(implementation / "README.md", errors)))
    for path in phase_paths:
        phase_count += 1
        text = _read(path, errors)
        phase_id_match = re.search(r"^> Plan ID: (IP-\d{2})$", text, re.MULTILINE)
        if not phase_id_match:
            errors.append(f"{path.name} has no Plan ID")
            continue
        phase_id = phase_id_match.group(1)
        headings = [int(value) for value in re.findall(r"^## (\d+)\. ", text, re.MULTILINE)]
        if headings != list(range(1, 11)):
            errors.append(f"{path.name} must contain sections 1–10 exactly once and in order")
        if "> Status: See README.md execution tracker" not in text:
            errors.append(f"{path.name} status metadata must point to the README tracker")
        if path.name not in phase_links:
            errors.append(f"{phase_id} is missing from the README tracker")

        section6_start = text.find("## 6.")
        section7_start = text.find("## 7.")
        section8_start = text.find("## 8.")
        if min(section6_start, section7_start, section8_start) < 0:
            errors.append(f"{path.name} is missing a numbered execution section")
            continue
        work = text[section6_start:section7_start]
        commit_plan = text[section7_start:section8_start]
        task_ids = re.findall(r"- \[[ xX]\] `(IP-\d{2}-T\d{2})`", work)
        commit_ids = re.findall(r"- Task IDs: `(IP-\d{2}-T\d{2})`\.", commit_plan)
        task_count += len(task_ids)
        errors.extend(f"{path.name}: {message}" for message in task_owner_errors(task_ids, commit_ids))

        blocks = re.split(r"(?=^\d+\. `)", commit_plan, flags=re.MULTILINE)
        entries = [block for block in blocks if re.match(r"\d+\. `", block)]
        if len(entries) != len(task_ids):
            errors.append(f"{path.name} has {len(entries)} commit entries for {len(task_ids)} tasks")
        for entry in entries:
            errors.extend(f"{path.name}: {message}" for message in commit_entry_errors(entry))

        expected_dependencies = set(re.findall(r"IP-\d{2}", re.search(r"^> Dependencies: (.*)$", text, re.MULTILINE).group(1))) if re.search(r"^> Dependencies: (.*)$", text, re.MULTILINE) else set()
        actual_dependencies = _phase_dependencies(index_rows, phase_id)
        if expected_dependencies != actual_dependencies:
            errors.append(f"{phase_id} dependencies disagree with the index")

    readme = _read(implementation / "README.md", errors)
    header = next((line for line in readme.splitlines() if line.startswith("| Phase |")), "")
    required_columns = ("Owner / agent", "Work branch", "PR", "Latest commit SHA", "Last check / evidence reference", "Blocker / next action", "Updated (UTC)")
    if not header or any(column not in header for column in required_columns):
        errors.append("README tracker is missing required progress columns")
    rows = []
    for row in readme.splitlines():
        if re.match(r"^\| IP-\d{2} \|", row):
            cells = [cell.strip() for cell in row.split("|")]
            if len(cells) != 12 or any(not cell for cell in cells[1:-1]):
                errors.append("README phase row has missing fields")
            if len(cells) >= 4:
                rows.append((cells[1], cells[3]))
    errors.extend(tracker_errors(rows))

    for path in (root / "README.md", root / "CONTEXT.md", root / "docs/README.md", root / "docs/plan/README.md", *phase_paths, index_path, implementation / "README.md"):
        text = _read(path, errors)
        for target in re.findall(r"\]\(([^)]+)\)", text):
            target = target.split("#", 1)[0]
            if not target or re.match(r"(?:https?:|mailto:)", target):
                continue
            resolved = (path.parent / target).resolve()
            if not resolved.exists():
                errors.append(f"broken local link in {path.relative_to(root)}: {target}")

    fixture_count, invariant_count, seam_count = _validate_catalog(root, errors)
    summary = {
        "phases": phase_count,
        "tasks": task_count,
        "requirements": 25,
        "fixtures": fixture_count,
        "invariants": invariant_count,
        "phase_seams": seam_count,
    }
    return errors, summary


def main() -> int:
    errors, summary = validate(ROOT)
    if errors:
        for error in errors:
            print(f"FAIL: {error}", file=sys.stderr)
        return 1
    print(
        "PASS: "
        + ", ".join(f"{key}={value}" for key, value in summary.items())
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
