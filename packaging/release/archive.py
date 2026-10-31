"""Deterministic archive and checksum helpers.

Archives contain only normalized regular files. Absolute names, parent
traversal, backslash separators, duplicate names, and symlink entries fail
closed before an artifact can be published.
"""

from __future__ import annotations

import hashlib
import json
import os
import stat
import zipfile
from pathlib import Path
from typing import Iterable, Sequence

ZIP_TIMESTAMP = (1980, 1, 1, 0, 0, 0)


class ArchiveError(ValueError):
    """Raised when an archive entry is unsafe or non-deterministic."""


def canonical_json_bytes(value: object) -> bytes:
    return (json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n").encode("utf-8")


def normalize_arcname(name: str) -> str:
    if not name or name.startswith("/") or name.startswith("\\"):
        raise ArchiveError(f"absolute archive path: {name!r}")
    if "\\" in name:
        raise ArchiveError(f"backslash archive path: {name!r}")
    normalized = name
    while normalized.startswith("./"):
        normalized = normalized[2:]
    parts = normalized.split("/")
    if any(part in {"", ".", ".."} for part in parts):
        raise ArchiveError(f"unsafe archive path: {name!r}")
    if ":" in parts[0]:
        raise ArchiveError(f"drive-qualified archive path: {name!r}")
    return normalized


def deterministic_zip(
    output: Path,
    entries: Iterable[tuple[str, bytes, int]],
) -> None:
    """Write a sorted, timestamp-normalized zip with regular-file entries only."""

    output.parent.mkdir(parents=True, exist_ok=True)
    normalized: list[tuple[str, bytes, int]] = []
    seen: set[str] = set()
    for name, content, mode in entries:
        arcname = normalize_arcname(name)
        if arcname in seen:
            raise ArchiveError(f"duplicate archive entry: {arcname}")
        seen.add(arcname)
        if not isinstance(content, (bytes, bytearray)):
            raise ArchiveError(f"archive content is not bytes: {arcname}")
        normalized.append((arcname, bytes(content), stat.S_IMODE(mode) or 0o644))
    normalized.sort(key=lambda item: item[0])
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for arcname, content, mode in normalized:
            info = zipfile.ZipInfo(arcname, ZIP_TIMESTAMP)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = (stat.S_IFREG | mode) << 16
            info.create_system = 3
            archive.writestr(info, content)


def inspect_zip(path: Path) -> list[zipfile.ZipInfo]:
    """Return sorted entries after rejecting unsafe or duplicate names."""

    try:
        with zipfile.ZipFile(path, "r") as archive:
            infos = archive.infolist()
    except (FileNotFoundError, zipfile.BadZipFile) as error:
        raise ArchiveError(f"invalid archive: {path}") from error
    names: set[str] = set()
    for info in infos:
        name = normalize_arcname(info.filename)
        if name != info.filename:
            raise ArchiveError(f"non-normalized archive entry: {info.filename!r}")
        if name in names:
            raise ArchiveError(f"duplicate archive entry: {name}")
        names.add(name)
        mode = info.external_attr >> 16
        if stat.S_ISLNK(mode):
            raise ArchiveError(f"symlink archive entry: {name}")
        if not stat.S_ISREG(mode) and mode != 0:
            raise ArchiveError(f"non-regular archive entry: {name}")
    return sorted(infos, key=lambda item: item.filename)


def read_zip_file(path: Path, name: str) -> bytes:
    with zipfile.ZipFile(path, "r") as archive:
        try:
            return archive.read(name)
        except KeyError as error:
            raise ArchiveError(f"missing archive entry: {name}") from error


def sha256_bytes(content: bytes) -> str:
    return hashlib.sha256(content).hexdigest()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def write_sha256sums(root: Path, output_name: str = "SHA256SUMS") -> Path:
    output = root / output_name
    lines: list[str] = []
    records: list[tuple[str, Path]] = []
    for path in root.rglob("*"):
        if not path.is_file() or path == output:
            continue
        records.append((path.relative_to(root).as_posix(), path))
    for relative, path in sorted(records, key=lambda item: item[0]):
        lines.append(f"{sha256_file(path)}  {relative}")
    output.write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")
    return output


def verify_sha256sums(root: Path, manifest_name: str = "SHA256SUMS") -> list[str]:
    manifest = root / manifest_name
    if not manifest.is_file():
        raise ArchiveError(f"missing checksum manifest: {manifest}")
    errors: list[str] = []
    seen: set[str] = set()
    previous = ""
    for line in manifest.read_text(encoding="utf-8").splitlines():
        if not line:
            continue
        digest, separator, relative = line.partition("  ")
        if separator != "  " or len(digest) != 64:
            errors.append(f"malformed checksum line: {line!r}")
            continue
        if relative <= previous:
            errors.append("checksum manifest is not lexicographically sorted")
        previous = relative
        if relative in seen:
            errors.append(f"duplicate checksum entry: {relative}")
        seen.add(relative)
        path = root / relative
        if not path.is_file():
            errors.append(f"checksum target is missing: {relative}")
            continue
        if sha256_file(path) != digest:
            errors.append(f"checksum mismatch: {relative}")
    for path in sorted(root.rglob("*")):
        if not path.is_file() or path == manifest:
            continue
        relative = path.relative_to(root).as_posix()
        if relative not in seen:
            errors.append(f"untracked artifact: {relative}")
    return errors


def collect_file_entries(root: Path, prefixes: Sequence[str] = ()) -> list[tuple[str, bytes, int]]:
    """Collect regular files below root, preserving only normalized relative paths."""

    entries: list[tuple[str, bytes, int]] = []
    for path in sorted(root.rglob("*")):
        if path.is_symlink():
            raise ArchiveError(f"symlink in payload: {path}")
        if not path.is_file():
            continue
        relative = path.relative_to(root).as_posix()
        if prefixes and not any(relative == prefix or relative.startswith(prefix.rstrip("/") + "/") for prefix in prefixes):
            continue
        mode = 0o755 if os.access(path, os.X_OK) else 0o644
        entries.append((relative, path.read_bytes(), mode))
    return entries
