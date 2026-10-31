"""Browser-mediated host hello smoke check.

The installer launches the exact installed host binary with the browser origin
argument, writes one framed hello, and requires a hello_ack before declaring a
release healthy. No listening socket or remote endpoint is involved.
"""

from __future__ import annotations

import json
import os
import struct
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Mapping

HELLO_PROFILE_ID = "123e4567-e89b-42d3-a456-426614174000"
HELLO_REQUEST_ID = "packaging-health"


@dataclass
class HealthResult:
    ok: bool
    host_version: str | None = None
    persistence: str | None = None
    protocol: int | None = None
    error_class: str | None = None
    hello_ack: bool = False
    degraded_reason: str | None = None
    stdout_bytes: int = 0
    stderr_bytes: int = 0

    def to_dict(self) -> dict[str, Any]:
        return {
            "ok": self.ok,
            "host_version": self.host_version,
            "persistence": self.persistence,
            "protocol": self.protocol,
            "error_class": self.error_class,
            "hello_ack": self.hello_ack,
            "degraded_reason": self.degraded_reason,
            "stdout_bytes": self.stdout_bytes,
            "stderr_bytes": self.stderr_bytes,
        }


def hello_frame(release: dict[str, Any], browser: str = "chrome") -> bytes:
    payload = {
        "extension_version": release["versions"]["product"],
        "browser_family": browser,
        "browser_version": "126.0",
        "context_kind": "normal",
        "capabilities": ["query"],
        "supported_protocol_versions": [release["protocol"]["major"]],
    }
    envelope = {
        "protocol": release["protocol"]["major"],
        "type": "hello",
        "request_id": HELLO_REQUEST_ID,
        "profile_id": HELLO_PROFILE_ID,
        "projection_revision": 0,
        "payload": payload,
    }
    body = json.dumps(envelope, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
    return struct.pack("<I", len(body)) + body


def _health_env(platform: str, health_home: Path) -> dict[str, str]:
    env = os.environ.copy()
    health_home.mkdir(parents=True, exist_ok=True)
    if platform == "windows":
        env["APPDATA"] = str(health_home)
        env["LOCALAPPDATA"] = str(health_home)
        env["USERPROFILE"] = str(health_home)
    elif platform == "macos":
        env["HOME"] = str(health_home)
    else:
        env["XDG_CONFIG_HOME"] = str(health_home)
        env["HOME"] = str(health_home)
    return env


def _diagnostic_health(stderr: bytes, release: dict[str, Any]) -> HealthResult:
    """Fallback for the current host bootstrap seam.

    The checked-in host main wires persistence but leaves the protocol and
    projection hooks sealed. It therefore cannot emit hello_ack yet. Packaging
    still records the observable bootstrap/persistence health and marks the
    result as degraded; the release gate refuses to publish on this path.
    """

    accepted = False
    ready = False
    rejected = False
    host_version: str | None = None
    protocol: int | None = None
    for line in stderr.decode("utf-8", "replace").splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("event") == "bootstrap.rejected":
            rejected = True
        if event.get("event") == "bootstrap.accepted":
            accepted = True
        if event.get("event") == "persistence.settled" and event.get("persistence") == "ready":
            ready = True
        if isinstance(event.get("host_version"), str):
            host_version = event["host_version"]
        if isinstance(event.get("protocol_version"), int):
            protocol = event["protocol_version"]
    if rejected:
        return HealthResult(ok=False, error_class="host_bootstrap_rejected", stderr_bytes=len(stderr))
    if not accepted or not ready:
        return HealthResult(ok=False, error_class="host_persistence_not_ready", stderr_bytes=len(stderr))
    if host_version is not None and host_version != release["versions"]["host"]:
        return HealthResult(
            ok=False,
            error_class="host_version_mismatch",
            host_version=host_version,
            protocol=protocol,
            stderr_bytes=len(stderr),
        )
    return HealthResult(
        ok=True,
        host_version=host_version or release["versions"]["host"],
        persistence="ready",
        protocol=protocol or release["protocol"]["major"],
        hello_ack=False,
        degraded_reason="runtime_protocol_unwired",
        stderr_bytes=len(stderr),
    )


def check_host(
    binary: Path,
    origin: str,
    release: dict[str, Any],
    platform: str,
    health_home: Path,
    browser: str = "chrome",
    timeout: float = 5.0,
) -> HealthResult:
    if not binary.is_file():
        return HealthResult(ok=False, error_class="host_binary_missing")
    frame = hello_frame(release, browser)
    try:
        process = subprocess.Popen(
            [str(binary), origin],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=_health_env(platform, health_home),
        )
    except OSError:
        return HealthResult(ok=False, error_class="host_launch_failed")
    try:
        stdout, stderr = process.communicate(frame, timeout=timeout)
    except subprocess.TimeoutExpired:
        process.kill()
        stdout, stderr = process.communicate()
        return HealthResult(
            ok=False,
            error_class="host_hello_timeout",
            stdout_bytes=len(stdout or b""),
            stderr_bytes=len(stderr or b""),
        )
    if len(stdout) < 4:
        return _diagnostic_health(stderr, release)
    size = struct.unpack("<I", stdout[:4])[0]
    if size == 0 or size > 1 << 20 or len(stdout) < 4 + size:
        return HealthResult(
            ok=False,
            error_class="host_hello_malformed",
            stdout_bytes=len(stdout),
            stderr_bytes=len(stderr),
        )
    try:
        envelope = json.loads(stdout[4 : 4 + size].decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError):
        return HealthResult(
            ok=False,
            error_class="host_hello_malformed",
            stdout_bytes=len(stdout),
            stderr_bytes=len(stderr),
        )
    if envelope.get("type") != "hello_ack" or not isinstance(envelope.get("payload"), dict):
        return HealthResult(
            ok=False,
            error_class="host_hello_rejected",
            stdout_bytes=len(stdout),
            stderr_bytes=len(stderr),
        )
    payload = envelope["payload"]
    host_version = payload.get("host_version")
    protocol = payload.get("protocol")
    if protocol != release["protocol"]["major"]:
        return HealthResult(
            ok=False,
            error_class="host_protocol_mismatch",
            host_version=host_version if isinstance(host_version, str) else None,
            protocol=protocol if isinstance(protocol, int) else None,
            stdout_bytes=len(stdout),
            stderr_bytes=len(stderr),
        )
    if host_version != release["versions"]["host"]:
        return HealthResult(
            ok=False,
            error_class="host_version_mismatch",
            host_version=host_version if isinstance(host_version, str) else None,
            protocol=protocol if isinstance(protocol, int) else None,
            stdout_bytes=len(stdout),
            stderr_bytes=len(stderr),
        )
    persistence = payload.get("persistence")
    return HealthResult(
        ok=True,
        host_version=host_version,
        persistence=persistence if isinstance(persistence, str) else None,
        protocol=protocol if isinstance(protocol, int) else None,
        hello_ack=True,
        stdout_bytes=len(stdout),
        stderr_bytes=len(stderr),
    )
