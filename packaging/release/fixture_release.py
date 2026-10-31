"""Build a second fixture release for upgrade/rollback evidence.

This module is intentionally limited to local verification. It derives a new
version from the checked-in release input and rebuilds the host binary and
extension archives with that version so the lifecycle evidence can exercise a
real upgrade pair without touching the release configuration.
"""

from __future__ import annotations

import copy
import json
from pathlib import Path
from typing import Any

from .archive import write_sha256sums
from .config import load_release_manifest
from .extension_package import build_extension_package
from .host_payload import build_host_binary, build_host_manifests, target_platforms


def make_release_artifacts(base: Path, version: str, root: Path | None = None) -> Path:
    root = root or Path(__file__).resolve().parents[2]
    release: dict[str, Any] = copy.deepcopy(load_release_manifest(root))
    release["versions"] = {key: version for key in release["versions"]}
    release["compatibility"]["product_versions"] = [version]
    release["compatibility"]["host_versions"] = [version]
    release["artifacts"]["chrome_extension"] = f"infoboard-extension-chrome-v{version}.zip"
    release["artifacts"]["edge_extension"] = f"infoboard-extension-edge-v{version}.zip"
    output = base / f"artifacts-{version}"
    output.mkdir(parents=True, exist_ok=True)
    (output / "release-manifest.json").write_text(json.dumps(release, indent=2) + "\n", encoding="utf-8")
    (output / "extension-ids.json").write_bytes((root / "packaging/config/extension-ids.json").read_bytes())
    for browser in ("chrome", "edge"):
        manifest = json.loads(
            (root / "packaging/extension" / browser / "manifest.json").read_text(encoding="utf-8")
        )
        manifest["version"] = version
        manifest_path = output / f"{browser}-manifest.json"
        manifest_path.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
        build_extension_package(
            output / release["artifacts"][f"{browser}_extension"],
            release,
            browser,
            manifest_path,
            root / "extension/dist",
        )
        manifest_path.unlink()
    for operating_system, arch in target_platforms(release):
        build_host_binary(root, output, release, operating_system, arch)
    build_host_manifests(root, output, release)
    for browser in ("chrome", "edge"):
        source = root / "packaging/host" / browser / "com.infoboard.host.json.tmpl"
        destination = output / "templates" / browser / "com.infoboard.host.json.tmpl"
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(source.read_bytes())
    lifecycle = output / "packaging/lifecycle/owned-paths.json"
    lifecycle.parent.mkdir(parents=True, exist_ok=True)
    lifecycle.write_bytes((root / "packaging/lifecycle/owned-paths.json").read_bytes())
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
