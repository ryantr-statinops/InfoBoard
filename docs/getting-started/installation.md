# Installation Guide

InfoBoard can be installed directly from the Chrome Web Store or loaded unpacked from source for development.

> [!NOTE]
> **Zero Runtime Dependencies for End-Users:**
> If you are an end-user installing InfoBoard via the Chrome Web Store or importing a pre-built release ZIP, **you do NOT need Node.js or any external software installed**. InfoBoard executes 100% inside your browser's native V8 JavaScript engine. Node.js is strictly a developer prerequisite used to compile TypeScript and run automated tests.

---

## 1. Install from Chrome Web Store (Recommended for Users)

1. Open the [InfoBoard Chrome Web Store Page](https://chromewebstore.google.com).
2. Click **Add to Chrome** (or your Chromium browser of choice: Brave, Edge, Arc, Vivaldi).
3. Confirm the permission prompt:
   - *Read your open tabs*: Used strictly in-memory to index tab titles and URLs for local search.
4. InfoBoard is now active! Press `Alt + W` (Windows/Linux) or `Command + Shift + K` (macOS) to launch the search interface.

---

## 2. Install Unpacked from Source (Developer Mode)

To run the latest development build directly from source:

### Prerequisites (Developers Only)
- **Chromium-based browser**: Google Chrome, Brave (`brave://extensions`), Microsoft Edge (`edge://extensions`), Arc, or Vivaldi.
- **Node.js**: `v20.x` or higher (`node -v`) *(or alternative JS runtimes like Bun v1.0+)*.
- **npm**: `v10.x` or higher (`npm -v`).

### Build Steps

```bash
# 1. Clone repository
git clone https://github.com/ryantr-statinops/InfoBoard.git
cd InfoBoard

# 2. Install development dependencies
npm ci

# 3. Build extension bundle (compiles TypeScript to extension/dist)
npm run build:extension
```

### Loading in Chromium / Brave / Edge

1. Open your browser and navigate to the extensions page:
   - **Google Chrome**: `chrome://extensions`
   - **Brave**: `brave://extensions`
   - **Microsoft Edge**: `edge://extensions`
2. Toggle on **Developer mode** in the top-right corner.
3. Click the **Load unpacked** button in the top-left toolbar.
4. Select the `extension/` directory inside your cloned InfoBoard workspace:
   ```text
   /path/to/InfoBoard/extension/
   ```
5. InfoBoard will appear in your installed extensions list.

---

## 3. Configuring Keyboard Shortcuts

Chrome assigns default shortcuts automatically, but you can customize them at any time:

1. Navigate to `chrome://extensions/shortcuts`.
2. Scroll to **InfoBoard**.
3. Locate **Open InfoBoard tab search** (default: `Alt + W` or `Command + Shift + K`).
4. Click the pencil icon to assign your preferred shortcut (e.g. `Ctrl + Space` or `Alt + Space`).
