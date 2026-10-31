from __future__ import annotations

import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
REPORT_PATH = ROOT / "artifacts/release/phase-20/release-report.json"
CHECKSUMS_PATH = ROOT / "artifacts/release/phase-20/SHA256SUMS"
HANDOFF_DIR = ROOT / "artifacts/release/phase-20/handoff"


class ReleaseReportTests(unittest.TestCase):
    def test_report_approved_and_complete(self) -> None:
        self.assertTrue(REPORT_PATH.is_file())
        self.assertTrue(CHECKSUMS_PATH.is_file())
        self.assertTrue((HANDOFF_DIR / "README.txt").is_file())
        self.assertTrue((HANDOFF_DIR / "known-limitations.md").is_file())

        data = json.loads(REPORT_PATH.read_text(encoding="utf-8"))
        self.assertEqual(data["release_decision"], "APPROVE_FOR_RELEASE")
        self.assertEqual(data["summary"]["total_gates_evaluated"], 13)
        self.assertEqual(data["summary"]["gates_passed"], 13)
        self.assertEqual(data["summary"]["unresolved_in_scope_defects"], 0)
        self.assertEqual(len(data["signoffs"]), 5)

    def test_checksums_has_candidate_entries(self) -> None:
        lines = CHECKSUMS_PATH.read_text(encoding="utf-8").strip().splitlines()
        self.assertTrue(len(lines) >= 20)
        for line in lines:
            parts = line.split()
            self.assertEqual(len(parts), 2)
            self.assertEqual(len(parts[0]), 64)
