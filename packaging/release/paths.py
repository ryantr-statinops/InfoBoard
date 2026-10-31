"""Platform-specific installation paths.

The host process currently stores SQLite under the user's config directory
(`os.UserConfigDir()/infoboard`). Packaging keeps that path unchanged and uses
a separate data root for the host binary and registration artifacts. Tests may
override every root so the same lifecycle contract can run in a sandbox.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping

HOST_NAME = "com.infoboard.host"
HOST_BINARY_NAME = "infoboard-host"
HOST_BINARY_WINDOWS = "infoboard-host.exe"


@dataclass(frozen=True)
class PlatformPaths:
    platform: str
    data_root: Path
    config_root: Path
    browser_roots: Mapping[str, Path]
    host_binary: Path
    host_manifests: Mapping[str, Path]
    registry_keys: Mapping[str, str]

    def browser_root(self, browser: str) -> Path:
        return self.browser_roots[browser]

    def host_manifest(self, browser: str) -> Path:
        return self.host_manifests[browser]


def _env(env: Mapping[str, str] | None) -> Mapping[str, str]:
    return os.environ if env is None else env


def _home(env: Mapping[str, str]) -> Path:
    value = env.get("HOME") or env.get("USERPROFILE") or str(Path.home())
    return Path(value)


def _default_data_root(platform: str, env: Mapping[str, str]) -> Path:
    if platform == "windows":
        base = env.get("LOCALAPPDATA")
        if not base:
            raise ValueError("LOCALAPPDATA is required for Windows packaging")
        return Path(base) / "InfoBoard"
    if platform == "macos":
        return _home(env) / "Library" / "Application Support" / "InfoBoard"
    base = env.get("XDG_DATA_HOME")
    if base:
        return Path(base) / "InfoBoard"
    return _home(env) / ".local" / "share" / "InfoBoard"


def _default_config_root(platform: str, env: Mapping[str, str]) -> Path:
    if platform == "windows":
        base = env.get("APPDATA")
        if not base:
            raise ValueError("APPDATA is required for Windows packaging")
        return Path(base) / "infoboard"
    if platform == "macos":
        return _home(env) / "Library" / "Application Support" / "infoboard"
    base = env.get("XDG_CONFIG_HOME")
    if base:
        return Path(base) / "infoboard"
    return _home(env) / ".config" / "infoboard"


def _default_browser_roots(platform: str, data_root: Path, env: Mapping[str, str]) -> dict[str, Path]:
    if platform == "windows":
        base = data_root / "native-messaging"
        return {"chrome": base / "chrome", "edge": base / "edge"}
    if platform == "macos":
        base = _home(env) / "Library" / "Application Support"
        return {
            "chrome": base / "Google" / "Chrome" / "NativeMessagingHosts",
            "edge": base / "Microsoft Edge" / "NativeMessagingHosts",
        }
    base = Path(env.get("XDG_CONFIG_HOME") or (_home(env) / ".config"))
    return {
        "chrome": base / "google-chrome" / "NativeMessagingHosts",
        "edge": base / "microsoft-edge" / "NativeMessagingHosts",
    }


def canonical_binary_path(operating_system: str) -> str:
    if operating_system == "windows":
        return r"C:\Users\user\AppData\Local\InfoBoard\host\infoboard-host.exe"
    if operating_system == "macos":
        return "/Users/user/Library/Application Support/InfoBoard/host/infoboard-host"
    return "/home/user/.local/share/InfoBoard/host/infoboard-host"


def resolve_paths(
    platform: str,
    env: Mapping[str, str] | None = None,
    root_override: Path | None = None,
    config_override: Path | None = None,
    browser_overrides: Mapping[str, Path] | None = None,
) -> PlatformPaths:
    values = _env(env)
    if root_override is not None:
        data_root = root_override / "data"
        config_root = config_override or (root_override / "config")
        browser_roots = dict(browser_overrides or {
            "chrome": root_override / "browsers" / "chrome",
            "edge": root_override / "browsers" / "edge",
        })
    else:
        data_root = _default_data_root(platform, values)
        config_root = config_override or _default_config_root(platform, values)
        browser_roots = dict(browser_overrides or _default_browser_roots(platform, data_root, values))
    binary_name = HOST_BINARY_WINDOWS if platform == "windows" else HOST_BINARY_NAME
    host_binary = data_root / "host" / binary_name
    host_manifests = {browser: root / f"{HOST_NAME}.json" for browser, root in browser_roots.items()}
    registry_keys = {}
    if platform == "windows":
        registry_keys = {
            "chrome": rf"HKCU\Software\Google\Chrome\NativeMessagingHosts\{HOST_NAME}",
            "edge": rf"HKCU\Software\Microsoft\Edge\NativeMessagingHosts\{HOST_NAME}",
        }
    return PlatformPaths(
        platform=platform,
        data_root=data_root,
        config_root=config_root,
        browser_roots=browser_roots,
        host_binary=host_binary,
        host_manifests=host_manifests,
        registry_keys=registry_keys,
    )
