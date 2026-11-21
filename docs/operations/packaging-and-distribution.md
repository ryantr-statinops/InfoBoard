# Operations: Packaging & Distribution

This document details the build, packaging, and release processes for distributing InfoBoard on the Chrome Web Store (CWS).

---

## 1. Automated Packaging Script

InfoBoard includes a dedicated packaging script [scripts/package-extension.mjs](file:///home/ryantran/Projects/InfoBoard/scripts/package-extension.mjs) that bundles a clean, production-ready zip archive:

```bash
npm run package
```

### What `npm run package` Does:
1. Cleans the previous build output (`npm run clean`).
2. Compiles TypeScript into `extension/dist/`.
3. Copies all static assets (`surface-shell.html`, `surface-shell.css`, `icons/`, `_locales/`) into `extension/dist/`.
4. Reads the version string from [extension/manifest.json](file:///home/ryantran/Projects/InfoBoard/extension/manifest.json) (e.g. `0.1.0`).
5. Assembles `release/infoboard-v0.1.0.zip` containing strictly the necessary extension files:
   - `manifest.json`
   - `dist/` (compiled bundle)
   - `icons/` (16, 32, 48, 128px PNGs)
   - `_locales/` (English and Vietnamese messages)
6. Excludes all development artifacts, TypeScript sources, test suites, node modules, and git files.

---

## 2. Release Acceptance Checklist

Before uploading a release build to the Chrome Web Store:

- [ ] All unit tests pass: `npm test` (34/34 passing in < 100ms).
- [ ] Bundle size check passes: Uncompressed `extension/dist` is well under **50 KB**.
- [ ] Zip archive size is verified: `release/*.zip` is under **100 KB**.
- [ ] Version number is incremented in `extension/manifest.json` following [Semantic Versioning](https://semver.org).
- [ ] Commit history on `dev` is clean and squashed/merged into `main` via PR.

---

## 3. Chrome Web Store Uploading Procedure

1. Log in to the [Chrome Developer Dashboard](https://chrome.google.com/webstore/devconsole).
2. Select the **InfoBoard** item (or click **Add new item**).
3. Upload `release/infoboard-v<version>.zip`.
4. Under **Store Listing**:
   - Verify descriptions in English and Vietnamese.
   - Upload high-resolution screenshots (1280x800) and promotional tiles (440x280).
5. Under **Privacy**:
   - Provide link to [PRIVACY.md](../../PRIVACY.md).
   - Confirm single-purpose policy: *Search and activate open browser tabs*.
6. Submit for review!
