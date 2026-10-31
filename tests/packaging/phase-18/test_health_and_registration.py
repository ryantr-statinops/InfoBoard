from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))

from packaging.release.config import load_release_manifest  # noqa: E402
from packaging.release.health import check_host  # noqa: E402
from packaging.release.paths import resolve_paths  # noqa: E402
from packaging.release.preflight import preflight  # noqa: E402
from packaging.release.registry import JsonRegistry  # noqa: E402


class HealthAndRegistrationTests(unittest.TestCase):
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
        cls.health_binary = cls.artifacts / "host/windows/amd64/infoboard-host.exe"

    @classmethod
    def tearDownClass(cls) -> None:
        cls.temp.cleanup()

    def test_health_smoke_reports_bootstrap_and_persistence(self) -> None:
        origin = f"chrome-extension://{self.release['extension_ids']['chrome']}"
        result = check_host(self.health_binary, origin, self.release, "windows", self.base / "health-home")
        self.assertTrue(result.ok)
        self.assertEqual(result.host_version, "0.1.0")
        self.assertEqual(result.persistence, "ready")
        self.assertFalse(result.hello_ack)
        self.assertEqual(result.degraded_reason, "runtime_protocol_unwired")

    def test_health_rejects_unregistered_origin(self) -> None:
        result = check_host(
            self.health_binary,
            "chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            self.release,
            "windows",
            self.base / "health-wrong-origin",
        )
        self.assertFalse(result.ok)
        self.assertEqual(result.error_class, "host_bootstrap_rejected")

    def test_health_rejects_missing_binary(self) -> None:
        result = check_host(self.base / "missing-host.exe", "chrome-extension://" + self.release["extension_ids"]["chrome"], self.release, "windows", self.base / "health-missing")
        self.assertFalse(result.ok)
        self.assertEqual(result.error_class, "host_binary_missing")

    def test_preflight_does_not_mutate_a_clean_root(self) -> None:
        root = self.base / "preflight-clean"
        paths = resolve_paths("windows", root_override=root)
        before = sorted(str(path) for path in root.rglob("*")) if root.exists() else []
        result = preflight(
            self.release,
            self.artifacts,
            paths,
            "amd64",
            ("chrome", "edge"),
            registry=JsonRegistry(self.base / "preflight-clean-registry.json"),
        )
        after = sorted(str(path) for path in root.rglob("*")) if root.exists() else []
        self.assertTrue(result.ok, result.errors)
        self.assertEqual(before, after)
        self.assertFalse((paths.data_root / "ownership.json").exists())

    def test_preflight_reports_unknown_ownership_without_removing_files(self) -> None:
        root = self.base / "preflight-unknown"
        paths = resolve_paths("windows", root_override=root)
        paths.data_root.mkdir(parents=True)
        unknown = paths.data_root / "user-file.txt"
        unknown.write_text("keep", encoding="utf-8")
        result = preflight(self.release, self.artifacts, paths, "amd64", ("chrome",))
        self.assertFalse(result.ok)
        self.assertTrue(any("ownership" in error for error in result.errors))
        self.assertEqual(unknown.read_text(encoding="utf-8"), "keep")

    def test_platform_paths_keep_host_binary_and_data_separate(self) -> None:
        linux = resolve_paths("linux", env={"HOME": "/home/alice", "XDG_CONFIG_HOME": "/home/alice/.config"})
        self.assertEqual(linux.data_root.as_posix(), "/home/alice/.local/share/InfoBoard")
        self.assertEqual(linux.config_root.as_posix(), "/home/alice/.config/infoboard")
        self.assertEqual(linux.host_binary.as_posix(), "/home/alice/.local/share/InfoBoard/host/infoboard-host")
        self.assertEqual(linux.browser_root("chrome").as_posix(), "/home/alice/.config/google-chrome/NativeMessagingHosts")
        self.assertEqual(linux.browser_root("edge").as_posix(), "/home/alice/.config/microsoft-edge/NativeMessagingHosts")

        macos = resolve_paths("macos", env={"HOME": "/Users/alice"})
        self.assertEqual(macos.data_root.as_posix(), "/Users/alice/Library/Application Support/InfoBoard")
        self.assertEqual(macos.config_root.as_posix(), "/Users/alice/Library/Application Support/infoboard")
        self.assertEqual(macos.browser_root("chrome").as_posix(), "/Users/alice/Library/Application Support/Google/Chrome/NativeMessagingHosts")
        self.assertEqual(macos.browser_root("edge").as_posix(), "/Users/alice/Library/Application Support/Microsoft Edge/NativeMessagingHosts")

        windows = resolve_paths(
            "windows",
            env={
                "LOCALAPPDATA": r"C:\Users\alice\AppData\Local",
                "APPDATA": r"C:\Users\alice\AppData\Roaming",
                "USERPROFILE": r"C:\Users\alice",
            },
        )
        self.assertEqual(windows.data_root.as_posix(), "C:/Users/alice/AppData/Local/InfoBoard")
        self.assertEqual(windows.config_root.as_posix(), "C:/Users/alice/AppData/Roaming/infoboard")
        self.assertEqual(windows.host_binary.as_posix(), "C:/Users/alice/AppData/Local/InfoBoard/host/infoboard-host.exe")
        self.assertEqual(set(windows.registry_keys), {"chrome", "edge"})
        self.assertTrue(all(key.startswith("HKCU\\") for key in windows.registry_keys.values()))

    def test_json_registry_round_trip(self) -> None:
        registry = JsonRegistry(self.base / "registry-round-trip.json")
        key = r"HKCU\Software\Google\Chrome\NativeMessagingHosts\com.infoboard.host"
        self.assertIsNone(registry.get(key))
        registry.set(key, "manifest.json")
        self.assertEqual(registry.get(key), "manifest.json")
        registry.delete(key)
        self.assertIsNone(registry.get(key))


if __name__ == "__main__":
    unittest.main()
