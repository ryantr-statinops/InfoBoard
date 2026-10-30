#!/usr/bin/env python3
"""Record the IP-17 browser permission/private-context matrix.

A blocked environment is recorded with an exact reason and is never counted as a
pass. Branded Chrome/Edge sideload and private-context UI journeys require a
managed test profile; this runner deliberately does not fake those results.
"""

from __future__ import annotations

import argparse
import json
import os
import platform
import re
import shutil
import subprocess
import sys
from pathlib import Path

WINDOWS_PATHS = {
    "chrome": [
        Path(os.environ.get("ProgramFiles", "")) / "Google/Chrome/Application/chrome.exe",
        Path(os.environ.get("ProgramFiles(x86)", "")) / "Google/Chrome/Application/chrome.exe",
        Path(os.environ.get("LocalAppData", "")) / "Google/Chrome/Application/chrome.exe",
    ],
    "edge": [
        Path(os.environ.get("ProgramFiles", "")) / "Microsoft/Edge/Application/msedge.exe",
        Path(os.environ.get("ProgramFiles(x86)", "")) / "Microsoft/Edge/Application/msedge.exe",
        Path(os.environ.get("LocalAppData", "")) / "Microsoft/Edge/Application/msedge.exe",
    ],
}

POSIX_PATHS = {
    "chrome": [
        Path("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
        Path("/usr/bin/google-chrome"),
        Path("/usr/bin/google-chrome-stable"),
    ],
    "edge": [
        Path("/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"),
        Path("/usr/bin/microsoft-edge"),
        Path("/usr/bin/microsoft-edge-stable"),
    ],
}


def discover_browser(name: str) -> Path | None:
    paths = WINDOWS_PATHS if platform.system() == "Windows" else POSIX_PATHS
    for candidate in paths.get(name, []):
        if candidate.is_file():
            return candidate
    executable = shutil.which(name)
    return Path(executable) if executable else None


def browser_version(path: Path) -> str | None:
    try:
        completed = subprocess.run(
            [str(path), "--version"],
            capture_output=True,
            text=True,
            timeout=5,
            check=False,
        )
    except (OSError, subprocess.SubprocessError):
        return None
    output = (completed.stdout or completed.stderr).strip()
    match = re.search(r"\b\d+(?:\.\d+){1,3}\b", output)
    return match.group(0) if match else None


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--browsers", default="chrome,edge")
    parser.add_argument("--contexts", default="normal,private")
    parser.add_argument("--fixtures", type=Path, default=Path("fixtures/privacy/phase-17"))
    parser.add_argument("--output", type=Path, default=Path("fixtures/privacy/phase-17/output/browser-matrix.json"))
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    fixture = json.loads((args.fixtures / "excluded-surface.json").read_text(encoding="utf-8"))
    browsers = [item.strip() for item in args.browsers.split(",") if item.strip()]
    contexts = [item.strip() for item in args.contexts.split(",") if item.strip()]
    if not browsers or not contexts:
        print("FAIL: at least one browser and context are required", file=sys.stderr)
        return 1
    unknown = set(browsers) - {"chrome", "edge"}
    if unknown:
        print(f"FAIL: unsupported browser(s): {sorted(unknown)}", file=sys.stderr)
        return 1
    if set(contexts) - {"normal", "private"}:
        print("FAIL: contexts must be normal and/or private", file=sys.stderr)
        return 1

    entries: list[dict] = []
    for browser in browsers:
        path = discover_browser(browser)
        version = browser_version(path) if path is not None else None
        for context in contexts:
            if path is None:
                entries.append({
                    "browser": browser,
                    "context": context,
                    "status": "blocked",
                    "pass": False,
                    "reason": "browser_not_installed",
                    "browser_version": None,
                })
            else:
                entries.append({
                    "browser": browser,
                    "context": context,
                    "status": "blocked",
                    "pass": False,
                    "reason": "branded_extension_sideload_requires_managed_profile_or_manual_ui",
                    "browser_version": version,
                    "browser_path_found": True,
                })

    passed = sum(1 for entry in entries if entry["status"] == "pass")
    blocked = sum(1 for entry in entries if entry["status"] == "blocked")
    failed = sum(1 for entry in entries if entry["status"] == "fail")
    report = {
        "fixture_id": fixture["fixture_id"],
        "platform": platform.platform(),
        "browsers": browsers,
        "contexts": contexts,
        "entries": entries,
        "summary": {
            "total": len(entries),
            "pass": passed,
            "blocked": blocked,
            "fail": failed,
            "all_pass": passed == len(entries),
        },
        "excluded_surface": {
            "journeys": fixture["journeys"],
            "surfaces": fixture["excluded_surfaces"],
            "expected_observations": fixture["expected"]["observations"],
        },
        "note": "Blocked entries are environment limitations, not passes. A real branded matrix remains release-gate work.",
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    if args.json:
        print(json.dumps(report, indent=2, sort_keys=True))
    elif blocked == len(entries):
        print(f"BLOCKED: {blocked}/{len(entries)} matrix entries blocked; no entry counted as pass")
    elif failed:
        print(f"FAIL: {failed}/{len(entries)} matrix entries failed", file=sys.stderr)
    else:
        print(f"PASS: {passed}/{len(entries)} matrix entries passed")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
