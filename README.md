# InfoBoard

InfoBoard is a keyboard-first Chromium Manifest V3 extension for searching and switching open tabs across windows. It runs entirely in the browser extension runtime with no native host, background daemon, or remote service.

## Features

- Search open tabs by title, domain, word boundary, acronym, and fuzzy subsequence.
- Two-phase typo fallback (Damerau-Levenshtein distance <= 1) when an exact/prefix/subsequence pass returns no results and the query is long enough.
- Context filters: `@audio` (`@a`), `@pinned` (`@p`), `@group` (`@g`), `@win` (`@w`), with autocomplete in the popup.
- MRU-aware ranking using `chrome.storage.session` and tab activity listeners.
- Quick switch: opening the popup pre-selects the previous tab for an Alt+Tab-style toggle.
- Direct tab close from the popup (`Alt+D` or `Delete` when input is empty), plus mouse close button.
- Localized UI (`en`, `vi`) via `chrome.i18n`.

## Install

### Chrome Web Store

Install from the Chrome Web Store listing when available. End users do not need Node.js.

### Load unpacked (development)

```bash
git clone https://github.com/ryantr-statinops/InfoBoard.git
cd InfoBoard
npm ci
npm run build:extension
```

1. Open `chrome://extensions` (or `edge://extensions`).
2. Enable **Developer mode**.
3. Click **Load unpacked** and select the `extension/` directory.
4. Press `Alt+W` (Windows/Linux) or `Command+Shift+K` (macOS).

## Usage

| Action | Windows/Linux | macOS |
|---|---|---|
| Open/toggle popup | `Alt+W` | `Command+Shift+K` |
| Navigate results | `ArrowUp` / `ArrowDown`, `Ctrl+P` / `Ctrl+N` | Same |
| Jump | `Home` / `End` | Same |
| Activate selected tab | `Enter` | `Return` |
| Close selected tab | `Alt+D`, or `Delete` when query is empty | Same |
| Dismiss popup | `Esc` | `Esc` |

Filter examples: `@audio lofi`, `@pinned git`, `@win`.

## Architecture

- `extension/manifest.json`: MV3 manifest, permissions, command, popup, service worker.
- `extension/src/background/service-worker.ts`: listens to tab/window activation and removal events.
- `extension/src/background/mru-manager.ts`: maintains an MRU ring buffer in `chrome.storage.session`.
- `extension/src/search/types.ts`: tab model, domain extraction, normalization.
- `extension/src/search/scorer.ts`: context filter parsing, lexical/subsequence scoring, typo fallback, MRU boost.
- `extension/src/search/surface-entrypoint.ts`: popup rendering, keyboard/filter interactions, tab activation/close.
- `extension/src/search/surface-shell.html` + `surface-shell.css`: popup markup and styles.
- `extension/_locales/`: English and Vietnamese message catalogs.
- `scripts/`: build asset copy and release packaging scripts.
- `tests/`: Node built-in test runner coverage for manifest/bundle, MRU, scorer, surface, and types.

## Permissions

| Permission | Purpose |
|---|---|
| `tabs` | Read open tab metadata needed for local search and activation. |
| `tabGroups` | Read tab group metadata for group-aware display/filtering. |
| `storage` | Store ephemeral MRU tab IDs in `chrome.storage.session`. |

InfoBoard does not request host permissions, content script access, browsing history, or native messaging. See [PRIVACY.md](PRIVACY.md).

## Development

Node.js is a build/test/packaging toolchain only; it is not part of the extension runtime.

```bash
npm ci
npm run build:extension
npm test
npm run package
```

- `npm run build:extension` compiles TypeScript and copies static extension assets into `extension/dist/`.
- `npm test` rebuilds and runs the Node test suite under `tests/`.
- `npm run package` creates `release/infoboard-v<version>.zip` from `extension/manifest.json` version.

CI runs build, test, and package on pushes/PRs to `main` and `dev` (`.github/workflows/ci.yml`). Tag pushes matching `v*` publish a GitHub Release with the packaged zip (`.github/workflows/release.yml`).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE)
