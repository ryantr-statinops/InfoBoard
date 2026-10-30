#!/usr/bin/env python3
"""Static permission and excluded-surface audit for IP-17."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

APPROVED_PERMISSIONS = {"nativeMessaging", "storage", "tabGroups", "tabs"}
APPROVED_COMMANDS = {"close-search", "open-search"}
DENIED_PERMISSIONS = {
    "bookmarks", "browsingData", "contentSettings", "cookies", "debugger",
    "declarativeNetRequest", "downloads", "history", "management", "privacy",
    "proxy", "scripting", "sessions", "topSites", "webNavigation", "webRequest",
}

SOURCE_PATTERNS = {
    "network_fetch": re.compile(r"\bfetch\s*\("),
    "network_xhr": re.compile(r"\bnew\s+XMLHttpRequest\b"),
    "network_websocket": re.compile(r"\bnew\s+WebSocket\b"),
    "network_eventsource": re.compile(r"\bnew\s+EventSource\b"),
    "network_beacon": re.compile(r"\bnavigator\.sendBeacon\b"),
    "tab_mutation": re.compile(r"\bchrome\.tabs\.(?:update|create|remove|move|discard|reload)\s*\("),
    "window_mutation": re.compile(r"\bchrome\.windows\.(?:create|update|remove)\s*\("),
    "denied_api": re.compile(
        r"\bchrome\.(?:bookmarks|browsingData|contentSettings|cookies|debugger|downloads|history|management|privacy|proxy|scripting|sessions|topSites|webNavigation|webRequest)\b"
    ),
    "sync_storage": re.compile(r"\bstorage\.sync\b"),
    "document_cookie": re.compile(r"\bdocument\.cookie\b"),
    "page_storage": re.compile(r"\b(?:localStorage|sessionStorage)\s*(?:\.|\[)"),
    "dynamic_code": re.compile(r"\b(?:eval\s*\(|new\s+Function\s*\()"),
}


def read_json(path: Path) -> dict:
    with path.open("r", encoding="utf-8") as handle:
        value = json.load(handle)
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object")
    return value


def audit_manifest(manifest: dict) -> list[str]:
    findings: list[str] = []
    if manifest.get("manifest_version") != 3:
        findings.append("manifest_version_not_3")

    permissions = manifest.get("permissions")
    if not isinstance(permissions, list) or set(permissions) != APPROVED_PERMISSIONS:
        findings.append("permissions_not_approved_set")
    permissions = permissions if isinstance(permissions, list) else []
    for denied in sorted(DENIED_PERMISSIONS):
        if denied in permissions:
            findings.append(f"denied_permission:{denied}")

    for key in ("host_permissions", "optional_permissions", "optional_host_permissions"):
        value = manifest.get(key)
        if value:
            findings.append(f"{key}_present")
            if any("*" in str(pattern) or str(pattern) == "<all_urls>" for pattern in value):
                findings.append("broad_host_pattern")

    if manifest.get("content_scripts"):
        findings.append("content_scripts_present")
    if manifest.get("web_accessible_resources"):
        findings.append("web_accessible_resources_present")

    commands = manifest.get("commands")
    if not isinstance(commands, dict) or set(commands) != APPROVED_COMMANDS:
        findings.append("commands_not_approved_set")
    return findings


def audit_sources(source_root: Path) -> list[str]:
    findings: list[str] = []
    for path in sorted(source_root.rglob("*")):
        if path.suffix not in {".ts", ".html"} or not path.is_file():
            continue
        text = path.read_text(encoding="utf-8")
        for name, pattern in SOURCE_PATTERNS.items():
            if pattern.search(text):
                findings.append(f"denied_source:{name}:{path.as_posix()}")
    return findings


def audit_matrix(fixtures: Path) -> list[str]:
    findings: list[str] = []
    matrix = read_json(fixtures / "permission-matrix.json")
    rows = matrix.get("matrix")
    if not isinstance(rows, list) or len(rows) != 8:
        findings.append("permission_matrix_row_count")
        rows = rows if isinstance(rows, list) else []
    capabilities: set[str] = set()
    for row in rows:
        if not isinstance(row, dict):
            findings.append("permission_matrix_row_not_object")
            continue
        capability = row.get("capability")
        if not isinstance(capability, str) or not capability:
            findings.append("permission_matrix_missing_capability")
            continue
        if capability in capabilities:
            findings.append(f"permission_matrix_duplicate:{capability}")
        capabilities.add(capability)
        for field in ("product_use", "owner", "denial_behavior", "evidence"):
            if not isinstance(row.get(field), str) or not row[field]:
                findings.append(f"permission_matrix_missing_{field}:{capability}")
        requirements = row.get("requirement_ids")
        if not isinstance(requirements, list) or not requirements:
            findings.append(f"permission_matrix_missing_requirements:{capability}")
    return findings


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--manifest", type=Path, required=True)
    parser.add_argument("--fixtures", type=Path, required=True)
    parser.add_argument("--source", type=Path, default=Path("extension/src"))
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    findings: list[str] = []
    manifest = read_json(args.manifest)
    findings.extend(audit_manifest(manifest))
    findings.extend(audit_matrix(args.fixtures))
    findings.extend(audit_sources(args.source))

    negative = read_json(args.fixtures / "manifest-denied.json")
    if not audit_manifest(negative):
        findings.append("negative_manifest_unexpectedly_passed")

    report = {
        "ok": not findings,
        "findings": findings,
        "permissions": sorted(manifest.get("permissions", [])),
        "commands": sorted(manifest.get("commands", {})),
        "source_root": args.source.as_posix(),
    }
    if args.json:
        print(json.dumps(report, indent=2, sort_keys=True))
    elif findings:
        print("FAIL: " + ", ".join(findings), file=sys.stderr)
    else:
        print("PASS: approved permissions, commands, capability matrix, and source surface")
    return 0 if not findings else 1


if __name__ == "__main__":
    raise SystemExit(main())
