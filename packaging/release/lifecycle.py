"""One install/upgrade/repair/rollback/uninstall lifecycle contract.

The installer consumes the IP-15 owned-path manifest, writes only exact
registration artifacts, and never derives a deletion target from request data.
Platform-specific roots are resolved by :mod:`packaging.release.paths`; Windows
registration uses HKCU only.
"""

from __future__ import annotations

import argparse
import getpass
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Iterable

if __package__ in (None, ""):
    sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from packaging.release.archive import sha256_file
from packaging.release.config import load_release_file, load_release_manifest
from packaging.release.health import HealthResult, check_host
from packaging.release.host_manifest import render_host_manifest, validate_host_manifest, write_host_manifest
from packaging.release.ownership import (
    OwnedEntry,
    load_owned_entries,
    ownership_marker_path,
    read_ownership_marker,
    remove_entries,
    safe_remove,
    safe_remove_tree,
    verify_absent,
    write_ownership_marker,
)
from packaging.release.paths import PlatformPaths, resolve_paths
from packaging.release.preflight import preflight
from packaging.release.registry import Registry, create_registry

BINARY_NAME = "infoboard-host"
BINARY_NAME_WINDOWS = "infoboard-host.exe"
STATE_NAME = "install-state.json"
PAYLOAD_MANIFEST = "payload-manifest.json"


@dataclass
class LifecycleStatus:
    operation: str
    status: str
    retryable: bool = False
    error_class: str | None = None
    version: str | None = None
    steps: list[str] = field(default_factory=list)
    owned_state_verified: bool = False
    publication_status: str = ""
    blocked_reason: str | None = None
    health: dict[str, Any] | None = None
    details: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        return {
            "operation": self.operation,
            "status": self.status,
            "retryable": self.retryable,
            "error_class": self.error_class,
            "version": self.version,
            "steps": self.steps,
            "owned_state_verified": self.owned_state_verified,
            "publication_status": self.publication_status,
            "blocked_reason": self.blocked_reason,
            "health": self.health,
            "details": self.details,
        }


class LifecycleError(RuntimeError):
    def __init__(self, code: str, retryable: bool = False) -> None:
        super().__init__(code)
        self.code = code
        self.retryable = retryable


class Installer:
    def __init__(
        self,
        release: dict[str, Any],
        artifacts_root: Path,
        paths: PlatformPaths,
        arch: str,
        browsers: Iterable[str],
        registry: Registry,
        health_binary: Path | None = None,
        no_acl: bool = False,
        stop_processes: bool = True,
        require_publication: bool = False,
        dry_run: bool = False,
    ) -> None:
        self.release = release
        self.artifacts_root = artifacts_root
        self.paths = paths
        self.arch = arch
        self.browsers = tuple(browsers)
        self.registry = registry
        self.health_binary = health_binary
        self.no_acl = no_acl
        self.stop_processes = stop_processes
        self.require_publication = require_publication
        self.dry_run = dry_run
        self.state_path = paths.data_root / STATE_NAME

    # ------------------------------------------------------------------ state
    def _load_state(self) -> dict[str, Any] | None:
        if not self.state_path.is_file():
            return None
        try:
            return json.loads(self.state_path.read_text(encoding="utf-8"))
        except json.JSONDecodeError as error:
            raise LifecycleError("install_state_corrupt", retryable=False) from error

    def _write_state(
        self,
        version: str,
        previous: str | None,
        payload: list[dict[str, Any]],
        health: HealthResult,
    ) -> None:
        self.paths.data_root.mkdir(parents=True, exist_ok=True)
        manifests: dict[str, Any] = {}
        for browser in self.browsers:
            path = self.paths.host_manifest(browser)
            manifests[browser] = {"path": str(path), "sha256": sha256_file(path)}
        state = {
            "schema_version": 1,
            "product": "InfoBoard",
            "current_version": version,
            "previous_version": previous,
            "host_binary": str(self.paths.host_binary),
            "host_sha256": sha256_file(self.paths.host_binary),
            "manifests": manifests,
            "registry": {browser: self.paths.registry_keys.get(browser) for browser in self.browsers},
            "payload": payload,
            "verified": True,
            "hello_ack": health.hello_ack,
            "degraded_reason": health.degraded_reason,
            "publication_status": self.release["publication_status"],
        }
        temporary = self.state_path.with_name(self.state_path.name + ".tmp")
        temporary.write_text(json.dumps(state, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        os.replace(temporary, self.state_path)
        self._secure_file(self.state_path)

    # ----------------------------------------------------------------- payload
    def _binary_name(self) -> str:
        return BINARY_NAME_WINDOWS if self.paths.platform == "windows" else BINARY_NAME

    def _artifact_host_binary(self) -> Path:
        return self.artifacts_root / "host" / self.paths.platform / self.arch / self._binary_name()

    def _stage_payload(self, version: str) -> tuple[Path, list[dict[str, Any]]]:
        stage = self.paths.data_root / "staging" / version
        if stage.exists():
            self._remove_version_dir(stage)
        host = stage / "host" / self._binary_name()
        host.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(self._artifact_host_binary(), host)
        self._secure_file(host, executable=True)
        payload: list[dict[str, Any]] = [
            {"relative_path": f"host/{self._binary_name()}", "kind": "file", "sha256": sha256_file(host)}
        ]
        for browser in self.browsers:
            manifest = render_host_manifest(
                self._template_path(browser),
                self.paths.host_binary,
                self.release["extension_ids"][browser],
            )
            destination = stage / "manifests" / browser / "com.infoboard.host.json"
            write_host_manifest(destination, manifest, self.paths.platform, self.release["extension_ids"][browser])
            self._secure_file(destination)
            payload.append(
                {
                    "relative_path": f"manifests/{browser}/com.infoboard.host.json",
                    "kind": "file",
                    "sha256": sha256_file(destination),
                }
            )
        self._write_payload_manifest(stage, payload)
        self._write_version_marker(stage, version)
        write_ownership_marker(self.paths.data_root, "InfoBoard", version)
        self._secure_dir(stage)
        self._secure_file(ownership_marker_path(self.paths.data_root))
        return stage, payload

    def _template_path(self, browser: str) -> Path:
        return self.artifacts_root / "templates" / browser / "com.infoboard.host.json.tmpl"

    def _write_payload_manifest(self, root: Path, payload: list[dict[str, Any]]) -> None:
        (root / PAYLOAD_MANIFEST).write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        self._secure_file(root / PAYLOAD_MANIFEST)

    def _write_version_marker(self, root: Path, version: str) -> None:
        marker = root / "ownership.json"
        marker.write_text(
            json.dumps({"schema_version": 1, "product": "InfoBoard", "owner": "IP-18", "version": version}, indent=2)
            + "\n",
            encoding="utf-8",
        )
        self._secure_file(marker)

    def _read_payload_manifest(self, root: Path) -> list[dict[str, Any]] | None:
        path = root / PAYLOAD_MANIFEST
        if not path.is_file():
            return None
        try:
            payload = json.loads(path.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            return None
        if not isinstance(payload, list):
            return None
        return payload

    def _verify_payload(self, root: Path) -> list[str]:
        payload = self._read_payload_manifest(root)
        if payload is None:
            return ["payload manifest is missing or invalid"]
        errors: list[str] = []
        for record in payload:
            if not isinstance(record, dict) or not isinstance(record.get("relative_path"), str):
                errors.append("payload record is invalid")
                continue
            path = root / record["relative_path"]
            if not path.is_file():
                errors.append(f"payload file is missing: {record['relative_path']}")
            elif sha256_file(path) != record.get("sha256"):
                errors.append(f"payload checksum mismatch: {record['relative_path']}")
        return errors

    def _remove_version_dir(self, root: Path) -> bool:
        payload = self._read_payload_manifest(root)
        if payload is None:
            return False
        for record in payload:
            if not isinstance(record, dict) or not isinstance(record.get("relative_path"), str):
                continue
            try:
                safe_remove(root, OwnedEntry("payload", record["relative_path"], "file"))
            except Exception:  # noqa: BLE001 - cleanup continues and reports absence later
                pass
        for name in (PAYLOAD_MANIFEST, "ownership.json"):
            try:
                safe_remove(root, OwnedEntry(name, name, "file"))
            except Exception:  # noqa: BLE001
                pass
        for path in sorted(root.rglob("*"), key=lambda item: len(item.parts), reverse=True):
            if path.is_dir() and not any(path.iterdir()):
                try:
                    path.rmdir()
                except OSError:
                    pass
        if root.is_dir() and not any(root.iterdir()):
            try:
                root.rmdir()
            except OSError:
                pass
        return True

    # ---------------------------------------------------------------- publish
    def _backup_current(self, version: str) -> Path:
        backup = self.paths.data_root / "backup" / version
        if backup.exists():
            self._remove_version_dir(backup)
        host = backup / "host" / self._binary_name()
        host.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(self.paths.host_binary, host)
        self._secure_file(host, executable=True)
        payload: list[dict[str, Any]] = [
            {"relative_path": f"host/{self._binary_name()}", "kind": "file", "sha256": sha256_file(host)}
        ]
        for browser in self.browsers:
            source = self.paths.host_manifest(browser)
            if not source.is_file():
                continue
            destination = backup / "manifests" / browser / "com.infoboard.host.json"
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source, destination)
            self._secure_file(destination)
            payload.append(
                {
                    "relative_path": f"manifests/{browser}/com.infoboard.host.json",
                    "kind": "file",
                    "sha256": sha256_file(destination),
                }
            )
        self._write_payload_manifest(backup, payload)
        self._write_version_marker(backup, version)
        self._secure_dir(backup)
        return backup

    def _publish_stage(self, stage: Path) -> None:
        self.paths.data_root.mkdir(parents=True, exist_ok=True)
        self.paths.host_binary.parent.mkdir(parents=True, exist_ok=True)
        staged_host = stage / "host" / self._binary_name()
        temporary = self.paths.host_binary.with_name(self.paths.host_binary.name + ".new")
        shutil.copyfile(staged_host, temporary)
        self._secure_file(temporary, executable=True)
        os.replace(temporary, self.paths.host_binary)
        self._secure_file(self.paths.host_binary, executable=True)
        for browser in self.browsers:
            staged_manifest = stage / "manifests" / browser / "com.infoboard.host.json"
            manifest = json.loads(staged_manifest.read_text(encoding="utf-8"))
            validate_host_manifest(manifest, self.release["extension_ids"][browser], self.paths.platform)
            write_host_manifest(
                self.paths.host_manifest(browser),
                manifest,
                self.paths.platform,
                self.release["extension_ids"][browser],
            )
            self._secure_file(self.paths.host_manifest(browser))
        if self.paths.platform == "windows":
            for browser in self.browsers:
                key = self.paths.registry_keys[browser]
                self.registry.set(key, str(self.paths.host_manifest(browser)))

    def _restore_backup(self, version: str) -> bool:
        backup = self.paths.data_root / "backup" / version
        if self._verify_payload(backup):
            return False
        host = backup / "host" / self._binary_name()
        temporary = self.paths.host_binary.with_name(self.paths.host_binary.name + ".restore")
        shutil.copyfile(host, temporary)
        self._secure_file(temporary, executable=True)
        os.replace(temporary, self.paths.host_binary)
        for browser in self.browsers:
            source = backup / "manifests" / browser / "com.infoboard.host.json"
            if source.is_file():
                manifest = json.loads(source.read_text(encoding="utf-8"))
                write_host_manifest(
                    self.paths.host_manifest(browser),
                    manifest,
                    self.paths.platform,
                    self.release["extension_ids"][browser],
                )
                self._secure_file(self.paths.host_manifest(browser))
        if self.paths.platform == "windows":
            for browser in self.browsers:
                self.registry.set(self.paths.registry_keys[browser], str(self.paths.host_manifest(browser)))
        return True

    # ----------------------------------------------------------------- health
    def _health(self, version_override: str | None = None) -> HealthResult:
        binary = self.health_binary or self.paths.host_binary
        browser = self.browsers[0] if self.browsers else "chrome"
        origin = f"chrome-extension://{self.release['extension_ids'][browser]}"
        health_platform = "windows" if binary.suffix.lower() == ".exe" else self.paths.platform
        health_release = self.release
        if version_override:
            health_release = json.loads(json.dumps(self.release))
            health_release["versions"]["product"] = version_override
            health_release["versions"]["host"] = version_override
        with tempfile.TemporaryDirectory(prefix="infoboard-health-") as directory:
            return check_host(binary, origin, health_release, health_platform, Path(directory), browser=browser)

    def _verify_installed(self) -> list[str]:
        state = self._load_state()
        if state is None:
            return ["install state is missing"]
        errors: list[str] = []
        if state.get("current_version") != self.release["versions"]["host"]:
            errors.append("installed host version differs from the release")
        if not self.paths.host_binary.is_file():
            errors.append("host binary is missing")
        elif sha256_file(self.paths.host_binary) != state.get("host_sha256"):
            errors.append("host binary checksum drifted")
        for browser in self.browsers:
            path = self.paths.host_manifest(browser)
            if not path.is_file():
                errors.append(f"{browser} host manifest is missing")
                continue
            try:
                manifest = json.loads(path.read_text(encoding="utf-8"))
                validate_host_manifest(manifest, self.release["extension_ids"][browser], self.paths.platform)
            except Exception:  # noqa: BLE001 - redacted verification boundary
                errors.append(f"{browser} host manifest is invalid")
            if self.paths.platform == "windows":
                current = self.registry.get(self.paths.registry_keys[browser])
                if current != str(path):
                    errors.append(f"{browser} registration points to a different manifest")
        return errors

    # -------------------------------------------------------------- operations
    def preflight(self) -> Any:
        return preflight(
            self.release,
            self.artifacts_root,
            self.paths,
            self.arch,
            self.browsers,
            registry=self.registry,
            require_publication=self.require_publication,
        )

    def install(self, operation: str = "install") -> LifecycleStatus:
        result = self.preflight()
        if not result.ok:
            return LifecycleStatus(
                operation=operation,
                status="blocked",
                retryable=False,
                error_class="preflight_failed",
                version=self.release["versions"]["host"],
                publication_status=self.release["publication_status"],
                blocked_reason="; ".join(result.errors),
                details={"preflight": result.checks},
            )
        state = self._load_state()
        current = state.get("current_version") if state else None
        if current == self.release["versions"]["host"]:
            errors = self._verify_installed()
            health = self._health()
            if not errors and health.ok:
                return LifecycleStatus(
                    operation=operation,
                    status="no_op",
                    version=current,
                    steps=["preflight", "verify_installed", "health"],
                    owned_state_verified=True,
                    publication_status=self.release["publication_status"],
                    health=health.to_dict(),
                )
            return self.repair()
        if operation == "upgrade" and not current:
            return LifecycleStatus(
                operation=operation,
                status="blocked",
                error_class="not_installed",
                version=self.release["versions"]["host"],
                publication_status=self.release["publication_status"],
                blocked_reason="upgrade requires an installed release",
            )
        previous = current
        if previous:
            self._backup_current(previous)
        stage, payload = self._stage_payload(self.release["versions"]["host"])
        if self.dry_run:
            return LifecycleStatus(
                operation=operation,
                status="complete",
                version=self.release["versions"]["host"],
                steps=["preflight", "stage", "dry_run"],
                publication_status=self.release["publication_status"],
                details={"stage": str(stage)},
            )
        try:
            self._publish_stage(stage)
        except Exception as error:  # noqa: BLE001 - rollback boundary
            if previous:
                self._restore_backup(previous)
            return LifecycleStatus(
                operation=operation,
                status="retryable",
                retryable=True,
                error_class="publish_failed",
                version=previous,
                publication_status=self.release["publication_status"],
                blocked_reason="publish failed; previous release restored" if previous else "publish failed",
                details={"error": type(error).__name__},
            )
        health = self._health()
        if not health.ok:
            if previous:
                self._restore_backup(previous)
            self._remove_version_dir(stage)
            return LifecycleStatus(
                operation=operation,
                status="retryable",
                retryable=True,
                error_class=health.error_class or "health_check_failed",
                version=previous,
                publication_status=self.release["publication_status"],
                blocked_reason="health check failed; previous release restored" if previous else "health check failed",
                health=health.to_dict(),
            )
        self._write_state(self.release["versions"]["host"], previous, payload, health)
        self._remove_version_dir(stage)
        return LifecycleStatus(
            operation=operation,
            status="complete",
            version=self.release["versions"]["host"],
            steps=["preflight", "stage", "publish", "health", "state"],
            owned_state_verified=True,
            publication_status=self.release["publication_status"],
            health=health.to_dict(),
            details={"previous_version": previous},
        )

    def repair(self) -> LifecycleStatus:
        state = self._load_state()
        if state is None:
            return LifecycleStatus(
                operation="repair",
                status="blocked",
                error_class="not_installed",
                publication_status=self.release["publication_status"],
                blocked_reason="repair requires an installed release",
            )
        result = self.preflight()
        if not result.ok:
            return LifecycleStatus(
                operation="repair",
                status="blocked",
                error_class="preflight_failed",
                version=state.get("current_version"),
                publication_status=self.release["publication_status"],
                blocked_reason="; ".join(result.errors),
            )
        version = state.get("current_version")
        if version != self.release["versions"]["host"]:
            return LifecycleStatus(
                operation="repair",
                status="blocked",
                error_class="version_mismatch",
                version=version,
                publication_status=self.release["publication_status"],
                blocked_reason="repair artifacts do not match the installed version",
            )
        stage, payload = self._stage_payload(version)
        self._publish_stage(stage)
        health = self._health()
        if not health.ok:
            return LifecycleStatus(
                operation="repair",
                status="retryable",
                retryable=True,
                error_class=health.error_class or "health_check_failed",
                version=version,
                publication_status=self.release["publication_status"],
                blocked_reason="repair health check failed",
                health=health.to_dict(),
            )
        self._write_state(version, state.get("previous_version"), payload, health)
        self._remove_version_dir(stage)
        return LifecycleStatus(
            operation="repair",
            status="complete",
            version=version,
            steps=["preflight", "stage", "publish", "health", "state"],
            owned_state_verified=True,
            publication_status=self.release["publication_status"],
            health=health.to_dict(),
        )

    def rollback(self) -> LifecycleStatus:
        state = self._load_state()
        if state is None or not state.get("previous_version"):
            return LifecycleStatus(
                operation="rollback",
                status="retryable",
                retryable=True,
                error_class="no_previous_release",
                version=state.get("current_version") if state else None,
                publication_status=self.release["publication_status"],
                blocked_reason="reinstall the verified release; no previous backup exists",
            )
        previous = state["previous_version"]
        backup = self.paths.data_root / "backup" / previous
        errors = self._verify_payload(backup)
        if errors:
            return LifecycleStatus(
                operation="rollback",
                status="retryable",
                retryable=True,
                error_class="backup_unverified",
                version=state.get("current_version"),
                publication_status=self.release["publication_status"],
                blocked_reason="; ".join(errors),
            )
        if not self._restore_backup(previous):
            return LifecycleStatus(
                operation="rollback",
                status="retryable",
                retryable=True,
                error_class="restore_failed",
                version=state.get("current_version"),
                publication_status=self.release["publication_status"],
                blocked_reason="previous release could not be restored",
            )
        health = self._health(version_override=previous)
        if not health.ok:
            return LifecycleStatus(
                operation="rollback",
                status="retryable",
                retryable=True,
                error_class=health.error_class or "health_check_failed",
                version=previous,
                publication_status=self.release["publication_status"],
                blocked_reason="rollback health check failed",
                health=health.to_dict(),
            )
        payload = self._read_payload_manifest(backup) or []
        self._write_state(previous, None, payload, health)
        return LifecycleStatus(
            operation="rollback",
            status="complete",
            version=previous,
            steps=["verify_backup", "restore", "health", "state"],
            owned_state_verified=True,
            publication_status=self.release["publication_status"],
            health=health.to_dict(),
        )

    def uninstall(self) -> LifecycleStatus:
        state = self._load_state()
        if self.stop_processes:
            self._stop_host()
        failures: list[str] = []
        for browser in self.browsers:
            try:
                safe_remove(self.paths.browser_root(browser), OwnedEntry("host-manifest", "com.infoboard.host.json", "file"))
            except Exception:  # noqa: BLE001 - missing or unowned entries are reported below
                failures.append(f"{browser} manifest")
        try:
            safe_remove(self.paths.data_root, OwnedEntry("host-binary", f"host/{self._binary_name()}", "file"))
        except Exception:  # noqa: BLE001
            failures.append("host binary")
        if self.paths.platform == "windows":
            for browser in self.browsers:
                try:
                    self.registry.delete(self.paths.registry_keys[browser])
                except Exception:  # noqa: BLE001
                    failures.append(f"{browser} registry")
        db_entries, _ = load_owned_entries(self.artifacts_root)
        cleanup = remove_entries(self.paths.config_root, db_entries)
        failures.extend(cleanup.failed)
        for version in {state.get("current_version") if state else None, state.get("previous_version") if state else None}:
            if version:
                for parent in ("staging", "backup"):
                    directory = self.paths.data_root / parent / version
                    if directory.exists():
                        self._remove_version_dir(directory)
        state_path = self.paths.data_root / STATE_NAME
        if state_path.is_file():
            state_path.unlink()
        for parent in ("staging", "backup"):
            directory = self.paths.data_root / parent
            if directory.is_dir() and not any(directory.iterdir()):
                directory.rmdir()
        if self.paths.data_root.is_dir():
            for directory in sorted(
                (path for path in self.paths.data_root.rglob("*") if path.is_dir()),
                key=lambda path: len(path.parts),
                reverse=True,
            ):
                if not any(directory.iterdir()):
                    try:
                        directory.rmdir()
                    except OSError:
                        pass
        if self.paths.data_root.is_dir():
            entries = list(self.paths.data_root.iterdir())
            if len(entries) == 1 and entries[0].name == "ownership.json":
                entries[0].unlink()
            if not any(self.paths.data_root.iterdir()):
                self.paths.data_root.rmdir()
        remaining = verify_absent(self.paths.data_root, [OwnedEntry("host-binary", f"host/{self._binary_name()}", "file")])
        remaining.extend(verify_absent(self.paths.config_root, db_entries))
        for browser in self.browsers:
            path = self.paths.host_manifest(browser)
            if path.exists() or path.is_symlink():
                remaining.append(f"{browser} manifest")
        if self.paths.platform == "windows":
            for browser in self.browsers:
                if self.registry.get(self.paths.registry_keys[browser]) is not None:
                    remaining.append(f"{browser} registry")
        return LifecycleStatus(
            operation="uninstall",
            status="complete" if not failures and not remaining else "retryable",
            retryable=bool(failures or remaining),
            error_class=None if not failures and not remaining else "uninstall_incomplete",
            version=state.get("current_version") if state else None,
            steps=["stop_host", "remove_registration", "remove_owned_data", "verify_absent"],
            owned_state_verified=not remaining,
            publication_status=self.release["publication_status"],
            blocked_reason="; ".join(failures + remaining) or None,
        )

    # --------------------------------------------------------------- security
    def _secure_file(self, path: Path, executable: bool = False) -> None:
        if not path.exists():
            return
        try:
            os.chmod(path, 0o700 if executable else 0o600)
        except OSError:
            pass
        if self.paths.platform == "windows" and not self.no_acl:
            self._icacls(path)

    def _secure_dir(self, path: Path) -> None:
        try:
            os.chmod(path, 0o700)
        except OSError:
            pass
        if self.paths.platform == "windows" and not self.no_acl:
            self._icacls(path)

    def _icacls(self, path: Path) -> None:
        user = getpass.getuser()
        try:
            subprocess.run(
                ["icacls", str(path), "/inheritance:r", "/grant:r", f"{user}:(F)"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                check=False,
            )
        except OSError:
            pass

    def _stop_host(self) -> bool:
        if not self.paths.host_binary.is_file():
            return True
        if self.paths.platform == "windows":
            script = (
                "$target = [System.IO.Path]::GetFullPath('" + str(self.paths.host_binary).replace("'", "''") + "'); "
                "Get-CimInstance Win32_Process | Where-Object { $_.ExecutablePath -eq $target } | "
                "ForEach-Object { Stop-Process -Id $_.ProcessId -Force }"
            )
            try:
                subprocess.run(
                    ["powershell", "-NoProfile", "-NonInteractive", "-Command", script],
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                    check=False,
                )
            except OSError:
                return False
            return True
        try:
            output = subprocess.run(["ps", "-eo", "pid=,args="], text=True, stdout=subprocess.PIPE, check=False).stdout
            for line in output.splitlines():
                parts = line.strip().split(None, 1)
                if len(parts) == 2 and parts[1].split()[0] == str(self.paths.host_binary):
                    os.kill(int(parts[0]), 15)
        except (OSError, ValueError, IndexError):
            return False
        return True


def _load_release(artifacts_root: Path, config_path: Path | None) -> dict[str, Any]:
    if config_path is not None:
        return load_release_file(config_path)
    copied = artifacts_root / "release-manifest.json"
    if copied.is_file():
        return load_release_file(copied)
    return load_release_manifest(Path(__file__).resolve().parents[2])


def main(platform: str, operation: str, argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=f"InfoBoard host {operation} ({platform})")
    parser.add_argument("--release-root", type=Path, required=True)
    parser.add_argument("--config", type=Path)
    parser.add_argument("--root", type=Path)
    parser.add_argument("--config-root", type=Path)
    parser.add_argument("--browser-root-chrome", type=Path)
    parser.add_argument("--browser-root-edge", type=Path)
    parser.add_argument("--registry-file", type=Path)
    parser.add_argument("--health-binary", type=Path)
    parser.add_argument("--arch", default="amd64")
    parser.add_argument("--browsers", default="chrome,edge")
    parser.add_argument("--no-acl", action="store_true")
    parser.add_argument("--no-stop-processes", action="store_true")
    parser.add_argument("--require-publication", action="store_true")
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args(argv)
    release = _load_release(args.release_root, args.config)
    browser_overrides = {}
    if args.browser_root_chrome:
        browser_overrides["chrome"] = args.browser_root_chrome
    if args.browser_root_edge:
        browser_overrides["edge"] = args.browser_root_edge
    paths = resolve_paths(
        platform,
        root_override=args.root,
        config_override=args.config_root,
        browser_overrides=browser_overrides or None,
    )
    registry = create_registry(platform, args.registry_file)
    browsers = tuple(item.strip() for item in args.browsers.split(",") if item.strip())
    installer = Installer(
        release=release,
        artifacts_root=args.release_root,
        paths=paths,
        arch=args.arch,
        browsers=browsers,
        registry=registry,
        health_binary=args.health_binary,
        no_acl=args.no_acl,
        stop_processes=not args.no_stop_processes,
        require_publication=args.require_publication,
        dry_run=args.dry_run,
    )
    if operation == "install":
        status = installer.install("install")
    elif operation == "upgrade":
        status = installer.install("upgrade")
    elif operation == "repair":
        status = installer.repair()
    elif operation == "rollback":
        status = installer.rollback()
    elif operation == "uninstall":
        status = installer.uninstall()
    elif operation == "preflight":
        result = installer.preflight()
        status = LifecycleStatus(
            operation="preflight",
            status="complete" if result.ok else "blocked",
            error_class=None if result.ok else "preflight_failed",
            version=release["versions"]["host"],
            publication_status=release["publication_status"],
            blocked_reason="; ".join(result.errors) or None,
            details={"preflight": result.checks},
        )
    else:
        raise SystemExit(f"unknown operation: {operation}")
    print(json.dumps(status.to_dict(), indent=2, sort_keys=True))
    if status.status in {"complete", "no_op"}:
        return 0
    if status.status == "retryable":
        return 2
    return 3


if __name__ == "__main__":
    raise SystemExit("run a platform installer wrapper")
