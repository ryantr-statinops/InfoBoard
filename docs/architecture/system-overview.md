# Architecture: System Overview

InfoBoard is engineered as a **Pure Manifest V3 Chrome Extension**. It operates 100% inside Chromium's V8 engine and browser runtime sandbox, requiring no external companion native applications, no background daemons, no SQLite persistence, and no cloud connections.

---

## 1. System Topology

```mermaid
flowchart TD
    subgraph Chromium_Browser["Chromium Browser Runtime (Chrome / Edge / Brave)"]
        subgraph Action_Popup["Action Popup Surface (UI)"]
            A1["Input Search Bar (#query)"]
            A2["Live Tab Results (#tab-results)"]
            A3["Shortcuts Footer Basin"]
            A4["500ms Waterfall Motion Engine"]
        end

        subgraph In_Memory_Core["In-Memory Execution Engine (< 1ms)"]
            B1["Tab Normalizer & Domain Extractor"]
            B2["Multi-Tier Scorer (scorer.ts)"]
            B3["Direct Tab Controller (chrome.tabs.remove / update)"]
            B4["Bilingual i18n Bridge (chrome.i18n)"]
        end

        subgraph Background_Worker["Background Service Worker (Event-Driven)"]
            C1["chrome.tabs.onActivated"]
            C2["chrome.tabs.onRemoved"]
            C3["chrome.windows.onFocusChanged"]
            C4["MRU Ring Buffer Manager (max 100 tabs)"]
        end

        subgraph Browser_Storage["In-Memory Ephemeral Storage"]
            D1["chrome.storage.session ('mru_tab_ids')"]
        end
    end

    A1 -->|Query string| B2
    B2 -->|Ranked tab array| A2
    Action_Popup -->|Fetch MRU & tabs| D1
    Action_Popup -->|Direct close / switch| B3
    B3 -->|API Call| Chromium_Browser
    C1 --> C4
    C2 --> C4
    C3 --> C4
    C4 -->|Update ring buffer| D1
```

---

## 2. Component Ownership & Boundaries

### A. Focused Search Surface (`extension/src/search/`)
- **`surface-shell.html` & `surface-shell.css`**:
  - Encapsulates the 480x480px popup viewport with responsive dark/light theme variables.
  - Houses the custom 6px rounded scrollbar and vector SVG icons.
  - Implements the 500ms bidirectional waterfall animations (entrance cascade and reverse exit).
- **`surface-entrypoint.ts`**:
  - Initializes search input autofocus, ARIA accessibility attributes (`role="listbox"`, `role="option"`).
  - Handles keyboard shortcuts: arrow navigation (`↑`/`↓`), quick Enter selection, direct close (`Alt + D` / `Delete`), and dismiss (`Esc`).
  - Pre-selects the previous tab (index 1) on empty query for an instant `Alt+Tab` quick-switch flow.

### B. In-Memory Scorer & Tab Indexer (`extension/src/search/scorer.ts`)
- Evaluates queries against live tab snapshots entirely in RAM.
- Completes scoring and deterministic ranking of hundreds of tabs in less than **1 millisecond**.

### C. Background MRU Ring Buffer (`extension/src/background/`)
- **`service-worker.ts`**:
  - Event-driven service worker that sleeps when idle.
  - Wakes up on `tabs.onActivated`, `tabs.onRemoved`, and `windows.onFocusChanged` to update MRU recency in `chrome.storage.session`.
- **`mru-manager.ts`**:
  - Maintains a bounded ring buffer (capped at `MAX_MRU_SIZE = 100`) preventing memory bloat.

---

## 3. Ephemeral Execution Model

Manifest V3 terminates service workers after ~30 seconds of inactivity. InfoBoard is architected specifically to handle this:
1. **Zero Cold-Start Latency**: Popup loads tabs directly via `chrome.tabs.query({})` and reads `chrome.storage.session` in parallel (`Promise.all`), requiring no message exchange with a sleeping service worker.
2. **Session Storage Persistence**: `chrome.storage.session` lives in RAM across service worker sleep/wake cycles and automatically clears when the browser shuts down.

---

## 4. Build-Time vs. Runtime Boundary

A core architectural guarantee of InfoBoard is the strict isolation between the **development toolchain (build-time)** and the **client execution sandbox (runtime)**:

```mermaid
flowchart LR
    subgraph Build_Stage["Build-Time Pipeline (Node.js / Bun)"]
        S1["TypeScript Source (src/**/*.ts)"]
        S2["Static Assets (HTML, CSS, Icons, _locales)"]
        T1["TypeScript Compiler (tsc)"]
        T2["Asset Distributor (copy-extension-assets.mjs)"]
        S1 --> T1
        S2 --> T2
    end

    subgraph Production_Artifacts["Distribution Bundle (extension/dist)"]
        P1["Pure ES2022 JavaScript (< 50KB total)"]
        P2["Structured Assets & Dictionaries"]
        T1 --> P1
        T2 --> P2
    end

    subgraph Runtime_Sandbox["Browser Runtime Sandbox (Zero Dependencies)"]
        B1["Chromium V8 Engine"]
        B2["Supported: Google Chrome, Brave, Edge, Arc"]
        P1 --> B1
        P2 --> B1
        B1 --> B2
    end
```

### Architectural Guarantees:
- **Zero Client Dependencies**: End-users installing InfoBoard run solely within Chromium's sandboxed V8 engine. No background daemons, local web servers, Python, or Node.js processes exist on the client machine.
- **Development Toolchain Role**: Node.js (or Bun) serves exclusively as an offline build harness—executing `tsc` type-checking, running the 34-test in-memory validation suite via `node --test`, and assembling production distribution ZIPs.
