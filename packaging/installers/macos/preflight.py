from pathlib import Path
import sys

for candidate in Path(__file__).resolve().parents:
    if (candidate / "packaging" / "release" / "lifecycle.py").is_file():
        sys.path.insert(0, str(candidate))
        break

from packaging.release.lifecycle import main  # noqa: E402

if __name__ == "__main__":
    raise SystemExit(main(platform="macos", operation="preflight"))
