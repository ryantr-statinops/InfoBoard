#!/usr/bin/env python3
"""Verify InfoBoard release artifacts and negative fixtures."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

if __package__ in (None, ""):
    sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from packaging.release.archive import verify_sha256sums
from packaging.release.compatibility import release_tuple_errors
from packaging.release.config import (
    ReleaseConfigError,
    load_release_file,
    validate_release_manifest,
)
from packaging.release.extension_package import inspect_extension_package
from packaging.release.host_manifest import validate_host_manifest
from packaging.release.installer_package import inspect_installer_archive


def verify_artifacts(artifacts_root: Path, fixture_root: Path | None = None) -> dict[str, Any]:
    artifacts_root = Path(artifacts_root).resolve()
    manifest_path = artifacts_root / "release-manifest.json"
    if not manifest_path.is_file():
        raise ReleaseConfigError(f"release manifest missing in artifacts: {manifest_path}")

    release = load_release_file(manifest_path)
    extension_ids_path = artifacts_root / "extension-ids.json"
    if not extension_ids_path.is_file():
        raise ReleaseConfigError(f"extension-ids.json missing in artifacts: {extension_ids_path}")
    extension_ids = json.loads(extension_ids_path.read_text(encoding="utf-8"))
    validate_release_manifest(release, extension_ids)

    checksum_errors = verify_sha256sums(artifacts_root)
    if checksum_errors:
        raise ReleaseConfigError(f"checksum verification failed: {'; '.join(checksum_errors)}")

    compatibility_errors = release_tuple_errors(release)
    if compatibility_errors:
        raise ReleaseConfigError(f"compatibility error: {'; '.join(compatibility_errors)}")

    extension_checks = {}
    for browser in ("chrome", "edge"):
        archive_name = release["artifacts"][f"{browser}_extension"]
        archive_path = artifacts_root / archive_name
        if not archive_path.is_file():
            raise ReleaseConfigError(f"extension archive missing: {archive_path}")
        extension_checks[browser] = inspect_extension_package(archive_path, release, browser)

    host_manifest_checks: list[dict[str, Any]] = []
    manifests_root = artifacts_root / "host-manifests"
    if manifests_root.is_dir():
        for manifest_file in sorted(manifests_root.rglob("*.json")):
            manifest_json = json.loads(manifest_file.read_text(encoding="utf-8"))
            browser = manifest_file.parent.name
            operating_system = manifest_file.parent.parent.parent.name
            expected_id = release["extension_ids"][browser]
            validate_host_manifest(manifest_json, expected_id, operating_system)
            host_manifest_checks.append(
                {
                    "path": str(manifest_file.relative_to(artifacts_root)),
                    "valid": True,
                    "allowed_origins": manifest_json["allowed_origins"],
                }
            )

    installer_checks = {}
    for target in release["targets"]:
        os_name = target["os"]
        arch = target["arch"]
        installer_name = (
            f"{release['artifacts']['host_installer_prefix']}-{os_name}-v{release['versions']['product']}.zip"
        )
        installer_path = artifacts_root / installer_name
        if installer_path.is_file():
            installer_checks[f"{os_name}-{arch}"] = inspect_installer_archive(installer_path, os_name, arch)

    negative_checks = {}
    if fixture_root and Path(fixture_root).is_dir():
        fixture_dir = Path(fixture_root).resolve()
        negative_manifest_cases = {
            "release-manifest.missing-id.json": "extension_ids",
            "release-manifest.unresolved-id.json": "placeholder",
            "release-manifest.duplicate-version.json": "Chrome package version",
            "release-manifest.unsupported-target.json": "unsupported target OS",
            "release-manifest.permission-drift.json": "permissions",
        }
        for filename, expected_snippet in negative_manifest_cases.items():
            case_path = fixture_dir / filename
            if case_path.is_file():
                try:
                    bad_manifest = json.loads(case_path.read_text(encoding="utf-8"))
                    validate_release_manifest(bad_manifest)
                    raise AssertionError(f"expected failure for {filename}")
                except ReleaseConfigError as err:
                    if expected_snippet not in str(err):
                        raise AssertionError(f"{filename} failed with unexpected message: {err}")
                    negative_checks[filename] = "rejected_correctly"

        for browser in ("chrome", "edge"):
            for bad_suffix, expected_msg in [
                ("extra-origin.json", "allowed origin"),
                ("relative-path.json", "absolute"),
            ]:
                bad_file = fixture_dir / f"host-manifest.{browser}.{bad_suffix}"
                if bad_file.is_file():
                    try:
                        bad_manifest = json.loads(bad_file.read_text(encoding="utf-8"))
                        validate_host_manifest(bad_manifest, release["extension_ids"][browser], browser)
                        raise AssertionError(f"expected failure for {bad_file.name}")
                    except ReleaseConfigError as err:
                        if expected_msg not in str(err):
                            raise AssertionError(f"{bad_file.name} failed with unexpected message: {err}")
                        negative_checks[bad_file.name] = "rejected_correctly"

    return {
        "status": "verified",
        "artifacts_root": str(artifacts_root),
        "product_version": release["versions"]["product"],
        "compatibility": release["compatibility"],
        "extension_checks": extension_checks,
        "host_manifest_count": len(host_manifest_checks),
        "installer_checks": installer_checks,
        "negative_checks": negative_checks,
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Verify InfoBoard release artifacts")
    parser.add_argument("--artifacts", required=True, type=Path, help="Path to release artifacts directory")
    parser.add_argument("--fixture-root", type=Path, default=None, help="Path to test fixtures directory")
    args = parser.parse_args(argv)

    try:
        report = verify_artifacts(args.artifacts, args.fixture_root)
        print(json.dumps(report, indent=2))
        return 0
    except (ReleaseConfigError, AssertionError) as err:
        print(f"VERIFICATION FAILED: {err}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
