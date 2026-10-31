"""Candidate identity and compatibility tuple validation for IP-20."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[3]


class CandidateValidationError(ValueError):
    """Raised when release candidate metadata fails validation."""


def validate_candidate(candidate: dict[str, Any], policy: dict[str, Any] | None = None) -> None:
    required = [
        "candidate_id",
        "release_version",
        "release_commit",
        "extension_version",
        "host_version",
        "protocol",
        "schema_version_sqlite",
        "ranking_model_version",
        "browser_package_ids",
        "host_name",
        "supported_matrix",
    ]
    for key in required:
        if key not in candidate:
            raise CandidateValidationError(f"Missing required candidate field: {key}")

    if candidate["extension_version"] != candidate["host_version"]:
        raise CandidateValidationError(
            f"Extension version ({candidate['extension_version']}) and host version ({candidate['host_version']}) must be the declared compatible pair"
        )

    if candidate["host_name"] != "com.infoboard.host":
        raise CandidateValidationError(f"Unexpected host_name: {candidate['host_name']}")

    if len(candidate["supported_matrix"]) != 6:
        raise CandidateValidationError("supported_matrix must declare all six cells (C01 through C06)")

    if policy:
        policies = policy.get("policies", [])
        if len(policies) != 6:
            raise CandidateValidationError("Compatibility policy must cover all six cells")
