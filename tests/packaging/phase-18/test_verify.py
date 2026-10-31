from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))

from packaging.release.build import build  # noqa: E402
from packaging.release.verify import verify_artifacts  # noqa: E402

FIXTURES = ROOT / "fixtures" / "packaging" / "phase-18"


class VerifyTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.temp = tempfile.TemporaryDirectory()
        cls.base = Path(cls.temp.name)
        cls.artifacts = cls.base / "artifacts"
        build(
            root=ROOT,
            config_path=ROOT / "packaging/config/release-manifest.json",
            output=cls.artifacts,
            skip_extension_build=True,
            skip_host_build=False,
        )

    @classmethod
    def tearDownClass(cls) -> None:
        cls.temp.cleanup()

    def test_verify_artifacts_passes_with_negative_fixtures(self) -> None:
        report = verify_artifacts(self.artifacts, FIXTURES)
        self.assertEqual(report["status"], "verified")
        self.assertEqual(report["product_version"], "0.1.0")
        self.assertEqual(report["host_manifest_count"], 6)
        self.assertIn("chrome", report["extension_checks"])
        self.assertIn("edge", report["extension_checks"])
        self.assertTrue(len(report["negative_checks"]) >= 8)
        for fixture_name, outcome in report["negative_checks"].items():
            self.assertEqual(outcome, "rejected_correctly")

    def test_verify_cli(self) -> None:
        completed = subprocess.run(
            [
                sys.executable,
                str(ROOT / "packaging/release/verify.py"),
                "--artifacts",
                str(self.artifacts),
                "--fixture-root",
                str(FIXTURES),
            ],
            cwd=str(ROOT),
            text=True,
            capture_output=True,
            check=False,
        )
        self.assertEqual(completed.returncode, 0, f"stdout: {completed.stdout}\nstderr: {completed.stderr}")
        data = json.loads(completed.stdout)
        self.assertEqual(data["status"], "verified")
