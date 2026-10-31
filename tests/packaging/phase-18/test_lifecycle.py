from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))

from packaging.release.installer_package import inspect_installer_archive  # noqa: E402


class LifecycleTests(unittest.TestCase):
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
        cls.health = cls.artifacts / "host/windows/amd64/infoboard-host.exe"

    @classmethod
    def tearDownClass(cls) -> None:
        cls.temp.cleanup()

    def run_installer(self, operation: str, root: Path, registry: Path, *extra: str) -> tuple[int, dict]:
        completed = subprocess.run(
            [
                sys.executable,
                str(ROOT / "packaging/installers/windows" / f"{operation}.py"),
                "--release-root",
                str(self.artifacts),
                "--root",
                str(root),
                "--registry-file",
                str(registry),
                "--health-binary",
                str(self.health),
                "--no-acl",
                "--no-stop-processes",
                "--browsers",
                "chrome,edge",
                *extra,
            ],
            cwd=str(ROOT),
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        try:
            payload = json.loads(completed.stdout)
        except json.JSONDecodeError:
            self.fail(f"installer did not emit JSON:\n{completed.stdout}\n{completed.stderr}")
        return completed.returncode, payload

    def test_clean_install_repeat_and_uninstall(self) -> None:
        root = self.base / "clean"
        registry = self.base / "clean-registry.json"
        code, installed = self.run_installer("install", root, registry)
        self.assertEqual(code, 0, installed)
        self.assertEqual(installed["status"], "complete")
        self.assertTrue(installed["owned_state_verified"])
        self.assertFalse(installed["health"]["hello_ack"])
        self.assertEqual(installed["health"]["degraded_reason"], "runtime_protocol_unwired")
        state = json.loads((root / "data/install-state.json").read_text(encoding="utf-8"))
        self.assertEqual(state["current_version"], "0.1.0")
        self.assertTrue((root / "data/host/infoboard-host.exe").is_file())
        for browser, extension_id in (
            ("chrome", "abcdefghijklmnopabcdefghijklmnop"),
            ("edge", "ponmlkjihgfedcbaponmlkjihgfedcba"),
        ):
            manifest = json.loads((root / "browsers" / browser / "com.infoboard.host.json").read_text(encoding="utf-8"))
            self.assertEqual(manifest["allowed_origins"], [f"chrome-extension://{extension_id}/*"])
            self.assertEqual(manifest["type"], "stdio")
        code, repeated = self.run_installer("install", root, registry)
        self.assertEqual(code, 0, repeated)
        self.assertEqual(repeated["status"], "no_op")
        code, removed = self.run_installer("uninstall", root, registry)
        self.assertEqual(code, 0, removed)
        self.assertEqual(removed["status"], "complete")
        self.assertTrue(removed["owned_state_verified"])
        self.assertFalse((root / "data/install-state.json").exists())
        self.assertFalse((root / "data/host/infoboard-host.exe").exists())
        self.assertFalse((root / "browsers/chrome/com.infoboard.host.json").exists())
        self.assertFalse((root / "browsers/edge/com.infoboard.host.json").exists())

    def test_uninstall_is_idempotent(self) -> None:
        root = self.base / "idempotent"
        registry = self.base / "idempotent-registry.json"
        code, removed = self.run_installer("uninstall", root, registry)
        self.assertEqual(code, 0, removed)
        self.assertEqual(removed["status"], "complete")
        self.assertTrue(removed["owned_state_verified"])

    def test_repair_restores_missing_host_binary(self) -> None:
        root = self.base / "repair"
        registry = self.base / "repair-registry.json"
        code, installed = self.run_installer("install", root, registry)
        self.assertEqual(code, 0, installed)
        (root / "data/host/infoboard-host.exe").unlink()
        code, repaired = self.run_installer("repair", root, registry)
        self.assertEqual(code, 0, repaired)
        self.assertEqual(repaired["status"], "complete")
        self.assertTrue((root / "data/host/infoboard-host.exe").is_file())

    def test_rollback_without_previous_is_retryable(self) -> None:
        root = self.base / "rollback"
        registry = self.base / "rollback-registry.json"
        code, installed = self.run_installer("install", root, registry)
        self.assertEqual(code, 0, installed)
        code, rolled_back = self.run_installer("rollback", root, registry)
        self.assertEqual(code, 2, rolled_back)
        self.assertEqual(rolled_back["status"], "retryable")
        self.assertEqual(rolled_back["error_class"], "no_previous_release")
        self.assertIn("reinstall", rolled_back["blocked_reason"])

    def test_upgrade_requires_an_installed_release(self) -> None:
        root = self.base / "upgrade"
        registry = self.base / "upgrade-registry.json"
        code, upgraded = self.run_installer("upgrade", root, registry)
        self.assertEqual(code, 3, upgraded)
        self.assertEqual(upgraded["status"], "blocked")
        self.assertEqual(upgraded["error_class"], "not_installed")

    def test_uninstall_preserves_unknown_files(self) -> None:
        root = self.base / "preserve"
        registry = self.base / "preserve-registry.json"
        code, installed = self.run_installer("install", root, registry)
        self.assertEqual(code, 0, installed)
        unknown = root / "data/unknown-user-file.txt"
        unknown.write_text("keep", encoding="utf-8")
        code, removed = self.run_installer("uninstall", root, registry)
        self.assertEqual(code, 0, removed)
        self.assertTrue(unknown.is_file())
        self.assertTrue((root / "data/ownership.json").is_file())

    def test_installer_archives_are_inspectable(self) -> None:
        for operating_system in ("linux", "macos", "windows"):
            with self.subTest(os=operating_system):
                path = self.artifacts / f"infoboard-host-installer-{operating_system}-v0.1.0.zip"
                result = inspect_installer_archive(path, operating_system, "amd64")
                self.assertGreater(result["entries"], 10)


if __name__ == "__main__":
    unittest.main()
