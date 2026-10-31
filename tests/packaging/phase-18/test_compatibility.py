from __future__ import annotations

import copy
import json
import sqlite3
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))

from packaging.release.compatibility import release_tuple_errors  # noqa: E402
from packaging.release.config import load_release_manifest  # noqa: E402
from packaging.release.extension_package import inspect_extension_package  # noqa: E402
from packaging.release.paths import resolve_paths  # noqa: E402
from packaging.release.preflight import preflight  # noqa: E402
from packaging.release.registry import JsonRegistry  # noqa: E402


class CompatibilityTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.temp = tempfile.TemporaryDirectory()
        cls.base = Path(cls.temp.name)
        cls.artifacts = cls.base / "artifacts"
        completed = subprocess.run(
            [
                sys.executable,
                "packaging/release/build.py",
                "--config",
                "packaging/config/release-manifest.json",
                "--output",
                str(cls.artifacts),
                "--skip-extension-build",
            ],
            cwd=str(ROOT),
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        if completed.returncode != 0:
            raise RuntimeError(f"build failed:\n{completed.stdout}\n{completed.stderr}")
        cls.release = load_release_manifest(ROOT)

    @classmethod
    def tearDownClass(cls) -> None:
        cls.temp.cleanup()

    def test_release_tuple_is_supported(self) -> None:
        self.assertEqual(release_tuple_errors(self.release), [])

    def test_protocol_ranking_storage_and_versions_are_checked(self) -> None:
        cases = [
            ("protocol", {"major": 2}),
            ("ranking_model_version", "rank-2"),
            ("storage_schema_version", 2),
        ]
        for key, value in cases:
            with self.subTest(key=key):
                release = copy.deepcopy(self.release)
                release[key] = value
                self.assertTrue(release_tuple_errors(release))
        for key in ("product", "host"):
            with self.subTest(version=key):
                release = copy.deepcopy(self.release)
                release["versions"][key] = "9.9.9"
                self.assertTrue(release_tuple_errors(release))
        release = copy.deepcopy(self.release)
        release["versions"]["chrome"] = "9.9.9"
        self.assertTrue(release_tuple_errors(release))

    def test_preflight_blocks_unsupported_release_tuple(self) -> None:
        release = copy.deepcopy(self.release)
        release["protocol"]["major"] = 2
        paths = resolve_paths("windows", root_override=self.base / "unsupported")
        result = preflight(
            release,
            self.artifacts,
            paths,
            "amd64",
            ("chrome",),
            registry=JsonRegistry(self.base / "unsupported-registry.json"),
        )
        self.assertFalse(result.ok)
        self.assertFalse(result.checks["release_compatibility"])

    def test_future_storage_schema_is_blocked_and_older_is_readable(self) -> None:
        for version, expected in ((999, False), (0, True)):
            with self.subTest(version=version):
                root = self.base / f"schema-{version}"
                paths = resolve_paths("windows", root_override=root)
                paths.config_root.mkdir(parents=True)
                database = paths.config_root / "infoboard.sqlite"
                connection = sqlite3.connect(database)
                try:
                    connection.execute(f"PRAGMA user_version = {version}")
                    connection.commit()
                finally:
                    connection.close()
                result = preflight(
                    self.release,
                    self.artifacts,
                    paths,
                    "amd64",
                    ("chrome",),
                    registry=JsonRegistry(self.base / f"schema-{version}-registry.json"),
                )
                self.assertEqual(result.ok, expected, result.errors)
                self.assertEqual(result.checks["storage_schema"], version)

    def test_extension_and_host_manifests_do_not_add_privacy_surfaces(self) -> None:
        for browser in ("chrome", "edge"):
            package = self.artifacts / self.release["artifacts"][f"{browser}_extension"]
            result = inspect_extension_package(package, self.release, browser)
            manifest = result["manifest"]
            self.assertNotIn("host_permissions", manifest)
            self.assertNotIn("content_scripts", manifest)
            self.assertNotIn("externally_connectable", manifest)
            self.assertNotIn("web_accessible_resources", manifest)
            self.assertEqual(manifest["permissions"], ["storage", "tabs", "tabGroups", "nativeMessaging"])
            host = json.loads(
                (
                    self.artifacts
                    / "host-manifests/windows/amd64"
                    / browser
                    / "com.infoboard.host.json"
                ).read_text(encoding="utf-8")
            )
            self.assertEqual(set(host), {"name", "description", "path", "type", "allowed_origins"})
            self.assertEqual(len(host["allowed_origins"]), 1)

    def test_packaging_code_never_calls_tab_mutation_apis(self) -> None:
        forbidden = ("chrome.tabs", "browser.tabs", "tabs.query", "tabs.update", "tabs.remove")
        for directory in (ROOT / "packaging/release", ROOT / "packaging/installers"):
            for path in directory.rglob("*.py"):
                text = path.read_text(encoding="utf-8")
                for needle in forbidden:
                    with self.subTest(path=path, needle=needle):
                        self.assertNotIn(needle, text)


if __name__ == "__main__":
    unittest.main()
