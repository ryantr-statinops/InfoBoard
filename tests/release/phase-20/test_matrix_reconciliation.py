from __future__ import annotations

import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
FIXTURES = ROOT / "fixtures/release/phase-20"
MATRIX_REPORT = ROOT / "artifacts/verification/phase-19/matrix-report.json"
TRACEABILITY = ROOT / "artifacts/release/phase-20/traceability.json"
GATE_LEDGER = ROOT / "artifacts/release/phase-20/gate-ledger.json"


class MatrixReconciliationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.candidate = json.loads((FIXTURES / "release-candidate.json").read_text(encoding="utf-8"))
        self.matrix = json.loads(MATRIX_REPORT.read_text(encoding="utf-8"))
        self.traceability = json.loads(TRACEABILITY.read_text(encoding="utf-8"))
        self.gates = json.loads(GATE_LEDGER.read_text(encoding="utf-8"))

    def test_matrix_cells_match_candidate_supported_matrix(self) -> None:
        candidate_cells = [c["cell_id"] for c in self.candidate["supported_matrix"]]
        matrix_cells = [c["cell_id"] for c in self.matrix["cells"]]
        self.assertEqual(candidate_cells, matrix_cells)

    def test_traceability_has_all_25_fr_nfr_requirements(self) -> None:
        ids = {entry["id"] for entry in self.traceability["entries"]}
        for i in range(1, 16):
            self.assertIn(f"FR-{i:03d}", ids)
        for i in range(1, 11):
            self.assertIn(f"NFR-{i:03d}", ids)

    def test_gate_ledger_decision_is_approved(self) -> None:
        self.assertEqual(self.gates["overall_decision"], "APPROVE_FOR_RELEASE")
        for gate_name, gate in self.gates["gates"].items():
            self.assertEqual(gate["status"], "PASS", f"Gate {gate_name} must be PASS")
