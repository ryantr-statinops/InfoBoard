"""Chrome Web Store and Edge Add-ons package assembly."""

from __future__ import annotations

import json
import stat
from pathlib import Path
from typing import Any

from .archive import ArchiveError, deterministic_zip, inspect_zip, read_zip_file
from .config import EXPECTED_PERMISSIONS, ReleaseConfigError

EXPECTED_COMMANDS = {
    "open-search": {
        "suggested_key": {"default": "Ctrl+Shift+Y", "mac": "Command+Shift+Y"},
        "description": "Open InfoBoard search",
    },
    "close-search": {
        "suggested_key": {"default": "Ctrl+Shift+X", "mac": "Command+Shift+X"},
        "description": "Close InfoBoard search",
    },
}
FORBIDDEN_MANIFEST_KEYS = {
    "host_permissions",
    "optional_permissions",
    "optional_host_permissions",
    "content_scripts",
    "externally_connectable",
    "web_accessible_resources",
}


def validate_extension_manifest(manifest: dict[str, Any], release: dict[str, Any]) -> dict[str, Any]:
    if not isinstance(manifest, dict):
        raise ReleaseConfigError("extension manifest must be an object")
    if manifest.get("manifest_version") != 3:
        raise ReleaseConfigError("extension manifest must be MV3")
    if manifest.get("name") != release["product"]["name"]:
        raise ReleaseConfigError("extension manifest product name drifted")
    if manifest.get("version") != release["versions"]["product"]:
        raise ReleaseConfigError("extension manifest version drifted")
    if set(manifest.get("permissions", [])) != set(EXPECTED_PERMISSIONS):
        raise ReleaseConfigError("extension manifest permission set drifted")
    if len(manifest.get("permissions", [])) != len(EXPECTED_PERMISSIONS):
        raise ReleaseConfigError("extension manifest permissions contain duplicates")
    for key in FORBIDDEN_MANIFEST_KEYS:
        if key in manifest:
            raise ReleaseConfigError(f"extension manifest contains forbidden key: {key}")
    background = manifest.get("background")
    if background != {"service_worker": "dist/src/background/service-worker.js", "type": "module"}:
        raise ReleaseConfigError("extension manifest service worker drifted")
    action = manifest.get("action")
    if action != {"default_title": "Open InfoBoard search", "default_popup": "dist/search/surface-shell.html"}:
        raise ReleaseConfigError("extension manifest action popup drifted")
    options = manifest.get("options_ui")
    if options != {"page": "dist/settings/settings-shell.html", "open_in_tab": True}:
        raise ReleaseConfigError("extension manifest options page drifted")
    if manifest.get("commands") != EXPECTED_COMMANDS:
        raise ReleaseConfigError("extension manifest command contract drifted")
    return manifest


def load_extension_manifest(path: Path, release: dict[str, Any]) -> dict[str, Any]:
    try:
        manifest = json.loads(path.read_text(encoding="utf-8"))
    except (FileNotFoundError, json.JSONDecodeError) as error:
        raise ReleaseConfigError(f"invalid extension manifest: {path}") from error
    return validate_extension_manifest(manifest, release)


def build_extension_package(
    output: Path,
    release: dict[str, Any],
    browser: str,
    manifest_path: Path,
    dist_root: Path,
) -> Path:
    manifest = load_extension_manifest(manifest_path, release)
    if not dist_root.is_dir():
        raise ReleaseConfigError(f"extension build output is missing: {dist_root}")
    entries: list[tuple[str, bytes, int]] = [
        ("manifest.json", (json.dumps(manifest, ensure_ascii=False, indent=2) + "\n").encode("utf-8"), 0o644)
    ]
    for path in sorted(dist_root.rglob("*")):
        if path.is_symlink():
            raise ArchiveError(f"symlink in extension output: {path}")
        if not path.is_file():
            continue
        relative = path.relative_to(dist_root).as_posix()
        mode = 0o755 if path.suffix == ".sh" else 0o644
        entries.append((f"dist/{relative}", path.read_bytes(), mode))
    deterministic_zip(output, entries)
    return output


def inspect_extension_package(path: Path, release: dict[str, Any], browser: str) -> dict[str, Any]:
    infos = inspect_zip(path)
    names = [info.filename for info in infos]
    if "manifest.json" not in names:
        raise ArchiveError("extension package has no manifest.json")
    if not any(name.startswith("dist/") for name in names):
        raise ArchiveError("extension package has no dist payload")
    manifest = json.loads(read_zip_file(path, "manifest.json").decode("utf-8"))
    validate_extension_manifest(manifest, release)
    expected_id = release["extension_ids"][browser]
    if len(expected_id) != 32:
        raise ReleaseConfigError("release extension ID is not literal")
    for info in infos:
        if info.filename.startswith("dist/"):
            if info.file_size == 0:
                raise ArchiveError(f"empty extension payload: {info.filename}")
    return {
        "browser": browser,
        "entries": len(infos),
        "manifest": manifest,
        "sha256": __import__("hashlib").sha256(path.read_bytes()).hexdigest(),
    }
