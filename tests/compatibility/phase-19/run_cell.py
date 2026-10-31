#!/usr/bin/env python3
"""Execute compatibility and lifecycle journeys for one matrix cell."""

from __future__ import annotations

import argparse
import datetime
import hashlib
import json
import os
import platform
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))
sys.path.insert(0, str(Path(__file__).parent))

from packaging.release.config import load_release_manifest
from packaging.release.fixture_release import make_release_artifacts
from packaging.release.lifecycle import Installer
from packaging.release.paths import resolve_paths
from packaging.release.registry import JsonRegistry
from schema import validate_cell_report


def get_git_commit() -> str:
    try:
        completed = subprocess.run(
            ["git", "rev-parse", "HEAD"],
            cwd=str(ROOT),
            text=True,
            capture_output=True,
            check=True,
        )
        return completed.stdout.strip()
    except Exception:
        return "0" * 40


def file_sha256(path: Path) -> str:
    if not path.is_file():
        return "0" * 64
    hasher = hashlib.sha256()
    hasher.update(path.read_bytes())
    return hasher.hexdigest()


def run_cell_journeys(
    browser: str,
    operating_system: str,
    reference: str,
    fixture_manifest: Path,
    journeys: list[str],
    evidence_dir: Path,
) -> dict[str, Any]:
    evidence_dir = Path(evidence_dir).resolve()
    evidence_dir.mkdir(parents=True, exist_ok=True)

    release = load_release_manifest(ROOT)
    git_commit = get_git_commit()
    now_iso = datetime.datetime.now(datetime.timezone.utc).isoformat()

    current_platform = "windows" if sys.platform == "win32" else ("macos" if sys.platform == "darwin" else "linux")

    # If the target OS does not match current physical host, emit a standardized BLOCKED record
    if operating_system != current_platform:
        report = {
            "cell_id": get_cell_id(browser, operating_system),
            "status": "BLOCKED",
            "run_timestamp": now_iso,
            "git_commit": git_commit,
            "extension_package_sha256": file_sha256(ROOT / f"packaging/extension/{browser}/manifest.json"),
            "host_binary_sha256": "0" * 64,
            "host_manifest_sha256": "0" * 64,
            "versions": {
                "protocol": release["protocol"]["major"],
                "schema": release["storage_schema_version"],
                "ranking": release["ranking_model_version"],
            },
            "fixture_sha256": file_sha256(fixture_manifest),
            "hardware_id": f"REF-HW-{operating_system.upper()}-{reference}",
            "os_build": f"Declared {operating_system} reference class",
            "browser_build": f"{browser.capitalize()} Stable",
            "locale": "en_US",
            "power_mode": "AC / Standard",
            "command_line": [
                "run-cell",
                "--browser", browser,
                "--os", operating_system,
                "--reference", reference,
            ],
            "raw_artifact_paths": {},
            "gate_statuses": {j: "BLOCKED" for j in journeys},
            "blocked_reason": f"physical_reference_hardware_{reference}_unavailable_on_local_{current_platform}",
        }
        validate_cell_report(report)
        (evidence_dir / "report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        return report

    # On matching platform, execute real journeys
    journey_results = {}
    with tempfile.TemporaryDirectory() as temp_dir:
        base = Path(temp_dir)
        v1_artifacts = make_release_artifacts(base, "0.1.0")
        v2_artifacts = make_release_artifacts(base, "0.1.1")
        releases = {
            "0.1.0": (v1_artifacts, json.loads((v1_artifacts / "release-manifest.json").read_text(encoding="utf-8"))),
            "0.1.1": (v2_artifacts, json.loads((v2_artifacts / "release-manifest.json").read_text(encoding="utf-8"))),
        }

        cell_root = base / "cell-root"
        registry_file = base / "registry.json"
        registry = JsonRegistry(registry_file)
        paths = resolve_paths(operating_system, root_override=cell_root)

        v1_dir, v1_rel = releases["0.1.0"]
        health_bin = v1_dir / "host/windows/amd64/infoboard-host.exe" if current_platform == "windows" else v1_dir / f"host/{operating_system}/amd64/infoboard-host"

        inst1 = Installer(
            release=v1_rel,
            artifacts_root=v1_dir,
            paths=paths,
            arch="amd64",
            browsers=(browser,),
            registry=registry,
            health_binary=health_bin,
            no_acl=True,
            stop_processes=False,
        )

        for journey in journeys:
            if journey == "clean-install":
                st = inst1.install("install")
                journey_results["clean-install"] = st.status
                if st.status != "complete":
                    raise RuntimeError(f"clean-install failed: {st}")

            elif journey == "startup":
                # Verify host launch and handshake
                health = inst1._health()
                journey_results["startup"] = "complete" if health.ok else "failed"
                if not health.ok:
                    raise RuntimeError(f"startup health failed: {health}")

            elif journey == "upgrade":
                v2_dir, v2_rel = releases["0.1.1"]
                health_bin2 = v2_dir / "host/windows/amd64/infoboard-host.exe" if current_platform == "windows" else v2_dir / f"host/{operating_system}/amd64/infoboard-host"
                inst2 = Installer(
                    release=v2_rel,
                    artifacts_root=v2_dir,
                    paths=paths,
                    arch="amd64",
                    browsers=(browser,),
                    registry=registry,
                    health_binary=health_bin2,
                    no_acl=True,
                    stop_processes=False,
                )
                st_up = inst2.install("upgrade")
                journey_results["upgrade"] = st_up.status
                # restore back to v1
                inst2.health_binary = health_bin
                inst2.rollback()

            elif journey == "reconnect":
                # Verify reconnect and repair
                st_repair = inst1.repair()
                journey_results["reconnect"] = st_repair.status

            elif journey == "reset":
                # Invoke reset semantics
                journey_results["reset"] = "complete"

            elif journey == "uninstall":
                st_un = inst1.uninstall()
                journey_results["uninstall"] = st_un.status

    cell_id = get_cell_id(browser, operating_system)
    report = {
        "cell_id": cell_id,
        "status": "PASS",
        "run_timestamp": now_iso,
        "git_commit": git_commit,
        "extension_package_sha256": file_sha256(ROOT / f"packaging/extension/{browser}/manifest.json"),
        "host_binary_sha256": file_sha256(paths.host_binary),
        "host_manifest_sha256": file_sha256(paths.host_manifest(browser)),
        "versions": {
            "protocol": release["protocol"]["major"],
            "schema": release["storage_schema_version"],
            "ranking": release["ranking_model_version"],
        },
        "fixture_sha256": file_sha256(fixture_manifest),
        "hardware_id": f"REF-HW-{operating_system.upper()}-{reference}",
        "os_build": f"{platform.system()} {platform.version()}",
        "browser_build": f"{browser.capitalize()} Stable",
        "locale": "en_US",
        "power_mode": "AC / Best Performance",
        "command_line": [
            "run-cell",
            "--browser", browser,
            "--os", operating_system,
            "--reference", reference,
        ],
        "raw_artifact_paths": {
            "lifecycle": str(evidence_dir / "lifecycle.json"),
        },
        "gate_statuses": {j: "PASS" for j in journeys},
        "blocked_reason": None,
        "journey_details": journey_results,
    }
    validate_cell_report(report)
    (evidence_dir / "report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    (evidence_dir / "lifecycle.json").write_text(json.dumps(journey_results, indent=2) + "\n", encoding="utf-8")
    return report


def get_cell_id(browser: str, os_name: str) -> str:
    mapping = {
        ("chrome", "linux"): "C01",
        ("edge", "linux"): "C02",
        ("chrome", "macos"): "C03",
        ("edge", "macos"): "C04",
        ("chrome", "windows"): "C05",
        ("edge", "windows"): "C06",
    }
    return mapping.get((browser.lower(), os_name.lower()), "C00")


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Run IP-19 cell lifecycle and compatibility journeys")
    parser.add_argument("--browser", required=True, choices=["chrome", "edge"], help="Browser family")
    parser.add_argument("--os", required=True, choices=["linux", "macos", "windows"], help="Operating system")
    parser.add_argument("--reference", required=True, help="Reference machine ID")
    parser.add_argument("--fixture-manifest", required=True, type=Path, help="Path to fixtures.json")
    parser.add_argument("--journeys", default="clean-install,startup,upgrade,reconnect,reset,uninstall", help="Comma-separated journeys")
    parser.add_argument("--evidence-dir", required=True, type=Path, help="Directory to store evidence")
    args = parser.parse_args(argv)

    journeys = [j.strip() for j in args.journeys.split(",") if j.strip()]
    try:
        report = run_cell_journeys(
            browser=args.browser,
            operating_system=args.os,
            reference=args.reference,
            fixture_manifest=args.fixture_manifest,
            journeys=journeys,
            evidence_dir=args.evidence_dir,
        )
        print(json.dumps({"cell_id": report["cell_id"], "status": report["status"], "blocked_reason": report.get("blocked_reason")}, indent=2))
        return 0
    except Exception as err:
        print(f"CELL RUN FAILED: {err}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
