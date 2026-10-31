from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))

from packaging.release.config import (  # noqa: E402
    ReleaseConfigError,
    load_release_manifest,
    validate_release_manifest,
)

FIXTURES = ROOT / "fixtures" / "packaging" / "phase-18"


class ReleaseManifestTests(unittest.TestCase):
    def test_checked_in_release_manifest_is_valid_and_fixture_blocked(self) -> None:
        manifest = load_release_manifest(ROOT)
        self.assertEqual(manifest["host_name"], "com.infoboard.host")
        self.assertEqual(manifest["publication_status"], "blocked_pending_store_ids")
        self.assertEqual(manifest["release_channel"], "fixture")
        self.assertEqual(len(manifest["targets"]), 6)

    def test_valid_fixture_matches_checked_in_manifest(self) -> None:
        fixture = json.loads((FIXTURES / "release-manifest.valid.json").read_text(encoding="utf-8"))
        validate_release_manifest(fixture)
        self.assertEqual(fixture["extension_ids"]["chrome"], "abcdefghijklmnopabcdefghijklmnop")

    def test_invalid_fixtures_fail_closed(self) -> None:
        cases = {
            "release-manifest.missing-id.json": "extension_ids",
            "release-manifest.unresolved-id.json": "placeholder",
            "release-manifest.duplicate-version.json": "Chrome package version",
            "release-manifest.unsupported-target.json": "unsupported target OS",
            "release-manifest.permission-drift.json": "permissions",
        }
        for filename, expected in cases.items():
            with self.subTest(filename=filename):
                fixture = json.loads((FIXTURES / filename).read_text(encoding="utf-8"))
                with self.assertRaises(ReleaseConfigError) as raised:
                    validate_release_manifest(fixture)
                self.assertIn(expected, str(raised.exception))

    def test_approved_publication_cannot_use_fixture_ids(self) -> None:
        fixture = json.loads((FIXTURES / "release-manifest.valid.json").read_text(encoding="utf-8"))
        fixture["release_channel"] = "store"
        fixture["publication_status"] = "approved"
        with self.assertRaises(ReleaseConfigError) as raised:
            validate_release_manifest(fixture)
        self.assertIn("fixture extension ID", str(raised.exception))

    def test_missing_extension_id_mapping_disagreement_is_rejected(self) -> None:
        fixture = json.loads((FIXTURES / "release-manifest.valid.json").read_text(encoding="utf-8"))
        extension_ids = json.loads((ROOT / "packaging/config/extension-ids.json").read_text(encoding="utf-8"))
        extension_ids["chrome"]["extension_id"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        with self.assertRaises(ReleaseConfigError) as raised:
            validate_release_manifest(fixture, extension_ids)
        self.assertIn("disagrees", str(raised.exception))


if __name__ == "__main__":
    unittest.main()
