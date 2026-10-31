"""Host binary and host-manifest assembly."""

from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Any

from .config import ReleaseConfigError
from .host_manifest import render_host_manifest, write_host_manifest
from .paths import canonical_binary_path
from .tooling import ToolError, run

BINARY_NAME = "infoboard-host"
BINARY_NAME_WINDOWS = "infoboard-host.exe"


def registered_origins(release: dict[str, Any]) -> str:
    # The host compares the browser-provided origin exactly. The Native
    # Messaging manifest uses the same IDs with a trailing /* wildcard, but the
    # compiled host allowlist must contain the exact origins.
    return ",".join(
        f"chrome-extension://{release['extension_ids'][browser]}" for browser in ("chrome", "edge")
    )


def target_platforms(release: dict[str, Any]) -> list[tuple[str, str]]:
    return sorted({(target["os"], target["arch"]) for target in release["targets"]})


def build_host_binary(
    root: Path,
    output: Path,
    release: dict[str, Any],
    operating_system: str,
    arch: str,
) -> Path:
    binary_name = BINARY_NAME_WINDOWS if operating_system == "windows" else BINARY_NAME
    destination = Path(output).resolve() / "host" / operating_system / arch / binary_name
    destination.parent.mkdir(parents=True, exist_ok=True)
    env = {
        "GOOS": operating_system if operating_system != "macos" else "darwin",
        "GOARCH": arch,
        "CGO_ENABLED": "0",
        "SOURCE_DATE_EPOCH": "0",
    }
    ldflags = (
        f"-X main.hostVersion={release['versions']['host']} "
        f"-X main.registeredOrigins={registered_origins(release)}"
    )
    try:
        run(
            [
                "go",
                "build",
                "-trimpath",
                "-buildvcs=false",
                f"-ldflags={ldflags}",
                "-o",
                str(destination),
                "./cmd/infoboard-host",
            ],
            root / "host",
            env=env,
        )
    except ToolError as error:
        raise ReleaseConfigError(f"host build failed for {operating_system}/{arch}: {error}") from error
    if not destination.is_file() or destination.stat().st_size == 0:
        raise ReleaseConfigError(f"host binary is missing after build: {destination}")
    return destination


def build_host_manifests(root: Path, output: Path, release: dict[str, Any]) -> list[Path]:
    results: list[Path] = []
    for operating_system, arch in target_platforms(release):
        binary_path = canonical_binary_path(operating_system)
        for browser in ("chrome", "edge"):
            manifest = render_host_manifest(
                root / "packaging" / "host" / browser / "com.infoboard.host.json.tmpl",
                binary_path,
                release["extension_ids"][browser],
            )
            destination = output / "host-manifests" / operating_system / arch / browser / "com.infoboard.host.json"
            write_host_manifest(destination, manifest, operating_system, release["extension_ids"][browser])
            results.append(destination)
    return results


def inspect_host_manifests(output: Path, release: dict[str, Any]) -> list[dict[str, Any]]:
    results: list[dict[str, Any]] = []
    for operating_system, arch in target_platforms(release):
        for browser in ("chrome", "edge"):
            path = output / "host-manifests" / operating_system / arch / browser / "com.infoboard.host.json"
            if not path.is_file():
                raise ReleaseConfigError(f"missing generated host manifest: {path}")
            manifest = json.loads(path.read_text(encoding="utf-8"))
            if manifest.get("name") != release["host_name"]:
                raise ReleaseConfigError("generated host manifest name drifted")
            if manifest.get("type") != "stdio":
                raise ReleaseConfigError("generated host manifest type drifted")
            expected = f"chrome-extension://{release['extension_ids'][browser]}/*"
            if manifest.get("allowed_origins") != [expected]:
                raise ReleaseConfigError("generated host manifest origin drifted")
            if not os.path.isabs(manifest.get("path", "")) and not manifest.get("path", "").startswith(("C:\\", "C:/")):
                raise ReleaseConfigError("generated host manifest path is not absolute")
            results.append(
                {
                    "os": operating_system,
                    "arch": arch,
                    "browser": browser,
                    "path": str(path),
                    "binary_path": manifest["path"],
                    "origin": expected,
                }
            )
    return results
