"""Windows HKCU registration with a JSON fixture adapter."""

from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Protocol


class RegistryError(RuntimeError):
    pass


class Registry(Protocol):
    def get(self, key: str) -> str | None: ...
    def set(self, key: str, value: str) -> None: ...
    def delete(self, key: str) -> None: ...
    def acl_ok(self, key: str) -> bool: ...


class JsonRegistry:
    """Fixture adapter; production never uses this for registration."""

    def __init__(self, path: Path) -> None:
        self.path = path
        self.path.parent.mkdir(parents=True, exist_ok=True)

    def _load(self) -> dict[str, str]:
        if not self.path.is_file():
            return {}
        return json.loads(self.path.read_text(encoding="utf-8"))

    def _save(self, values: dict[str, str]) -> None:
        self.path.write_text(json.dumps(values, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    def get(self, key: str) -> str | None:
        return self._load().get(key)

    def set(self, key: str, value: str) -> None:
        values = self._load()
        values[key] = value
        self._save(values)

    def delete(self, key: str) -> None:
        values = self._load()
        values.pop(key, None)
        self._save(values)

    def acl_ok(self, key: str) -> bool:
        return True


class WindowsRegistry:
    """HKCU-only registry adapter using the Python standard library."""

    def __init__(self) -> None:
        if os.name != "nt":
            raise RegistryError("Windows registry adapter is only available on Windows")
        import winreg  # noqa: PLC0415

        self.winreg = winreg

    def _split(self, key: str) -> tuple[object, str]:
        if not key.upper().startswith("HKCU\\"):
            raise RegistryError("only HKCU registration keys are allowed")
        path = key.split("\\", 1)[1]
        return self.winreg.HKEY_CURRENT_USER, path

    def get(self, key: str) -> str | None:
        root, path = self._split(key)
        try:
            with self.winreg.OpenKey(root, path, 0, self.winreg.KEY_READ) as handle:
                value, _ = self.winreg.QueryValueEx(handle, "")
                return value
        except FileNotFoundError:
            return None
        except OSError as error:
            raise RegistryError("registry read failed") from error

    def set(self, key: str, value: str) -> None:
        root, path = self._split(key)
        try:
            with self.winreg.CreateKeyEx(root, path, 0, self.winreg.KEY_WRITE) as handle:
                self.winreg.SetValueEx(handle, "", 0, self.winreg.REG_SZ, value)
        except OSError as error:
            raise RegistryError("registry write failed") from error

    def delete(self, key: str) -> None:
        root, path = self._split(key)
        try:
            self.winreg.DeleteKey(root, path)
        except FileNotFoundError:
            return
        except OSError as error:
            raise RegistryError("registry delete failed") from error

    def acl_ok(self, key: str) -> bool:
        # HKCU is a per-user hive. The installer never writes HKLM or a
        # system-wide registration key; the real ACL inspection is exercised by
        # the Windows lifecycle evidence run.
        return self.get(key) is not None


def create_registry(platform: str, registry_file: Path | None = None) -> Registry:
    if registry_file is not None:
        return JsonRegistry(registry_file)
    if platform == "windows":
        return WindowsRegistry()
    return JsonRegistry(Path(os.devnull))
