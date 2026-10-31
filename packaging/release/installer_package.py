"""Deterministic per-platform host installer archives."""

from __future__ import annotations

from pathlib import Path
from typing import Any

from .archive import deterministic_zip, inspect_zip, read_zip_file
from .config import ReleaseConfigError

BINARY_NAME = "infoboard-host"
BINARY_NAME_WINDOWS = "infoboard-host.exe"


def _binary_name(operating_system: str) -> str:
    return BINARY_NAME_WINDOWS if operating_system == "windows" else BINARY_NAME


def build_installer_archive(
    output: Path,
    root: Path,
    artifacts_root: Path,
    release: dict[str, Any],
    operating_system: str,
    arch: str,
) -> Path:
    entries: list[tuple[str, bytes, int]] = []

    installer_root = root / "packaging" / "installers" / operating_system
    for path in sorted(installer_root.glob("*.py")):
        entries.append((f"installer/{path.name}", path.read_bytes(), 0o755))

    release_package = root / "packaging" / "release"
    for path in sorted(release_package.glob("*.py")):
        if path.name.startswith("__pycache__"):
            continue
        entries.append((f"packaging/release/{path.name}", path.read_bytes(), 0o644))
    entries.append(("packaging/__init__.py", (root / "packaging" / "__init__.py").read_bytes(), 0o644))
    entries.append(
        (
            "packaging/lifecycle/owned-paths.json",
            (root / "packaging" / "lifecycle" / "owned-paths.json").read_bytes(),
            0o644,
        )
    )

    for name in ("release-manifest.json", "extension-ids.json"):
        entries.append((name, (artifacts_root / name).read_bytes(), 0o644))
    for browser in ("chrome", "edge"):
        template = artifacts_root / "templates" / browser / "com.infoboard.host.json.tmpl"
        entries.append((f"templates/{browser}/com.infoboard.host.json.tmpl", template.read_bytes(), 0o644))
        manifest = artifacts_root / "host-manifests" / operating_system / arch / browser / "com.infoboard.host.json"
        entries.append(
            (
                f"host-manifests/{operating_system}/{arch}/{browser}/com.infoboard.host.json",
                manifest.read_bytes(),
                0o644,
            )
        )

    binary = artifacts_root / "host" / operating_system / arch / _binary_name(operating_system)
    if not binary.is_file():
        raise ReleaseConfigError(f"installer host binary is missing: {binary}")
    entries.append((f"host/{operating_system}/{arch}/{binary.name}", binary.read_bytes(), 0o755))
    deterministic_zip(output, entries)
    return output


def inspect_installer_archive(path: Path, operating_system: str, arch: str) -> dict[str, Any]:
    infos = inspect_zip(path)
    names = {info.filename for info in infos}
    required = {
        "installer/install.py",
        "installer/uninstall.py",
        "installer/upgrade.py",
        "installer/repair.py",
        "installer/rollback.py",
        "installer/preflight.py",
        "release-manifest.json",
        "extension-ids.json",
        "packaging/lifecycle/owned-paths.json",
        "packaging/release/lifecycle.py",
        f"host/{operating_system}/{arch}/{_binary_name(operating_system)}",
    }
    missing = sorted(required - names)
    if missing:
        raise ReleaseConfigError(f"installer archive is missing: {', '.join(missing)}")
    if read_zip_file(path, "packaging/lifecycle/owned-paths.json") == b"":
        raise ReleaseConfigError("installer ownership manifest is empty")
    return {"os": operating_system, "arch": arch, "entries": len(infos), "path": str(path)}
