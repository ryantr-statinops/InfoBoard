"""Reviewed ownership manifest and path-safe removal.

The installer consumes the IP-15 owned-path manifest instead of inventing a
second cleanup policy. All roots are trusted composition inputs; request data
never supplies a deletion target.
"""

from __future__ import annotations

import json
import os
import re
import stat
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable

ENTRY_ID = re.compile(r"^[A-Za-z0-9._-]{1,64}$")
FORBIDDEN_PATH_PARTS = {"", ".", ".."}


class OwnershipError(ValueError):
    """Raised when an ownership entry or marker is unsafe."""


@dataclass(frozen=True)
class OwnedEntry:
    id: str
    relative_path: str
    kind: str


@dataclass
class CleanupResult:
    removed: list[str]
    missing: list[str]
    failed: list[str]
    preserved_unknown: list[str]

    @property
    def verified(self) -> bool:
        return not self.failed


def _validate_entry(raw: Any) -> OwnedEntry:
    if not isinstance(raw, dict):
        raise OwnershipError("ownership entry must be an object")
    entry_id = raw.get("id")
    relative_path = raw.get("relative_path")
    kind = raw.get("kind")
    if not isinstance(entry_id, str) or not ENTRY_ID.fullmatch(entry_id):
        raise OwnershipError("ownership entry id is invalid")
    if not isinstance(relative_path, str) or not relative_path:
        raise OwnershipError("ownership relative_path is invalid")
    if relative_path.startswith(("/", "\\")) or re.match(r"^[A-Za-z]:", relative_path):
        raise OwnershipError("ownership path must be relative")
    if any(character in relative_path for character in "*?["):
        raise OwnershipError("ownership path must not contain wildcards")
    parts = relative_path.replace("\\", "/").split("/")
    if any(part in FORBIDDEN_PATH_PARTS for part in parts):
        raise OwnershipError("ownership path escapes its root")
    if kind not in {"file", "directory"}:
        raise OwnershipError("ownership kind is invalid")
    return OwnedEntry(entry_id, "/".join(parts), kind)


def load_owned_entries(root: Path) -> tuple[list[OwnedEntry], list[OwnedEntry]]:
    path = root / "packaging" / "lifecycle" / "owned-paths.json"
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (FileNotFoundError, json.JSONDecodeError) as error:
        raise OwnershipError(f"invalid ownership manifest: {path}") from error
    if data.get("version") != 1:
        raise OwnershipError("ownership manifest version must be 1")
    entries = [_validate_entry(item) for item in data.get("entries", [])]
    packaging_entries = [_validate_entry(item) for item in data.get("packaging_entries", [])]
    all_entries = entries + packaging_entries
    if not all_entries:
        raise OwnershipError("ownership manifest must contain at least one entry")
    ids = [entry.id for entry in all_entries]
    paths = [entry.relative_path for entry in all_entries]
    if len(ids) != len(set(ids)):
        raise OwnershipError("ownership manifest contains duplicate ids")
    if len(paths) != len(set(paths)):
        raise OwnershipError("ownership manifest contains duplicate paths")
    return entries, packaging_entries


def _target(root: Path, entry: OwnedEntry) -> Path:
    target = root / Path(entry.relative_path)
    try:
        target.resolve().relative_to(root.resolve())
    except ValueError as error:
        raise OwnershipError("ownership target escapes root") from error
    return target


def safe_remove(root: Path, entry: OwnedEntry) -> str:
    """Remove one reviewed entry. Return removed, missing, or preserved_unknown."""

    target = _target(root, entry)
    try:
        info = target.lstat()
    except FileNotFoundError:
        return "missing"
    if target.is_symlink():
        raise OwnershipError(f"refusing symlink target: {entry.id}")
    if entry.kind == "file":
        if not stat.S_ISREG(info.st_mode):
            raise OwnershipError(f"owned file is not regular: {entry.id}")
        target.unlink()
        return "removed"
    if not stat.S_ISDIR(info.st_mode):
        raise OwnershipError(f"owned directory is not a directory: {entry.id}")
    _remove_tree(target)
    return "removed"


def _remove_tree(path: Path) -> None:
    info = path.lstat()
    if stat.S_ISLNK(info.st_mode):
        raise OwnershipError(f"refusing symlink during removal: {path}")
    if stat.S_ISREG(info.st_mode):
        path.unlink()
        return
    if not stat.S_ISDIR(info.st_mode):
        raise OwnershipError(f"refusing non-regular owned entry: {path}")
    for child in sorted(path.iterdir(), key=lambda item: item.name):
        _remove_tree(child)
    path.rmdir()


def remove_entries(root: Path, entries: Iterable[OwnedEntry]) -> CleanupResult:
    result = CleanupResult(removed=[], missing=[], failed=[], preserved_unknown=[])
    for entry in entries:
        try:
            outcome = safe_remove(root, entry)
        except (OwnershipError, OSError):
            result.failed.append(entry.id)
            continue
        if outcome == "removed":
            result.removed.append(entry.id)
        else:
            result.missing.append(entry.id)
    return result


def verify_absent(root: Path, entries: Iterable[OwnedEntry]) -> list[str]:
    missing: list[str] = []
    for entry in entries:
        target = _target(root, entry)
        if target.exists() or target.is_symlink():
            missing.append(entry.id)
    return missing


def ownership_marker_path(data_root: Path) -> Path:
    return data_root / "ownership.json"


def write_ownership_marker(data_root: Path, product: str, version: str) -> Path:
    data_root.mkdir(parents=True, exist_ok=True)
    marker = ownership_marker_path(data_root)
    payload = {
        "schema_version": 1,
        "product": product,
        "owner": "IP-18",
        "version": version,
    }
    marker.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
    return marker


def read_ownership_marker(data_root: Path) -> dict[str, Any] | None:
    marker = ownership_marker_path(data_root)
    if not marker.is_file():
        return None
    try:
        payload = json.loads(marker.read_text(encoding="utf-8"))
    except json.JSONDecodeError as error:
        raise OwnershipError("ownership marker is invalid JSON") from error
    if payload.get("product") != "InfoBoard" or payload.get("owner") != "IP-18":
        raise OwnershipError("ownership marker does not belong to InfoBoard")
    return payload


def root_is_owned_or_empty(data_root: Path) -> bool:
    if not data_root.exists():
        return True
    marker = read_ownership_marker(data_root)
    if marker is not None:
        return True
    return not any(data_root.iterdir())
