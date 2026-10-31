from __future__ import annotations

import copy
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

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
