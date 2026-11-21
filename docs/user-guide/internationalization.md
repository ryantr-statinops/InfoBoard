# User Guide: Internationalization (i18n)

InfoBoard provides bilingual localization for **English** and **Vietnamese (Tiếng Việt)** through standard Chrome WebExtension i18n APIs (`chrome.i18n`).

---

## 1. Supported Languages

| Locale Code | Language | Default Status |
| :---: | :--- | :--- |
| `en` | English | Default fallback locale (`"default_locale": "en"`) |
| `vi` | Tiếng Việt (Vietnamese) | Active when browser language is set to Vietnamese |

---

## 2. Localized Surface Elements

All user-facing strings are dynamically localized:

| UI Key | English (`en`) | Tiếng Việt (`vi`) |
| :--- | :--- | :--- |
| `searchPlaceholder` | Search open tabs by title or domain... | Tìm kiếm tab theo tiêu đề hoặc tên miền... |
| `emptyState` | No matching tabs found | Không tìm thấy tab phù hợp |
| `shortcutNavigate` | Navigate | Di chuyển |
| `shortcutSwitchTab` | Switch tab | Chuyển tab |
| `shortcutCloseTab` | Close tab | Đóng tab |
| `shortcutClose` | Close | Đóng |
| `badgeCurrent` | Current | Hiện tại |
| `badgePinned` | Pinned | Đã ghim |
| `badgeAudio` | Audio | Âm thanh |
| `badgeMuted` | Muted | Tắt tiếng |

---

## 3. Contributing New Translations

To add a new language (e.g. Japanese, French, Spanish):
1. Create a new directory under `extension/_locales/<locale_code>/` (e.g. `extension/_locales/ja/`).
2. Copy `extension/_locales/en/messages.json` into the new folder.
3. Translate all values in the `"message"` fields while keeping keys unchanged.
4. Update `scripts/copy-extension-assets.mjs` to ensure the new locale directory is copied into `dist/_locales/`.
5. Open a Pull Request!
