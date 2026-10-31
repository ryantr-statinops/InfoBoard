#!/usr/bin/env python3
"""Build deterministic InfoBoard release artifacts."""

from __future__ import annotations

import argparse
import hashlib
import json
import shutil
import sys
from pathlib import Path
from typing import Any

if __package__ in (None, ""):
    sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from packaging.release.archive import canonical_json_bytes, sha256_file, write_sha256sums
from packaging.release.config import load_release_manifest
from packaging.release.extension_package import build_extension_package, inspect_extension_package
from packaging.release.host_payload import (
    build_host_binary,
    build_host_manifests,
    inspect_host_manifests,
    target_platforms,
)
from packaging.release.installer_package import build_installer_archive, inspect_installer_archive
from packaging.release.tooling import repo_head, repo_status, run, tool_versions


def _copy_inputs(root: Path, output: Path) -> None:
    config = root / "packaging" / "config"
    for name in ("release-manifest.json", "extension-ids.json"):
        shutil.copyfile(config / name, output / name)
    for browser in ("chrome", "edge"):
        source = root / "packaging" / "host" / browser / "com.infoboard.host.json.tmpl"
        destination = output / "templates" / browser / "com.infoboard.host.json.tmpl"
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source, destination)
    lifecycle = root / "packaging" / "lifecycle" / "owned-paths.json"
    lifecycle_destination = output / "packaging" / "lifecycle" / "owned-paths.json"
    lifecycle_destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(lifecycle, lifecycle_destination)


def _source_digest(root: Path) -> str:
    digest = hashlib.sha256()
    paths = [
        root / "extension" / "manifest.json",
        root / "packaging" / "config" / "release-manifest.json",
        root / "packaging" / "config" / "extension-ids.json",
        root / "packaging" / "extension" / "chrome" / "manifest.json",
        root / "packaging" / "extension" / "edge" / "manifest.json",
    ]
    for path in sorted(paths):
        digest.update(path.relative_to(root).as_posix().encode("utf-8"))
        digest.update(path.read_bytes())
    dist = root / "extension" / "dist"
    if dist.is_dir():
        for path in sorted(dist.rglob("*")):
            if path.is_file() and not path.is_symlink():
                digest.update(path.relative_to(root).as_posix().encode("utf-8"))
                digest.update(path.read_bytes())
    return digest.hexdigest()


def build(
    root: Path,
    config_path: Path,
    output: Path,
    skip_extension_build: bool = False,
    skip_host_build: bool = False,
) -> dict[str, Any]:
    release = load_release_manifest(root)
    if config_path.resolve() != (root / "packaging/config/release-manifest.json").resolve():
        # A caller may point at a fixture manifest; validate it as the release input.
        from packaging.release.config import load_release_file

        release = load_release_file(config_path)

    output.mkdir(parents=True, exist_ok=True)
    _copy_inputs(root, output)

    if not skip_extension_build:
        run(["npm", "run", "build:extension"], root)
    dist_root = root / "extension" / "dist"
    if not dist_root.is_dir():
        raise SystemExit("extension/dist is missing; run npm run build:extension")

    chrome_zip = output / release["artifacts"]["chrome_extension"]
    edge_zip = output / release["artifacts"]["edge_extension"]
    build_extension_package(
        chrome_zip,
        release,
        "chrome",
        root / "packaging" / "extension" / "chrome" / "manifest.json",
        dist_root,
    )
    build_extension_package(
        edge_zip,
        release,
        "edge",
        root / "packaging" / "extension" / "edge" / "manifest.json",
        dist_root,
    )
    inspect_extension_package(chrome_zip, release, "chrome")
    inspect_extension_package(edge_zip, release, "edge")

    host_artifacts: list[str] = []
    if not skip_host_build:
        for operating_system, arch in target_platforms(release):
            destination = build_host_binary(root, output, release, operating_system, arch)
            host_artifacts.append(destination.relative_to(output).as_posix())
        build_host_manifests(root, output, release)
    manifest_records = []
    installer_artifacts: list[str] = []
    if not skip_host_build:
        manifest_records = [
            {**record, "path": Path(record["path"]).relative_to(output).as_posix()}
            for record in inspect_host_manifests(output, release)
        ]
        for operating_system, arch in target_platforms(release):
            installer_name = f"{release['artifacts']['host_installer_prefix']}-{operating_system}-v{release['versions']['product']}.zip"
            installer_path = output / installer_name
            build_installer_archive(installer_path, root, output, release, operating_system, arch)
            inspect_installer_archive(installer_path, operating_system, arch)
            installer_artifacts.append(installer_name)

    build_input = {
        "schema_version": 1,
        "source_digest": _source_digest(root),
        "release_manifest_sha256": sha256_file(output / "release-manifest.json"),
        "extension_ids_sha256": sha256_file(output / "extension-ids.json"),
        "compatibility": release["compatibility"],
        "targets": release["targets"],
        "publication_status": release["publication_status"],
        "release_channel": release["release_channel"],
        "host_artifacts": host_artifacts,
        "host_manifests": manifest_records,
        "installer_artifacts": installer_artifacts,
        "tool_versions": tool_versions(root),
        "git_head": repo_head(root),
        "git_status": repo_status(root),
    }
    (output / "build-input.json").write_bytes(canonical_json_bytes(build_input))
    write_sha256sums(output)
    return {
        "output": str(output),
        "artifacts": [
            release["artifacts"]["chrome_extension"],
            release["artifacts"]["edge_extension"],
            release["artifacts"]["build_input"],
            release["artifacts"]["checksums"],
            *host_artifacts,
            *installer_artifacts,
        ],
        "publication_status": release["publication_status"],
        "source_digest": build_input["source_digest"],
        "chrome_sha256": sha256_file(chrome_zip),
        "edge_sha256": sha256_file(edge_zip),
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", type=Path, default=Path("packaging/config/release-manifest.json"))
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--skip-extension-build", action="store_true")
    parser.add_argument("--skip-host-build", action="store_true")
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    args = parser.parse_args(argv)
    summary = build(args.root, args.config, args.output, args.skip_extension_build, args.skip_host_build)
    print(json.dumps(summary, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
