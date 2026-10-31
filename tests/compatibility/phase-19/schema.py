"""Cell report schema and validator for IP-19 matrix verification."""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Any

VALID_CELL_IDS = {"C01", "C02", "C03", "C04", "C05", "C06"}
VALID_STATUSES = {"PASS", "FAIL", "BLOCKED"}


class CellReportValidationError(ValueError):
    """Raised when a cell report fails schema validation."""


def validate_cell_report(report: dict[str, Any]) -> None:
    if not isinstance(report, dict):
        raise CellReportValidationError("Cell report must be a JSON object")

    cell_id = report.get("cell_id")
    if cell_id not in VALID_CELL_IDS:
        raise CellReportValidationError(f"Invalid or missing cell_id: {cell_id}. Must be one of {sorted(VALID_CELL_IDS)}")

    status = report.get("status")
    if status not in VALID_STATUSES:
        raise CellReportValidationError(
            f"Invalid or missing status: {status!r}. Status cannot be empty or unqualified; must be PASS, FAIL, or BLOCKED"
        )

    if status == "BLOCKED":
        blocked_reason = report.get("blocked_reason")
        if not blocked_reason or not isinstance(blocked_reason, str) or not blocked_reason.strip():
            raise CellReportValidationError("A BLOCKED cell report must supply an explicit non-empty blocked_reason")

    required_keys = [
        "run_timestamp",
        "git_commit",
        "extension_package_sha256",
        "host_binary_sha256",
        "host_manifest_sha256",
        "versions",
        "fixture_sha256",
        "hardware_id",
        "os_build",
        "browser_build",
        "locale",
        "power_mode",
        "command_line",
        "raw_artifact_paths",
        "gate_statuses",
    ]

    for key in required_keys:
        if key not in report:
            raise CellReportValidationError(f"Missing required field: {key}")

    versions = report.get("versions")
    if not isinstance(versions, dict) or not {"protocol", "schema", "ranking"}.issubset(versions.keys()):
        raise CellReportValidationError("versions field must be an object declaring protocol, schema, and ranking")

    gates = report.get("gate_statuses")
    if not isinstance(gates, dict) or not gates:
        raise CellReportValidationError("gate_statuses must be a non-empty object mapping gate names to statuses")
