from __future__ import annotations

import json
import os
import sqlite3
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))

from packaging.release.config import load_release_manifest  # noqa: E402
from packaging.release.ownership import (  # noqa: E402
    OwnershipError,
    OwnedEntry,
    load_owned_entries,
    remove_entries,
    write_ownership_marker,
)
from packaging.release.paths import resolve_paths  # noqa: E402
from packaging.release.preflight import preflight  # noqa: E402
from packaging.release.registry import JsonRegistry  # noqa: E402


class PreflightTests(unittest.TestCase):
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

    def paths(self, name: str, platform: str = "linux"):
        root = self.base / name
        return resolve_paths(platform, root_override=root)

    def test_clean_preflight_passes(self) -> None:
        paths = self.paths("clean")
        result = preflight(
            self.release,
            self.artifacts,
            paths,
            "amd64",
            ("chrome", "edge"),
            registry=JsonRegistry(self.base / "clean-registry.json"),
        )
        self.assertTrue(result.ok, result.errors)

    def test_unknown_ownership_is_rejected(self) -> None:
        paths = self.paths("unknown")
        paths.data_root.mkdir(parents=True)
        (paths.data_root / "user-file.txt").write_text("not owned", encoding="utf-8")
        result = preflight(self.release, self.artifacts, paths, "amd64", ("chrome",))
        self.assertFalse(result.ok)
        self.assertTrue(any("ownership" in error for error in result.errors))

    def test_registration_conflict_is_rejected(self) -> None:
        paths = self.paths("registration", platform="windows")
        registry = JsonRegistry(self.base / "registration-registry.json")
        registry.set(paths.registry_keys["chrome"], r"C:\elsewhere\com.infoboard.host.json")
        result = preflight(self.release, self.artifacts, paths, "amd64", ("chrome",), registry=registry)
        self.assertFalse(result.ok)
        self.assertTrue(any("registration" in error for error in result.errors))

    def test_future_storage_schema_is_rejected(self) -> None:
        paths = self.paths("future-schema")
        paths.config_root.mkdir(parents=True)
        database = paths.config_root / "infoboard.sqlite"
        connection = sqlite3.connect(database)
        connection.execute("PRAGMA user_version = 99")
        connection.commit()
        connection.close()
        result = preflight(self.release, self.artifacts, paths, "amd64", ("chrome",))
        self.assertFalse(result.ok)
        self.assertTrue(any("schema" in error for error in result.errors))

    def test_publication_requires_approved_store_ids(self) -> None:
        paths = self.paths("publication")
        result = preflight(self.release, self.artifacts, paths, "amd64", ("chrome",), require_publication=True)
        self.assertFalse(result.ok)
        self.assertTrue(any("publication" in error for error in result.errors))

    def test_ownership_manifest_rejects_traversal(self) -> None:
        root = self.base / "manifest"
        manifest = root / "packaging/lifecycle/owned-paths.json"
        manifest.parent.mkdir(parents=True)
        manifest.write_text(
            json.dumps(
                {
                    "version": 1,
                    "entries": [{"id": "escape", "relative_path": "../escape", "kind": "file"}],
                }
            ),
            encoding="utf-8",
        )
        with self.assertRaises(OwnershipError):
            load_owned_entries(root)

    def test_safe_remove_preserves_unknown_files(self) -> None:
        root = self.base / "remove"
        root.mkdir(parents=True)
        (root / "owned.txt").write_text("owned", encoding="utf-8")
        (root / "unknown.txt").write_text("unknown", encoding="utf-8")
        result = remove_entries(root, [OwnedEntry("owned", "owned.txt", "file")])
        self.assertTrue(result.verified)
        self.assertFalse((root / "owned.txt").exists())
        self.assertTrue((root / "unknown.txt").exists())

    def test_symlink_is_rejected_when_supported(self) -> None:
        root = self.base / "symlink"
        root.mkdir(parents=True)
        target = root / "target.txt"
        target.write_text("target", encoding="utf-8")
        link = root / "link.txt"
        try:
            os.symlink(target, link)
        except (OSError, NotImplementedError):
            self.skipTest("symlink creation is not permitted")
        with self.assertRaises(OwnershipError):
            remove_entries(root, [OwnedEntry("link", "link.txt", "file")])

    def test_ownership_marker_is_required_for_existing_root(self) -> None:
        paths = self.paths("marker")
        paths.data_root.mkdir(parents=True)
        write_ownership_marker(paths.data_root, "InfoBoard", "0.1.0")
        result = preflight(self.release, self.artifacts, paths, "amd64", ("chrome",))
        self.assertTrue(result.ok, result.errors)


if __name__ == "__main__":
    unittest.main()
