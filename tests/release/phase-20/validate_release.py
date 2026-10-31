#!/usr/bin/env python3
"""Validate InfoBoard release candidate gates and artifacts for IP-20."""

from __future__ import annotations

import argparse
import datetime
import json
import sys
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))
sys.path.insert(0, str(Path(__file__).parent))

from candidate_checks import validate_candidate


def validate_full_release(
    candidate_path: Path,
    requirements_path: Path,
    compatibility_path: Path,
    artifacts_path: Path,
    evidence_dir: Path,
    output_path: Path,
) -> dict[str, Any]:
    output_path = Path(output_path).resolve()
    output_path.parent.mkdir(parents=True, exist_ok=True)

    candidate = json.loads(candidate_path.read_text(encoding="utf-8"))
    requirements = json.loads(requirements_path.read_text(encoding="utf-8"))
    compatibility = json.loads(compatibility_path.read_text(encoding="utf-8"))
    artifacts = json.loads(artifacts_path.read_text(encoding="utf-8"))

    # 1. Candidate check
    validate_candidate(candidate, compatibility)

    # 2. Requirements coverage check
    req_ids = {r["req_id"] for r in requirements.get("requirements", [])}
    for i in range(1, 16):
        if f"FR-{i:03d}" not in req_ids:
            raise ValueError(f"Missing FR-{i:03d}")
    for i in range(1, 11):
        if f"NFR-{i:03d}" not in req_ids:
            raise ValueError(f"Missing NFR-{i:03d}")

    # 3. Artifact manifest check
    art_list = artifacts.get("artifacts", [])
    if len(art_list) < 20:
        raise ValueError(f"Expected at least 20 artifacts, found {len(art_list)}")

    # 4. Gate ledger check
    gate_ledger_path = evidence_dir / "gate-ledger.json"
    if not gate_ledger_path.is_file():
        raise FileNotFoundError(f"Missing gate-ledger.json in {evidence_dir}")
    gate_ledger = json.loads(gate_ledger_path.read_text(encoding="utf-8"))
    if gate_ledger.get("overall_decision") != "APPROVE_FOR_RELEASE":
        raise ValueError(f"Gate ledger decision is not approved: {gate_ledger.get('overall_decision')}")

    # 5. Signoff check
    signoff_path = evidence_dir / "reviewer-signoff.json"
    if not signoff_path.is_file():
        raise FileNotFoundError(f"Missing reviewer-signoff.json in {evidence_dir}")
    signoff = json.loads(signoff_path.read_text(encoding="utf-8"))
    if signoff.get("final_release_decision") != "APPROVE":
        raise ValueError("Signoff decision is not approved")

    report = {
        "status": "PASS",
        "validated_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
        "candidate_id": candidate["candidate_id"],
        "release_version": candidate["release_version"],
        "release_commit": candidate["release_commit"],
        "checks": {
            "candidate_identity": "PASS",
            "requirements_coverage": "PASS",
            "compatibility_policy": "PASS",
            "artifact_manifest": "PASS",
            "gate_ledger": "PASS",
            "reviewer_signoff": "PASS",
        },
        "overall_result": "VALIDATED_AND_APPROVED_FOR_RELEASE",
    }
    output_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    return report


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Validate InfoBoard release candidate gates and artifacts")
    parser.add_argument("--candidate", required=True, type=Path)
    parser.add_argument("--requirements", required=True, type=Path)
    parser.add_argument("--compatibility", required=True, type=Path)
    parser.add_argument("--artifacts", required=True, type=Path)
    parser.add_argument("--evidence", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args(argv)

    try:
        report = validate_full_release(
            candidate_path=args.candidate,
            requirements_path=args.requirements,
            compatibility_path=args.compatibility,
            artifacts_path=args.artifacts,
            evidence_dir=args.evidence,
            output_path=args.output,
        )
        print(json.dumps({"status": report["status"], "overall_result": report["overall_result"]}, indent=2))
        return 0
    except Exception as err:
        print(f"RELEASE VALIDATION FAILED: {err}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
