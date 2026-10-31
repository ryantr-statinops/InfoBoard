"""Release-input loading and validation.

The release manifest is deliberately strict: packaging must fail closed when an
identity, version, target, or compatibility value is missing, duplicated, or
still a placeholder. The checked-in manifest currently uses deterministic
fixture IDs; publication remains blocked until the release owner supplies the
real store IDs.
"""

from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Any

EXPECTED_PERMISSIONS = ("storage", "tabs", "tabGroups", "nativeMessaging")
EXPECTED_HOST_NAME = "com.infoboard.host"
EXPECTED_PRODUCT_NAME = "InfoBoard"
EXPECTED_PRODUCT_ID = "com.infoboard"
SUPPORTED_BROWSERS = ("chrome", "edge")
SUPPORTED_OS = ("linux", "macos", "windows")
SUPPORTED_ARCHES = ("amd64", "arm64")
FIXTURE_EXTENSION_IDS = {
    "chrome": "abcdefghijklmnopabcdefghijklmnop",
    "edge": "ponmlkjihgfedcbaponmlkjihgfedcba",
}
SEMVER = re.compile(r"^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$")
EXTENSION_ID = re.compile(r"^[a-p]{32}$")
PLACEHOLDER_MARKERS = ("REPLACE", "PLACEHOLDER", "TODO", "<", ">", "${", "__")


class ReleaseConfigError(ValueError):
    """Raised when a release input cannot be trusted."""


def _require(condition: bool, message: str) -> None:
    if not condition:
        raise ReleaseConfigError(message)


def _read_json(path: Path) -> Any:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as error:
        raise ReleaseConfigError(f"missing release input: {path}") from error
    except json.JSONDecodeError as error:
        raise ReleaseConfigError(f"invalid JSON in {path}: {error}") from error


def _check_placeholder(value: str, field: str) -> None:
    upper = value.upper()
    _require(bool(value.strip()), f"{field} must not be empty")
    for marker in PLACEHOLDER_MARKERS:
        _require(marker not in upper, f"{field} contains an unresolved placeholder")


def _check_string_list(value: Any, field: str, expected: tuple[str, ...] | None = None) -> list[str]:
    _require(isinstance(value, list) and value, f"{field} must be a non-empty list")
    _require(all(isinstance(item, str) and item for item in value), f"{field} entries must be non-empty strings")
    _require(len(value) == len(set(value)), f"{field} must not contain duplicates")
    if expected is not None:
        _require(set(value) == set(expected), f"{field} must contain exactly {sorted(expected)}")
    return list(value)


def _validate_extension_ids(manifest: dict[str, Any], extension_ids: dict[str, Any] | None) -> None:
    ids = manifest.get("extension_ids")
    _require(isinstance(ids, dict), "extension_ids must be an object")
    _require(set(ids) == set(SUPPORTED_BROWSERS), "extension_ids must name chrome and edge")
    for browser in SUPPORTED_BROWSERS:
        value = ids.get(browser)
        _require(isinstance(value, str), f"extension_ids.{browser} must be a string")
        _check_placeholder(value, f"extension_ids.{browser}")
        _require(bool(EXTENSION_ID.fullmatch(value)), f"extension_ids.{browser} must be a literal 32-character a-p ID")
    if extension_ids is not None:
        _require(isinstance(extension_ids, dict), "extension-ids.json must be an object")
        _require(extension_ids.get("schema_version") == 1, "extension-ids.json schema_version must be 1")
        for browser in SUPPORTED_BROWSERS:
            listing = extension_ids.get(browser)
            _require(isinstance(listing, dict), f"extension-ids.json.{browser} must be an object")
            _require(listing.get("extension_id") == ids[browser], f"extension-ids.json.{browser} disagrees with release manifest")
            _check_placeholder(str(listing.get("listing_id", "")), f"extension-ids.json.{browser}.listing_id")
            _require(listing.get("channel") in {"fixture", "store"}, f"extension-ids.json.{browser}.channel is invalid")


def _validate_targets(targets: Any) -> None:
    _require(isinstance(targets, list) and targets, "targets must be a non-empty list")
    seen: set[tuple[str, str, str]] = set()
    for target in targets:
        _require(isinstance(target, dict), "each target must be an object")
        browser = target.get("browser")
        operating_system = target.get("os")
        arch = target.get("arch")
        _require(browser in SUPPORTED_BROWSERS, f"unsupported target browser: {browser}")
        _require(operating_system in SUPPORTED_OS, f"unsupported target OS: {operating_system}")
        _require(arch in SUPPORTED_ARCHES, f"unsupported target architecture: {arch}")
        key = (browser, operating_system, arch)
        _require(key not in seen, f"duplicate target: {key}")
        seen.add(key)
    expected = {(browser, operating_system) for browser in SUPPORTED_BROWSERS for operating_system in SUPPORTED_OS}
    _require({(browser, operating_system) for browser, operating_system, _ in seen} == expected,
             "targets must cover Chrome and Edge on Linux, macOS, and Windows")


def _validate_compatibility(manifest: dict[str, Any]) -> None:
    compatibility = manifest.get("compatibility")
    _require(isinstance(compatibility, dict), "compatibility must be an object")
    protocol = manifest["protocol"]["major"]
    ranking = manifest["ranking_model_version"]
    schema = manifest["storage_schema_version"]
    product_version = manifest["versions"]["product"]
    host_version = manifest["versions"]["host"]
    _require(protocol in compatibility.get("protocol_major", []), "compatibility.protocol_major must include the release protocol")
    _require(ranking in compatibility.get("ranking_model_versions", []), "compatibility.ranking_model_versions must include the release ranking model")
    _require(schema in compatibility.get("storage_schema_versions", []), "compatibility.storage_schema_versions must include the release schema")
    _require(product_version in compatibility.get("product_versions", []), "compatibility.product_versions must include the release product version")
    _require(host_version in compatibility.get("host_versions", []), "compatibility.host_versions must include the release host version")
    for field in ("protocol_major", "ranking_model_versions", "storage_schema_versions", "product_versions", "host_versions"):
        values = compatibility.get(field)
        _require(isinstance(values, list) and values, f"compatibility.{field} must be a non-empty list")
        _require(len(values) == len(set(map(str, values))), f"compatibility.{field} must not contain duplicates")


def validate_release_manifest(
    manifest: dict[str, Any],
    extension_ids: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Validate one release manifest and return it unchanged."""

    _require(isinstance(manifest, dict), "release manifest must be an object")
    _require(manifest.get("schema_version") == 1, "release manifest schema_version must be 1")

    product = manifest.get("product")
    _require(isinstance(product, dict), "product must be an object")
    _require(product.get("name") == EXPECTED_PRODUCT_NAME, "product.name must be InfoBoard")
    _require(product.get("id") == EXPECTED_PRODUCT_ID, "product.id must be com.infoboard")

    release_channel = manifest.get("release_channel")
    _require(release_channel in {"fixture", "store"}, "release_channel must be fixture or store")
    publication_status = manifest.get("publication_status")
    _require(publication_status in {"approved", "blocked_pending_store_ids"}, "publication_status is invalid")
    if publication_status == "approved":
        _require(release_channel == "store", "approved publication requires release_channel=store")
        ids = manifest.get("extension_ids", {})
        for browser in SUPPORTED_BROWSERS:
            _require(ids.get(browser) not in FIXTURE_EXTENSION_IDS.values(),
                     f"approved publication cannot use the {browser} fixture extension ID")

    versions = manifest.get("versions")
    _require(isinstance(versions, dict), "versions must be an object")
    for field in ("product", "chrome", "edge", "host"):
        value = versions.get(field)
        _require(isinstance(value, str), f"versions.{field} must be a string")
        _check_placeholder(value, f"versions.{field}")
        _require(bool(SEMVER.fullmatch(value)), f"versions.{field} must be semver")
    _require(versions["chrome"] == versions["product"], "Chrome package version must match the product version")
    _require(versions["edge"] == versions["product"], "Edge package version must match the product version")

    protocol = manifest.get("protocol")
    _require(isinstance(protocol, dict), "protocol must be an object")
    _require(isinstance(protocol.get("major"), int) and protocol["major"] > 0, "protocol.major must be a positive integer")
    _require(isinstance(protocol.get("minor"), int) and protocol["minor"] >= 0, "protocol.minor must be a non-negative integer")

    ranking = manifest.get("ranking_model_version")
    _require(isinstance(ranking, str) and ranking, "ranking_model_version must be a non-empty string")
    _check_placeholder(ranking, "ranking_model_version")
    _require(len(ranking.encode("utf-8")) <= 128, "ranking_model_version exceeds 128 bytes")

    schema = manifest.get("storage_schema_version")
    _require(isinstance(schema, int) and schema > 0, "storage_schema_version must be a positive integer")
    _require(manifest.get("host_name") == EXPECTED_HOST_NAME, f"host_name must be {EXPECTED_HOST_NAME}")

    _validate_extension_ids(manifest, extension_ids)
    _check_string_list(manifest.get("permissions"), "permissions", EXPECTED_PERMISSIONS)
    _validate_targets(manifest.get("targets"))
    _validate_compatibility(manifest)

    artifacts = manifest.get("artifacts")
    _require(isinstance(artifacts, dict), "artifacts must be an object")
    for field in ("chrome_extension", "edge_extension", "checksums", "build_input", "host_installer_prefix"):
        value = artifacts.get(field)
        _require(isinstance(value, str) and value, f"artifacts.{field} must be a non-empty string")
        _check_placeholder(value, f"artifacts.{field}")
    _require(artifacts["checksums"] == "SHA256SUMS", "artifacts.checksums must be SHA256SUMS")

    reproducibility = manifest.get("reproducibility")
    _require(isinstance(reproducibility, dict), "reproducibility must be an object")
    _require(reproducibility.get("source_date_epoch") == 0, "source_date_epoch must be 0")
    _require(reproducibility.get("archive_format") == "zip", "archive_format must be zip")
    _require(reproducibility.get("normalize_modes") is True, "normalize_modes must be true")
    _require(reproducibility.get("normalize_timestamps") is True, "normalize_timestamps must be true")
    _require(reproducibility.get("record_tool_versions") is True, "record_tool_versions must be true")

    return manifest


def load_release_manifest(root: Path) -> dict[str, Any]:
    """Load and validate the checked-in release manifest and ID mapping."""

    config_root = root / "packaging" / "config"
    manifest = _read_json(config_root / "release-manifest.json")
    extension_ids = _read_json(config_root / "extension-ids.json")
    return validate_release_manifest(manifest, extension_ids)


def load_release_file(path: Path, extension_ids: dict[str, Any] | None = None) -> dict[str, Any]:
    return validate_release_manifest(_read_json(path), extension_ids)
