"""Render and validate browser-specific Native Messaging manifests."""

from __future__ import annotations

import json
import os
import re
import sys
from pathlib import Path, PurePosixPath, PureWindowsPath
from typing import Any

from .config import EXPECTED_HOST_NAME, EXTENSION_ID, ReleaseConfigError

ALLOWED_KEYS = {"name", "description", "path", "type", "allowed_origins"}
ORIGIN_PREFIX = "chrome-extension://"
ORIGIN_SUFFIX = "/*"


def is_absolute_platform_path(value: str, operating_system: str) -> bool:
    if not value or "__" in value or "${" in value:
        return False
    if operating_system == "windows":
        path = PureWindowsPath(value)
        return path.is_absolute() and len(path.drive) == 2 and path.drive[1] == ":"
    path = PurePosixPath(value)
    if path.is_absolute() and not value.startswith("//"):
        return True
    if sys.platform == "win32":
        path_win = PureWindowsPath(value)
        return path_win.is_absolute() and len(path_win.drive) == 2 and path_win.drive[1] == ":"
    return False


def render_host_manifest(template_path: Path, binary_path: Path | str, extension_id: str) -> dict[str, Any]:
    if not EXTENSION_ID.fullmatch(extension_id):
        raise ReleaseConfigError("host manifest extension ID must be a literal 32-character a-p ID")
    text = template_path.read_text(encoding="utf-8")
    text = text.replace('"__BINARY_PATH__"', json.dumps(str(binary_path)))
    text = text.replace("__EXTENSION_ID__", extension_id)
    try:
        manifest = json.loads(text)
    except json.JSONDecodeError as error:
        raise ReleaseConfigError(f"rendered host manifest is invalid JSON: {error}") from error
    return validate_host_manifest(manifest, extension_id, operating_system=_os_for_path(binary_path))


def _os_for_path(binary_path: Path | str) -> str:
    value = str(binary_path)
    if re.match(r"^[A-Za-z]:[\\/]", value):
        return "windows"
    return "linux"


def validate_host_manifest(
    manifest: dict[str, Any],
    expected_extension_id: str,
    operating_system: str,
) -> dict[str, Any]:
    if not isinstance(manifest, dict):
        raise ReleaseConfigError("host manifest must be an object")
    if set(manifest) - ALLOWED_KEYS:
        raise ReleaseConfigError("host manifest contains an undeclared key")
    if manifest.get("name") != EXPECTED_HOST_NAME:
        raise ReleaseConfigError(f"host manifest name must be {EXPECTED_HOST_NAME}")
    if manifest.get("type") != "stdio":
        raise ReleaseConfigError("host manifest type must be stdio")
    path = manifest.get("path")
    if not isinstance(path, str) or not is_absolute_platform_path(path, operating_system):
        raise ReleaseConfigError("host manifest path must be an absolute platform path")
    origins = manifest.get("allowed_origins")
    if not isinstance(origins, list) or len(origins) != 1:
        raise ReleaseConfigError("host manifest must contain exactly one allowed origin")
    origin = origins[0]
    expected = f"{ORIGIN_PREFIX}{expected_extension_id}{ORIGIN_SUFFIX}"
    if origin != expected:
        raise ReleaseConfigError("host manifest allowed origin does not match the release extension ID")
    if "*" == origin or origin.startswith("*"):
        raise ReleaseConfigError("host manifest contains a broad origin")
    return manifest


def write_host_manifest(path: Path, manifest: dict[str, Any], operating_system: str, expected_extension_id: str) -> None:
    validate_host_manifest(manifest, expected_extension_id, operating_system)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(path.name + ".tmp")
    temporary.write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    try:
        os.chmod(temporary, 0o600)
    except OSError:
        pass
    os.replace(temporary, path)
