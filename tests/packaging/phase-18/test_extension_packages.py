from __future__ import annotations

import json
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))

from packaging.release.archive import ArchiveError, inspect_zip  # noqa: E402
from packaging.release.config import ReleaseConfigError, load_release_manifest  # noqa: E402
from packaging.release.extension_package import (  # noqa: E402
    build_extension_package,
    inspect_extension_package,
    validate_extension_manifest,
)


class ExtensionPackageTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.release = load_release_manifest(ROOT)
        cls.dist = ROOT / "extension" / "dist"
        if not cls.dist.is_dir():
            raise unittest.SkipTest("extension/dist is absent; run npm run build:extension")

    def test_chrome_and_edge_packages_are_exact_and_deterministic(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            temp = Path(directory)
            first = temp / "chrome-a.zip"
            second = temp / "chrome-b.zip"
            for browser, output in (("chrome", first), ("edge", second)):
                build_extension_package(
                    output,
                    self.release,
                    browser,
                    ROOT / "packaging" / "extension" / browser / "manifest.json",
                    self.dist,
                )
                inspected = inspect_extension_package(output, self.release, browser)
                self.assertEqual(inspected["manifest"]["manifest_version"], 3)
                self.assertEqual(set(inspected["manifest"]["permissions"]), set(self.release["permissions"]))
                self.assertNotIn("host_permissions", inspected["manifest"])
                self.assertIn("manifest.json", [info.filename for info in inspect_zip(output)])
            build_extension_package(
                first,
                self.release,
                "chrome",
                ROOT / "packaging" / "extension" / "chrome" / "manifest.json",
                self.dist,
            )
            build_extension_package(
                second,
                self.release,
                "chrome",
                ROOT / "packaging" / "extension" / "chrome" / "manifest.json",
                self.dist,
            )
            self.assertEqual(first.read_bytes(), second.read_bytes())

    def test_permission_drift_is_rejected(self) -> None:
        manifest = json.loads((ROOT / "packaging/extension/chrome/manifest.json").read_text(encoding="utf-8"))
        manifest["permissions"].append("cookies")
        with self.assertRaises(ReleaseConfigError):
            validate_extension_manifest(manifest, self.release)

    def test_host_permissions_are_rejected(self) -> None:
        manifest = json.loads((ROOT / "packaging/extension/chrome/manifest.json").read_text(encoding="utf-8"))
        manifest["host_permissions"] = ["<all_urls>"]
        with self.assertRaises(ReleaseConfigError):
            validate_extension_manifest(manifest, self.release)

    def test_traversal_archive_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "bad.zip"
            with zipfile.ZipFile(path, "w") as archive:
                archive.writestr("../escape.txt", b"no")
            with self.assertRaises(ArchiveError):
                inspect_zip(path)


if __name__ == "__main__":
    unittest.main()
