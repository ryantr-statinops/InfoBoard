# Architecture

InfoBoard is designed around a single guiding principle: **extreme lightness, zero latency, and zero friction**.

It replaces the previous complex multi-tier native architecture with a **Pure Manifest V3 Web Extension** running entirely within the Chromium browser process.

---

## Documents in this Section

1. [**System Overview**](system-overview.md): High-level system topology, runtime components, and design invariants.
2. [**Components and Ownership**](components-and-ownership.md): Detailed responsibilities of the popup UI, search scorer, background MRU manager, and Chrome API adapter.
3. [**Runtime Lifecycle**](runtime-lifecycle.md): Service worker sleep/wake management, session storage persistence, and popup lifecycle.
4. [**Data and Failure Boundaries**](data-and-failure-boundaries.md): Failure isolation, privacy boundaries, and scaling characteristics.
