#!/usr/bin/env python3
"""Byte-for-byte sentinel audit for IP-17 persisted/exported outputs."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--fixtures", type=Path, required=True)
    parser.add_argument("--diagnostics", type=Path, required=True)
    args = parser.parse_args()

    fixture_path = args.fixtures / "redaction-sentinels.json"
    with fixture_path.open("r", encoding="utf-8") as handle:
        fixture = json.load(handle)
    sentinels = fixture.get("sensitive_values", [])
    if not isinstance(sentinels, list) or not sentinels:
        print("FAIL: redaction fixture has no sentinels", file=sys.stderr)
        return 1

    input_text = read_text(fixture_path)
    missing_from_input = [sentinel for sentinel in sentinels if sentinel not in input_text]
    if missing_from_input:
        print(f"FAIL: sentinel fixture is not exercising every value: {missing_from_input}", file=sys.stderr)
        return 1

    findings: list[str] = []
    files = sorted(path for path in args.diagnostics.rglob("*") if path.is_file())
    if not files:
        findings.append("no_output_files")
    for path in files:
        text = read_text(path)
        try:
            json.loads(text)
        except json.JSONDecodeError as error:
            findings.append(f"invalid_json:{path.name}:{error.msg}")
            continue
        for sentinel in sentinels:
            if sentinel in text:
                findings.append(f"sensitive_value:{path.name}")

    if findings:
        print("FAIL: " + ", ".join(sorted(set(findings))), file=sys.stderr)
        return 1
    print(f"PASS: {len(sentinels)} sentinels absent from {len(files)} output files")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
