"""Small subprocess and tool-version helpers for release assembly."""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path
from typing import Sequence


class ToolError(RuntimeError):
    pass


def _native_command(command: Sequence[str]) -> list[str]:
    values = list(command)
    if os.name == "nt" and values:
        executable = values[0].lower()
        if executable in {"npm", "npx"} or executable.endswith((".cmd", ".bat")):
            return ["cmd", "/c", *values]
    return values


def run(command: Sequence[str], cwd: Path, env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    merged_env = os.environ.copy()
    merged_env.update(env or {})
    native = _native_command(command)
    completed = subprocess.run(
        native,
        cwd=str(cwd),
        env=merged_env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if completed.returncode != 0:
        raise ToolError(
            f"command failed ({completed.returncode}): {' '.join(command)}\n"
            f"stdout:\n{completed.stdout}\nstderr:\n{completed.stderr}"
        )
    return completed


def tool_version(command: Sequence[str]) -> str:
    try:
        return run(command, Path.cwd()).stdout.strip().splitlines()[0]
    except (ToolError, IndexError):
        return "unavailable"


def tool_versions(root: Path) -> dict[str, str]:
    return {
        "python": sys.version.split()[0],
        "go": tool_version(["go", "version"]),
        "node": tool_version(["node", "--version"]),
        "npm": tool_version(["npm", "--version"]),
    }


def repo_head(root: Path) -> str:
    try:
        return run(["git", "rev-parse", "HEAD"], root).stdout.strip()
    except ToolError:
        return "unknown"


def repo_status(root: Path) -> str:
    try:
        return run(["git", "status", "--porcelain"], root).stdout.strip()
    except ToolError:
        return "unknown"
