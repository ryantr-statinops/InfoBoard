"""Preflight validation before any user path is mutated."""

from __future__ import annotations

import json
import sqlite3
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Iterable

from .archive import verify_sha256sums
from .compatibility import release_tuple_errors
from .config import ReleaseConfigError
from .extension_package import inspect_extension_package
from .host_manifest import validate_host_manifest
from .ownership import root_is_owned_or_empty
from .paths import PlatformPaths
from .registry import Registry


@dataclass
class PreflightResult:
    ok: bool
    checks: dict[str, Any] = field(default_factory=dict)
    errors: list[str] = field(default_factory=list)

    def add(self, name: str, value: Any, error: str | None = None) -> None:
        self.checks[name] = value
        if error:
            self.errors.append(error)


def _read_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def _host_binary_name(platform: str) -> str:
    return "infoboard-host.exe" if platform == "windows" else "infoboard-host"


def preflight(
    release: dict[str, Any],
    artifacts_root: Path,
    paths: PlatformPaths,
    arch: str,
    browsers: Iterable[str],
    registry: Registry | None = None,
    require_publication: bool = False,
) -> PreflightResult:
    result = PreflightResult(ok=True)
    compatibility_errors = release_tuple_errors(release)
    result.add(
        "release_compatibility",
        not compatibility_errors,
        "; ".join(compatibility_errors) or None,
    )
    selected = tuple(browsers)
    if not selected or any(browser not in {"chrome", "edge"} for browser in selected):
        result.add("browsers", selected, "browsers must be chrome and/or edge")
        result.ok = False
        return result

    checksum_errors = verify_sha256sums(artifacts_root)
    result.add("checksums", "verified" if not checksum_errors else "failed", "; ".join(checksum_errors) or None)

    copied_manifest_path = artifacts_root / "release-manifest.json"
    if copied_manifest_path.is_file():
        copied = _read_json(copied_manifest_path)
        same = all(copied.get(key) == release.get(key) for key in ("versions", "protocol", "ranking_model_version", "storage_schema_version", "extension_ids", "compatibility"))
        result.add("release_manifest_copy", same, None if same else "artifact release manifest disagrees with the release input")
    else:
        result.add("release_manifest_copy", False, "artifact release manifest is missing")

    for browser in selected:
        target = next(
            (item for item in release["targets"] if item["browser"] == browser and item["os"] == paths.platform and item["arch"] == arch),
            None,
        )
        result.add(f"target.{browser}", bool(target), None if target else f"unsupported target: {browser}/{paths.platform}/{arch}")

        binary = artifacts_root / "host" / paths.platform / arch / _host_binary_name(paths.platform)
        result.add(f"host_binary.{browser}", binary.is_file(), None if binary.is_file() else f"host binary is missing: {binary}")

        manifest_path = artifacts_root / "host-manifests" / paths.platform / arch / browser / "com.infoboard.host.json"
        if not manifest_path.is_file():
            result.add(f"host_manifest.{browser}", False, f"host manifest is missing: {manifest_path}")
        else:
            try:
                manifest = _read_json(manifest_path)
                validate_host_manifest(manifest, release["extension_ids"][browser], paths.platform)
                result.add(f"host_manifest.{browser}", True)
            except (ReleaseConfigError, json.JSONDecodeError) as error:
                result.add(f"host_manifest.{browser}", False, str(error))

        extension_name = release["artifacts"][f"{browser}_extension"]
        extension_path = artifacts_root / extension_name
        if not extension_path.is_file():
            result.add(f"extension_package.{browser}", False, f"extension package is missing: {extension_path}")
        else:
            try:
                inspect_extension_package(extension_path, release, browser)
                result.add(f"extension_package.{browser}", True)
            except Exception as error:  # noqa: BLE001 - redacted preflight boundary
                result.add(f"extension_package.{browser}", False, str(error))

        if registry is not None and paths.platform == "windows":
            key = paths.registry_keys[browser]
            current = registry.get(key)
            expected = str(paths.host_manifest(browser))
            if current is not None and current != expected:
                result.add(f"registration.{browser}", False, "existing registration points to a different manifest")
            else:
                result.add(f"registration.{browser}", True)

    owned = root_is_owned_or_empty(paths.data_root)
    result.add("ownership_marker", owned, None if owned else "data root contains files without an InfoBoard ownership marker")

    build_input_path = artifacts_root / "build-input.json"
    if build_input_path.is_file():
        build_input = _read_json(build_input_path)
        compatible = build_input.get("compatibility") == release["compatibility"]
        result.add("compatibility", compatible, None if compatible else "artifact compatibility tuple drifted")
    else:
        result.add("compatibility", False, "build-input.json is missing")

    database = paths.config_root / "infoboard.sqlite"
    if database.is_file():
        try:
            connection = sqlite3.connect(database.as_uri() + "?mode=ro", uri=True)
            try:
                version = connection.execute("PRAGMA user_version").fetchone()[0]
            finally:
                connection.close()
            future = version > release["storage_schema_version"]
            result.add("storage_schema", version, "database schema is newer than the release" if future else None)
        except sqlite3.Error:
            result.add("storage_schema", "unreadable", "database schema could not be read")
    else:
        result.add("storage_schema", "absent")

    if require_publication and release["publication_status"] != "approved":
        result.add("publication", release["publication_status"], "publication is blocked pending approved store IDs")

    result.ok = not result.errors
    return result
