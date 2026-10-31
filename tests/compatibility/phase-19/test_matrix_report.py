from __future__ import annotations

import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
REPORT_PATH = ROOT / "artifacts/verification/phase-19/matrix-report.json"
README_PATH = ROOT / "artifacts/verification/phase-19/README.txt"


class MatrixReportTests(unittest.TestCase):
    def test_report_exists_and_contains_all_six_cells(self) -> None:
        self.assertTrue(REPORT_PATH.is_file(), "matrix-report.json must exist")
        data = json.loads(REPORT_PATH.read_text(encoding="utf-8"))
        self.assertEqual(data["schema_version"], 1)
        self.assertEqual(len(data["cells"]), 6)

        cell_ids = [c["cell_id"] for c in data["cells"]]
        self.assertEqual(cell_ids, ["C01", "C02", "C03", "C04", "C05", "C06"])

        for cell in data["cells"]:
            self.assertIn(cell["status"], {"PASS", "BLOCKED", "FAIL"})
            if cell["status"] == "BLOCKED":
                self.assertTrue(bool(cell["blocked_reason"]))
                self.assertIn("unavailable", cell["blocked_reason"])
            elif cell["status"] == "PASS":
                self.assertIsNone(cell["blocked_reason"])

    def test_traceability_and_gates_present(self) -> None:
        data = json.loads(REPORT_PATH.read_text(encoding="utf-8"))
        traceability = data["requirement_traceability"]
        for req in ["NFR-001", "NFR-002", "NFR-003", "NFR-006", "NFR-008", "NFR-009", "NFR-010"]:
            self.assertIn(req, traceability)

        gates = data["release_gate_ledger"]
        self.assertIn("compatibility_matrix", gates)
        self.assertEqual(gates["performance_p95_bounds"], "PASS")

    def test_readme_replay_guide_exists(self) -> None:
        self.assertTrue(README_PATH.is_file(), "README.txt replay guide must exist")
        text = README_PATH.read_text(encoding="utf-8")
        self.assertIn("run-cell", text)
        self.assertIn("measure-hot-path", text)
