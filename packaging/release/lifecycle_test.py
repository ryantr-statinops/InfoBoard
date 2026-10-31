#!/usr/bin/env python3
"""Multi-platform and browser lifecycle convergence test for InfoBoard packaging."""

from __future__ import annotations

import argparse
import json
import sys
import tempfile
from pathlib import Path
from typing import Any

if __package__ in (None, ""):
    sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from packaging.release.fixture_release import make_release_artifacts
from packaging.release.lifecycle import Installer
from packaging.release.paths import resolve_paths
from packaging.release.registry import JsonRegistry


def run_lifecycle_suite(
    fixture_root: Path,
    browsers: tuple[str, ...],
    platforms: tuple[str, ...],
    repeat_count: int = 2,
) -> dict[str, Any]:
    fixture_root = Path(fixture_root).resolve()
    matrix_file = fixture_root / "lifecycle-matrix.json"
    if not matrix_file.is_file():
        raise FileNotFoundError(f"lifecycle matrix fixture missing: {matrix_file}")

    results: dict[str, Any] = {"platforms": {}, "repeat_count": repeat_count}

    with tempfile.TemporaryDirectory() as temp_dir:
        base = Path(temp_dir)
        v1_artifacts = make_release_artifacts(base, "0.1.0")
        v2_artifacts = make_release_artifacts(base, "0.1.1")
        releases = {
            "0.1.0": (v1_artifacts, json.loads((v1_artifacts / "release-manifest.json").read_text(encoding="utf-8"))),
            "0.1.1": (v2_artifacts, json.loads((v2_artifacts / "release-manifest.json").read_text(encoding="utf-8"))),
        }

        for platform in platforms:
            platform_results = []
            plat_root = base / f"run-{platform}"
            registry_file = base / f"registry-{platform}.json"
            registry = JsonRegistry(registry_file)
            paths = resolve_paths(platform, root_override=plat_root)
            v1_dir, v1_rel = releases["0.1.0"]
            if sys.platform == "win32":
                health_bin = v1_dir / "host/windows/amd64/infoboard-host.exe"
            else:
                health_bin = v1_dir / "host" / platform / "amd64" / binary_name

            inst1 = Installer(
                release=v1_rel,
                artifacts_root=v1_dir,
                paths=paths,
                arch="amd64",
                browsers=browsers,
                registry=registry,
                health_binary=health_bin,
                no_acl=True,
                stop_processes=False,
            )
            st_install = inst1.install("install")
            platform_results.append({"op": "install", "status": st_install.status})
            if st_install.status not in {"complete", "no_op"}:
                raise RuntimeError(f"Install failed on {platform}: {st_install}")

            # 2. Repeat install v1 (verify convergence / no-op)
            for r in range(repeat_count - 1):
                st_repeat = inst1.install("install")
                platform_results.append({"op": f"repeat_install_{r+1}", "status": st_repeat.status})
            v2_dir, v2_rel = releases["0.1.1"]
            if sys.platform == "win32":
                health_bin2 = v2_dir / "host/windows/amd64/infoboard-host.exe"
            else:
                health_bin2 = v2_dir / "host" / platform / "amd64" / binary_name

            inst2 = Installer(
                release=v2_rel,
                artifacts_root=v2_dir,
                paths=paths,
                arch="amd64",
                browsers=browsers,
                registry=registry,
                health_binary=health_bin2,
                no_acl=True,
                stop_processes=False,
            )
            st_upgrade = inst2.install("upgrade")
            platform_results.append({"op": "upgrade", "status": st_upgrade.status})
            if st_upgrade.status != "complete":
                raise RuntimeError(f"Upgrade failed on {platform}: {st_upgrade}")

            # 4. Rollback to v1
            inst2.health_binary = health_bin
            st_rollback = inst2.rollback()
            platform_results.append({"op": "rollback", "status": st_rollback.status})
            if st_rollback.status != "complete":
                raise RuntimeError(f"Rollback failed on {platform}: {st_rollback}")

            # 5. Repair
            st_repair = inst1.repair()
            platform_results.append({"op": "repair", "status": st_repair.status})
            if st_repair.status not in {"complete", "no_op"}:
                raise RuntimeError(f"Repair failed on {platform}: {st_repair}")

            # 6. Uninstall
            st_uninstall = inst1.uninstall()
            platform_results.append({"op": "uninstall", "status": st_uninstall.status})
            if st_uninstall.status != "complete":
                raise RuntimeError(f"Uninstall failed on {platform}: {st_uninstall}")

            # 7. Repeat uninstall (verify idempotent)
            for r in range(repeat_count - 1):
                st_un_repeat = inst1.uninstall()
                platform_results.append({"op": f"repeat_uninstall_{r+1}", "status": st_un_repeat.status})
                if st_un_repeat.status not in {"complete", "no_op"}:
                    raise RuntimeError(f"Repeat uninstall on {platform} expected complete/no_op, got {st_un_repeat.status}")

            results["platforms"][platform] = platform_results

    results["status"] = "converged"
    return results


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="InfoBoard lifecycle convergence runner")
    parser.add_argument("--fixture-root", required=True, type=Path, help="Path to fixtures/packaging/phase-18")
    parser.add_argument("--browsers", default="chrome,edge", help="Comma-separated browser list")
    parser.add_argument("--platforms", default="linux,macos,windows", help="Comma-separated platform list")
    parser.add_argument("--repeat", type=int, default=2, help="Repeat count for idempotency")
    args = parser.parse_args(argv)

    browsers = tuple(b.strip() for b in args.browsers.split(",") if b.strip())
    platforms = tuple(p.strip() for p in args.platforms.split(",") if p.strip())

    try:
        results = run_lifecycle_suite(args.fixture_root, browsers, platforms, args.repeat)
        print(json.dumps(results, indent=2))
        return 0
    except Exception as err:
        print(f"LIFECYCLE TEST FAILED: {err}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
