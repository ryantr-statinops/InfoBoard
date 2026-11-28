# Contributing to InfoBoard

## Code of Conduct

All contributors and maintainers are expected to follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Development Setup

Prerequisites: Node.js 20+ (or compatible runtime for local development) and npm.

```bash
git clone https://github.com/ryantr-statinops/InfoBoard.git
cd InfoBoard
npm ci
npm run build:extension
```

Load the extension unpacked from `extension/` in `chrome://extensions` with Developer mode enabled.

Node.js is used for build, test, and packaging only. The extension runtime is Chromium Manifest V3.

## Commands

| Command | Purpose |
|---|---|
| `npm run clean` | Remove `extension/dist`. |
| `npm run build:extension` | Compile TypeScript and copy static assets into `extension/dist/`. |
| `npm test` | Rebuild and run the Node built-in test suite in `tests/`. |
| `npm run package` | Build and create `release/infoboard-v<version>.zip`. |

## Standards

- Keep the extension pure MV3: no runtime npm dependencies, no native host, no remote services required for core behavior.
- Keep changes compatible with `extension/tsconfig.json` strict TypeScript settings.
- Add or update tests under `tests/` for behavior changes.
- Preserve bundle/manifest constraints enforced by tests and CI.

## Git Workflow

- `main`: stable release branch.
- `dev`: active integration branch; target PRs to `dev` unless maintainers request otherwise.
- Use focused branches, for example `feat/...`, `fix/...`, `docs/...`.
- Prefer small, atomic commits with Conventional Commit messages.

## Pull Requests

Before opening a PR:

1. Run `npm test`.
2. For release-impacting changes, run `npm run package`.
3. Include a clear summary, motivation, and verification evidence.

CI validates build, tests, and packaging on pushes/PRs to `main` and `dev` (see `.github/workflows/ci.yml`).
