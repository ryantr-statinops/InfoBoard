from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(Path(__file__).parent))

from candidate_checks import CandidateValidationError, validate_candidate

FIXTURES = ROOT / "fixtures/release/phase-20"


class CandidateChecksTests(unittest.TestCase):
    def setUp(self) -> None:
        self.candidate = json.loads((FIXTURES / "release-candidate.json").read_text(encoding="utf-8"))
        self.policy = json.loads((FIXTURES / "compatibility-policy.json").read_text(encoding="utf-8"))

    def test_valid_candidate_passes(self) -> None:
        validate_candidate(self.candidate, self.policy)

    def test_version_mismatch_rejected(self) -> None:
        bad = dict(self.candidate, host_version="0.2.0")
        with self.assertRaises(CandidateValidationError):
            validate_candidate(bad, self.policy)

    def test_host_name_mismatch_rejected(self) -> None:
        bad = dict(self.candidate, host_name="com.other.host")
        with self.assertRaises(CandidateValidationError):
            validate_candidate(bad, self.policy)
