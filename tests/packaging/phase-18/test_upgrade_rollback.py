from __future__ import annotations

import copy
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))

from packaging.release.archive import write_sha256sums  # noqa: E402
from packaging.release.config import load_release_manifest  # noqa: E402
from packaging.release.extension_package import build_extension_package  # noqa: E402
from packaging.release.host_payload import build_host_binary, build_host_manifests  # noqa: E402


def make_release_artifacts(base: Path, version: str) -> Path:
    release = copy.deepcopy(load_release_manifest(ROOT))
    release["versions"] = {key: version for key in release["versions"]}
    release["compatibility"]["product_versions"] = [version]
    release["compatibility"]["host_versions"] = [version]
    release["artifacts"]["chrome_extension"] = f"infoboard-extension-chrome-v{version}.zip"
    release["artifacts"]["edge_extension"] = f"infoboard-extension-edge-v{version}.zip"
    output = base / f"artifacts-{version}"
    output.mkdir(parents=True)
    (output / "release-manifest.json").write_text(json.dumps(release, indent=2) + "\n", encoding="utf-8")
    (output / "extension-ids.json").write_bytes((ROOT / "packaging/config/extension-ids.json").read_bytes())
    for browser in ("chrome", "edge"):
        manifest = json.loads(
            (ROOT / "packaging/extension" / browser / "manifest.json").read_text(encoding="utf-8")
        )
        manifest["version"] = version
        manifest_path = output / f"{browser}-manifest.json"
        manifest_path.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
        build_extension_package(
            output / release["artifacts"][f"{browser}_extension"],
            release,
            browser,
            manifest_path,
            ROOT / "extension/dist",
        )
        manifest_path.unlink()
    build_host_binary(ROOT, output, release, "windows", "amd64")
    build_host_manifests(ROOT, output, release)
    for browser in ("chrome", "edge"):
        source = ROOT / "packaging/host" / browser / "com.infoboard.host.json.tmpl"
        destination = output / "templates" / browser / "com.infoboard.host.json.tmpl"
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(source.read_bytes())
    lifecycle = output / "packaging/lifecycle/owned-paths.json"
    lifecycle.parent.mkdir(parents=True, exist_ok=True)
    lifecycle.write_bytes((ROOT / "packaging/lifecycle/owned-paths.json").read_bytes())
    (output / "build-input.json").write_text(
        json.dumps(
            {
                "schema_version": 1,
                "compatibility": release["compatibility"],
                "targets": release["targets"],
                "publication_status": release["publication_status"],
                "release_channel": release["release_channel"],
            },
            indent=2,
        )
        + "\n",
        encoding="utf-8",
    )
    write_sha256sums(output)
    return output


class UpgradeRollbackTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.temp = tempfile.TemporaryDirectory()
        cls.base = Path(cls.temp.name)
        cls.v1 = make_release_artifacts(cls.base, "0.1.0")
        cls.v2 = make_release_artifacts(cls.base, "0.1.1")

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
                "chrome,edge",
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

    def test_upgrade_then_rollback_round_trip(self) -> None:
        root = self.base / "round-trip"
        registry = self.base / "round-trip-registry.json"
        code, installed = self.run_installer(
            "install", root, registry, self.v1, self.v1 / "host/windows/amd64/infoboard-host.exe"
        )
        self.assertEqual(code, 0, installed)
        code, upgraded = self.run_installer(
            "upgrade", root, registry, self.v2, self.v2 / "host/windows/amd64/infoboard-host.exe"
        )
        self.assertEqual(code, 0, upgraded)
        self.assertEqual(upgraded["status"], "complete")
        self.assertEqual(upgraded["version"], "0.1.1")
        state = json.loads((root / "data/install-state.json").read_text(encoding="utf-8"))
        self.assertEqual(state["current_version"], "0.1.1")
        self.assertEqual(state["previous_version"], "0.1.0")
        self.assertTrue((root / "data/backup/0.1.0/payload-manifest.json").is_file())
        code, rolled_back = self.run_installer(
            "rollback", root, registry, self.v2, self.v1 / "host/windows/amd64/infoboard-host.exe"
        )
        self.assertEqual(code, 0, rolled_back)
        self.assertEqual(rolled_back["status"], "complete")
        self.assertEqual(rolled_back["version"], "0.1.0")
        state = json.loads((root / "data/install-state.json").read_text(encoding="utf-8"))
        self.assertEqual(state["current_version"], "0.1.0")
        self.assertIsNone(state["previous_version"])

    def test_failed_upgrade_restores_previous_release(self) -> None:
        root = self.base / "failed-upgrade"
        registry = self.base / "failed-upgrade-registry.json"
        v1_health = self.v1 / "host/windows/amd64/infoboard-host.exe"
        code, installed = self.run_installer("install", root, registry, self.v1, v1_health)
        self.assertEqual(code, 0, installed)
        # Use the 0.1.0 binary as the health probe while installing the 0.1.1
        # release, forcing the health gate to reject the upgrade.
        code, failed = self.run_installer("upgrade", root, registry, self.v2, v1_health)
        self.assertEqual(code, 2, failed)
        self.assertEqual(failed["status"], "retryable")
        self.assertEqual(failed["error_class"], "host_version_mismatch")
        state = json.loads((root / "data/install-state.json").read_text(encoding="utf-8"))
        self.assertEqual(state["current_version"], "0.1.0")
        self.assertIsNone(state["previous_version"])
        self.assertTrue((root / "data/host/infoboard-host.exe").is_file())

    def test_corrupt_backup_blocks_rollback(self) -> None:
        root = self.base / "corrupt-backup"
        registry = self.base / "corrupt-backup-registry.json"
        code, installed = self.run_installer(
            "install", root, registry, self.v1, self.v1 / "host/windows/amd64/infoboard-host.exe"
        )
        self.assertEqual(code, 0, installed)
        code, upgraded = self.run_installer(
            "upgrade", root, registry, self.v2, self.v2 / "host/windows/amd64/infoboard-host.exe"
        )
        self.assertEqual(code, 0, upgraded)
        (root / "data/backup/0.1.0/host/infoboard-host.exe").write_bytes(b"corrupt")
        code, rolled_back = self.run_installer(
            "rollback", root, registry, self.v2, self.v1 / "host/windows/amd64/infoboard-host.exe"
        )
        self.assertEqual(code, 2, rolled_back)
        self.assertEqual(rolled_back["status"], "retryable")
        self.assertEqual(rolled_back["error_class"], "backup_unverified")
        state = json.loads((root / "data/install-state.json").read_text(encoding="utf-8"))
        self.assertEqual(state["current_version"], "0.1.1")


if __name__ == "__main__":
    unittest.main()
