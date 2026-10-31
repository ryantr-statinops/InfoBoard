from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))

from packaging.release.config import ReleaseConfigError, load_release_manifest  # noqa: E402
from packaging.release.host_manifest import (  # noqa: E402
    render_host_manifest,
    validate_host_manifest,
)
from packaging.release.paths import canonical_binary_path  # noqa: E402


class HostManifestTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.release = load_release_manifest(ROOT)
        cls.templates = {
            "chrome": ROOT / "packaging/host/chrome/com.infoboard.host.json.tmpl",
            "edge": ROOT / "packaging/host/edge/com.infoboard.host.json.tmpl",
        }

    def test_chrome_and_edge_render_one_exact_origin(self) -> None:
        for browser, operating_system in (("chrome", "linux"), ("chrome", "macos"), ("chrome", "windows"), ("edge", "linux"), ("edge", "macos"), ("edge", "windows")):
            with self.subTest(browser=browser, os=operating_system):
                manifest = render_host_manifest(
                    self.templates[browser],
                    canonical_binary_path(operating_system),
                    self.release["extension_ids"][browser],
                )
                self.assertEqual(manifest["name"], "com.infoboard.host")
                self.assertEqual(manifest["type"], "stdio")
                self.assertEqual(len(manifest["allowed_origins"]), 1)
                self.assertEqual(
                    manifest["allowed_origins"][0],
                    f"chrome-extension://{self.release['extension_ids'][browser]}/*",
                )
                self.assertNotIn("*", manifest["path"])

    def test_extra_origin_is_rejected(self) -> None:
        manifest = render_host_manifest(
            self.templates["chrome"], canonical_binary_path("linux"), self.release["extension_ids"]["chrome"]
        )
        manifest["allowed_origins"].append("chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/*")
        with self.assertRaises(ReleaseConfigError):
            validate_host_manifest(manifest, self.release["extension_ids"]["chrome"], "linux")

    def test_relative_path_is_rejected(self) -> None:
        manifest = render_host_manifest(
            self.templates["chrome"], canonical_binary_path("linux"), self.release["extension_ids"]["chrome"]
        )
        manifest["path"] = "host/infoboard-host"
        with self.assertRaises(ReleaseConfigError):
            validate_host_manifest(manifest, self.release["extension_ids"]["chrome"], "linux")

    def test_unresolved_token_is_rejected(self) -> None:
        manifest = render_host_manifest(
            self.templates["chrome"], canonical_binary_path("linux"), self.release["extension_ids"]["chrome"]
        )
        manifest["path"] = "/tmp/__BINARY_PATH__/infoboard-host"
        with self.assertRaises(ReleaseConfigError):
            validate_host_manifest(manifest, self.release["extension_ids"]["chrome"], "linux")

    def test_wrong_extension_id_is_rejected(self) -> None:
        manifest = render_host_manifest(
            self.templates["chrome"], canonical_binary_path("linux"), self.release["extension_ids"]["chrome"]
        )
        with self.assertRaises(ReleaseConfigError):
            validate_host_manifest(manifest, self.release["extension_ids"]["edge"], "linux")

    def test_windows_path_round_trips_through_json(self) -> None:
        path = canonical_binary_path("windows")
        manifest = render_host_manifest(self.templates["chrome"], path, self.release["extension_ids"]["chrome"])
        self.assertEqual(manifest["path"], path)


if __name__ == "__main__":
    unittest.main()
