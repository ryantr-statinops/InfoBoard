from __future__ import annotations

import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SIGNOFF_PATH = ROOT / "artifacts/release/phase-20/reviewer-signoff.json"


class SignoffTests(unittest.TestCase):
    def test_all_five_independent_roles_signed_off(self) -> None:
        self.assertTrue(SIGNOFF_PATH.is_file())
        data = json.loads(SIGNOFF_PATH.read_text(encoding="utf-8"))
        self.assertEqual(data["final_release_decision"], "APPROVE")
        self.assertEqual(data["unresolved_in_scope_defects_count"], 0)

        roles = [s["role"] for s in data["signoffs"]]
        expected_roles = [
            "Release / Change-Review",
            "Implementation / Code-Review",
            "Testing / Verification",
            "Security / Privacy",
            "Product / Operations",
        ]
        self.assertEqual(roles, expected_roles)
        for s in data["signoffs"]:
            self.assertEqual(s["disposition"], "APPROVE")
