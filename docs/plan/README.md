# Implementation & Planning

This directory contains historical implementation plans, architectural refactor archives, and evolution records for the InfoBoard project.

---

## 1. Architectural Evolution

- **Current Architecture (Active)**: **Pure Manifest V3 Tab Search**
  - All functionality runs directly inside the Chromium extension environment.
  - Zero native messaging daemons, zero external Go hosts, zero SQLite dependencies.
  - Sub-millisecond in-memory lexical scoring, MRU tracking via `chrome.storage.session`, and quick-toggle tab switching.
  - See [`../architecture/system-overview.md`](../architecture/system-overview.md) and [`../product/roadmap.md`](../product/roadmap.md).

- **Historical Architecture (Archived)**:
  - [`refactor/`](refactor/README.md): Initial refactor proposals and formal boundaries.
  - [`implementation/`](implementation/README.md): The original 20-phase implementation plan (IP-01 through IP-20) covering the deprecated Go Native Messaging host and SQLite projection pipeline. These documents are preserved for historical reference and architectural decision traceability.
