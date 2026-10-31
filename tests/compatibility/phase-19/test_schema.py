from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from schema import CellReportValidationError, validate_cell_report


class CellReportSchemaTests(unittest.TestCase):
    def setUp(self) -> None:
        self.valid_report = {
            "cell_id": "C05",
            "status": "PASS",
            "run_timestamp": "2026-10-31T12:00:00Z",
            "git_commit": "0123456789abcdef0123456789abcdef01234567",
            "extension_package_sha256": "abc1",
            "host_binary_sha256": "def2",
            "host_manifest_sha256": "ghi3",
            "versions": {"protocol": 1, "schema": 1, "ranking": "rank-1"},
            "fixture_sha256": "jkl4",
            "hardware_id": "REF-HW-WINDOWS-X86_64-01",
            "os_build": "Windows 11 Build 26100",
            "browser_build": "Google Chrome 130.0",
            "locale": "en_US",
            "power_mode": "AC / Best Performance",
            "command_line": ["run-cell", "--browser", "chrome", "--os", "windows"],
            "raw_artifact_paths": {"lifecycle": "artifacts/lifecycle.json"},
            "gate_statuses": {"performance": "PASS", "privacy": "PASS"},
            "blocked_reason": None,
        }

    def test_valid_report_passes(self) -> None:
        validate_cell_report(self.valid_report)

    def test_unqualified_or_empty_status_rejected(self) -> None:
        for bad_status in ["", "not run", "PENDING", None, "UNKNOWN"]:
            with self.subTest(status=bad_status):
                report = dict(self.valid_report, status=bad_status)
                with self.assertRaises(CellReportValidationError):
                    validate_cell_report(report)

    def test_blocked_requires_reason(self) -> None:
        report = dict(self.valid_report, status="BLOCKED", blocked_reason=None)
        with self.assertRaises(CellReportValidationError):
            validate_cell_report(report)

        report["blocked_reason"] = "Hardware unavailable"
        validate_cell_report(report)
