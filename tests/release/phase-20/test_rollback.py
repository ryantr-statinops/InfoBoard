from __future__ import annotations

import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
ROLLBACK_JSON = ROOT / "artifacts/release/phase-20/rollback.json"
ROLLBACK_EVIDENCE = ROOT / "artifacts/release/phase-20/rollback-evidence.json"


class RollbackTests(unittest.TestCase):
    def test_rollback_drill_passed_and_evidence_generated(self) -> None:
        self.assertTrue(ROLLBACK_JSON.is_file())
        self.assertTrue(ROLLBACK_EVIDENCE.is_file())
        data = json.loads(ROLLBACK_JSON.read_text(encoding="utf-8"))
        self.assertEqual(data["rehearsal_result"], "PASS")
        self.assertTrue(data["incompatible_downgrade_rejected_fail_closed"])
        self.assertTrue(data["unrelated_browser_tabs_preserved"])
        self.assertTrue(data["profile_isolation_intact"])
        self.assertTrue(len(data["drill_steps"]) >= 4)
