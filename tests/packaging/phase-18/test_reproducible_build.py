from __future__ import annotations

import hashlib
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))

from packaging.release.archive import sha256_file, verify_sha256sums  # noqa: E402


class ReproducibleBuildTests(unittest.TestCase):
    def _build(self, output: Path, *extra: str) -> None:
        completed = subprocess.run(
            [
                sys.executable,
                "packaging/release/build.py",
                "--config",
                "packaging/config/release-manifest.json",
                "--output",
                str(output),
                "--skip-extension-build",
                *extra,
            ],
            cwd=str(ROOT),
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        if completed.returncode != 0:
            self.fail(f"build failed:\n{completed.stdout}\n{completed.stderr}")

    def test_two_clean_builds_are_byte_identical(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            first = root / "a"
            second = root / "b"
            self._build(first, "--skip-host-build")
            self._build(second, "--skip-host-build")
            self.assertEqual((first / "SHA256SUMS").read_bytes(), (second / "SHA256SUMS").read_bytes())
            self.assertEqual(verify_sha256sums(first), [])
            self.assertEqual(verify_sha256sums(second), [])
            self.assertEqual(
                sha256_file(first / "infoboard-extension-chrome-v0.1.0.zip"),
                sha256_file(second / "infoboard-extension-chrome-v0.1.0.zip"),
            )

    def test_checksum_verifier_rejects_modified_bytes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "a"
            self._build(output, "--skip-host-build")
            target = output / "infoboard-extension-chrome-v0.1.0.zip"
            target.write_bytes(target.read_bytes() + b"tampered")
            errors = verify_sha256sums(output)
            self.assertTrue(any("checksum mismatch" in error for error in errors))

    def test_host_build_produces_binary_and_exact_manifests(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "host"
            self._build(output)
            for operating_system, binary in (
                ("linux", "host/linux/amd64/infoboard-host"),
                ("macos", "host/macos/amd64/infoboard-host"),
                ("windows", "host/windows/amd64/infoboard-host.exe"),
            ):
                self.assertTrue((output / binary).is_file(), operating_system)
            for browser, extension_id in (
                ("chrome", "abcdefghijklmnopabcdefghijklmnop"),
                ("edge", "ponmlkjihgfedcbaponmlkjihgfedcba"),
            ):
                manifest_path = output / "host-manifests/linux/amd64" / browser / "com.infoboard.host.json"
                manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
                self.assertEqual(manifest["allowed_origins"], [f"chrome-extension://{extension_id}/*"])
                self.assertEqual(manifest["type"], "stdio")
                self.assertTrue(manifest["path"].startswith("/"))


if __name__ == "__main__":
    unittest.main()
