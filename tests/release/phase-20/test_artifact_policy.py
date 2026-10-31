from __future__ import annotations

import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
FIXTURES = ROOT / "fixtures/release/phase-20"


class ArtifactPolicyTests(unittest.TestCase):
    def test_extension_permissions_are_exact_four(self) -> None:
        manifest_chrome = json.loads((ROOT / "packaging/extension/chrome/manifest.json").read_text(encoding="utf-8"))
        manifest_edge = json.loads((ROOT / "packaging/extension/edge/manifest.json").read_text(encoding="utf-8"))
        expected = ["nativeMessaging", "storage", "tabGroups", "tabs"]
        self.assertEqual(sorted(manifest_chrome["permissions"]), expected)
        self.assertEqual(sorted(manifest_edge["permissions"]), expected)
        self.assertNotIn("host_permissions", manifest_chrome)
        self.assertNotIn("host_permissions", manifest_edge)

    def test_host_manifest_origins_are_literal_and_exact(self) -> None:
        origins = json.loads((FIXTURES / "candidate-origins.json").read_text(encoding="utf-8"))
        for browser, spec in origins["origins"].items():
            self.assertFalse(spec["wildcard_allowed"])
            self.assertTrue(spec["expected_origin"].startswith("chrome-extension://"))
            self.assertTrue(spec["expected_origin"].endswith("/*"))
            self.assertEqual(len(spec["extension_id"]), 32)

    def test_no_signing_secrets_in_repo(self) -> None:
        provenance = json.loads((FIXTURES / "provenance.json").read_text(encoding="utf-8"))
        self.assertFalse(provenance["provenance"]["signing_material_present_in_repo"])
