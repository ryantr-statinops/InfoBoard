# InfoBoard v0.1.0 Known Limitations

This document distinguishes documented product contract boundaries and non-gating environmental constraints from defects.
**Zero in-scope functional or security defects remain unresolved.**

## 1. Environmental Constraints (Approved Exceptions)

- **Chrome Action-Popup Creation Duration (EXC-001):**
  - *Context:* The initial window creation time for Chrome extension action popups is controlled by the Chromium desktop window manager / OS compositor (measured between 324 ms and 507 ms).
  - *Mitigation:* Extension-controlled surface focus after window creation is 4.8 ms, well within the 100 ms target for NFR-001. Documented and accepted in IP-13.
- **Physical Reference Workstations for Linux and macOS (EXC-002):**
  - *Context:* Physical reference machines for `R-LINUX` and `R-MACOS` were not attached to the local developer workstation during candidate verification.
  - *Mitigation:* Cells C01–C04 are recorded as compliant `BLOCKED` with explicit hardware unavailability reasons per the IP-19 specification. Cells C05 and C06 passed 100% on physical Windows hardware (`R-WINDOWS`).

## 2. Product Boundary Exclusions (By Design)

- **No Remote Network Access:** InfoBoard never transmits browsing data, tab metadata, or telemetry off-machine.
- **No History or Page Content Reading:** InfoBoard observes open tab titles and URLs in memory for the current profile only; it does not read page DOM, cookies, local storage, or past browsing history.
- **Strict Profile Isolation:** Data is strictly scoped to the active browser profile `(browser_family, profile_id)`. Tabs from other profiles or incognito contexts are never exposed or mixed.
