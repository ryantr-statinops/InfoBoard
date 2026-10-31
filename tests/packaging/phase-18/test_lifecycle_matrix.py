from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
TESTS = Path(__file__).resolve().parent
sys.path.insert(0, str(ROOT))
sys.path.insert(0, str(TESTS))

from packaging.release.paths import resolve_paths  # noqa: E402
from release_fixtures import make_release_artifacts  # noqa: E402


class LifecycleMatrixTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.temp = tempfile.TemporaryDirectory()
        cls.base = Path(cls.temp.name)
        cls.v1 = make_release_artifacts(cls.base, "0.1.0")
        cls.v2 = make_release_artifacts(cls.base, "0.1.1")
        cls.matrix = json.loads(
            (ROOT / "fixtures/packaging/phase-18/lifecycle-matrix.json").read_text(encoding="utf-8")
        )

    @classmethod
    def tearDownClass(cls) -> None:
        cls.temp.cleanup()

    def run_installer(
        self,
        operation: str,
        root: Path,
        registry: Path,
        release_root: Path,
        health_binary: Path,
        browsers: str = "chrome,edge",
    ) -> tuple[int, dict]:
        completed = subprocess.run(
            [
                sys.executable,
                str(ROOT / "packaging/installers/windows" / f"{operation}.py"),
                "--release-root",
                str(release_root),
                "--root",
                str(root),
                "--registry-file",
                str(registry),
                "--health-binary",
                str(health_binary),
                "--no-acl",
                "--no-stop-processes",
                "--browsers",
                browsers,
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

    def test_matrix_statuses(self) -> None:
        root = self.base / "matrix"
        registry = self.base / "matrix-registry.json"
        releases = {"0.1.0": self.v1, "0.1.1": self.v2}
        for step in self.matrix["operations"]:
            with self.subTest(step=step["step"], operation=step["operation"]):
                release_root = releases[step["release"]]
                health_release = releases.get(step.get("health_release", step["release"]), release_root)
                health_binary = health_release / "host/windows/amd64/infoboard-host.exe"
                code, result = self.run_installer(
                    step["operation"], root, registry, release_root, health_binary
                )
                self.assertEqual(code, 0, result)
                self.assertEqual(result["status"], step["expected_status"], result)

    def test_browser_specific_roots_and_registration(self) -> None:
        root = self.base / "browser-specific"
        registry = self.base / "browser-specific-registry.json"
        code, installed = self.run_installer(
            "install",
            root,
            registry,
            self.v1,
            self.v1 / "host/windows/amd64/infoboard-host.exe",
            browsers="chrome",
        )
        self.assertEqual(code, 0, installed)
        self.assertTrue((root / "browsers/chrome/com.infoboard.host.json").is_file())
        self.assertFalse((root / "browsers/edge/com.infoboard.host.json").exists())
        keys = json.loads(registry.read_text(encoding="utf-8"))
        self.assertIn(r"HKCU\Software\Google\Chrome\NativeMessagingHosts\com.infoboard.host", keys)
        self.assertNotIn(r"HKCU\Software\Microsoft\Edge\NativeMessagingHosts\com.infoboard.host", keys)
        code, removed = self.run_installer(
            "uninstall",
            root,
            registry,
            self.v1,
            self.v1 / "host/windows/amd64/infoboard-host.exe",
            browsers="chrome",
        )
        self.assertEqual(code, 0, removed)
        self.assertFalse((root / "browsers/chrome/com.infoboard.host.json").exists())
        keys = json.loads(registry.read_text(encoding="utf-8"))
        self.assertNotIn(r"HKCU\Software\Google\Chrome\NativeMessagingHosts\com.infoboard.host", keys)

        code, installed = self.run_installer(
            "install",
            root,
            registry,
            self.v1,
            self.v1 / "host/windows/amd64/infoboard-host.exe",
            browsers="edge",
        )
        self.assertEqual(code, 0, installed)
        self.assertTrue((root / "browsers/edge/com.infoboard.host.json").is_file())
        self.assertFalse((root / "browsers/chrome/com.infoboard.host.json").exists())

    def test_unknown_file_preserves_ownership_and_allows_repair(self) -> None:
        root = self.base / "ownership"
        registry = self.base / "ownership-registry.json"
        code, installed = self.run_installer(
            "install", root, registry, self.v1, self.v1 / "host/windows/amd64/infoboard-host.exe"
        )
        self.assertEqual(code, 0, installed)
        unknown = root / "data/user-note.txt"
        unknown.write_text("keep", encoding="utf-8")
        code, removed = self.run_installer(
            "uninstall", root, registry, self.v1, self.v1 / "host/windows/amd64/infoboard-host.exe"
        )
        self.assertEqual(code, 0, removed)
        self.assertTrue(unknown.is_file())
        self.assertTrue((root / "data/ownership.json").is_file())
        code, reinstalled = self.run_installer(
            "install", root, registry, self.v1, self.v1 / "host/windows/amd64/infoboard-host.exe"
        )
        self.assertEqual(code, 0, reinstalled)
        self.assertTrue(unknown.is_file())
        self.assertTrue((root / "data/host/infoboard-host.exe").is_file())

    def test_platform_browser_roots_are_distinct(self) -> None:
        windows = resolve_paths("windows", root_override=self.base / "roots-windows")
        self.assertNotEqual(windows.browser_root("chrome"), windows.browser_root("edge"))
        self.assertNotEqual(windows.host_manifest("chrome"), windows.host_manifest("edge"))
        self.assertTrue(all(key.startswith("HKCU\\") for key in windows.registry_keys.values()))

        linux = resolve_paths("linux", env={"HOME": "/home/alice", "XDG_CONFIG_HOME": "/home/alice/.config"})
        self.assertNotEqual(linux.browser_root("chrome"), linux.browser_root("edge"))
        self.assertEqual(linux.registry_keys, {})

        macos = resolve_paths("macos", env={"HOME": "/Users/alice"})
        self.assertNotEqual(macos.browser_root("chrome"), macos.browser_root("edge"))
        self.assertEqual(macos.registry_keys, {})


if __name__ == "__main__":
    unittest.main()
