# Runtime architecture direction

## Proposed ownership

```text
Chromium extension -> browser events, shortcut, UI
Go runtime         -> hot index, ranking, IPC, health
SQLite             -> optional durable metadata and configuration
ChromaDB           -> optional asynchronous semantic index
RocksDB            -> optional embedding/cache layer
DuckDB             -> not needed for the initial tab-search path
```

SQLite remains the source of truth only for data that the new product explicitly decides to persist. The Go in-memory index is a rebuildable runtime projection.

## Hot path

```text
Shortcut
-> search UI receives focus
-> query sent to warm Go runtime
-> in-memory lexical/fuzzy search
-> deterministic ranking
-> selected tab activation
```

ChromaDB must not be called for every keystroke. Embeddings, if accepted later, run asynchronously and provide a semantic fallback or related-tab feature.

## Process options

### Warm Go daemon

The daemon starts at login or browser startup, keeps the current index in memory, and accepts local requests.

- Best latency and predictable behavior.
- Requires install, autostart, health, crash recovery, and secure local IPC.
- A loopback HTTP transport is easy to debug; it requires an authentication token and strict loopback binding.

### Native Messaging host

The extension starts a registered Go host and communicates through framed stdin/stdout messages.

- No local port exposed.
- More installation and packaging work.
- Startup and service-worker reconnect behavior must be measured.
- A Native Messaging connection is IPC; it is not automatically a permanently warm daemon.

### Current recommendation

Use a warm Go daemon for the latency-first prototype. Keep Native Messaging as the alternative if local-port security or distribution requirements dominate. Do not spawn a fresh Go process for every shortcut.

## Performance targets

Initial proposed targets, to be confirmed by benchmark:

- Shortcut to focused search UI: under 100 ms.
- Query update to result update: under 50 ms for the open-tab dataset.
- Enter to selected-tab activation: under 100 ms.
- Cold-start latency measured separately from hot-path latency.

## Failure behavior

- Go runtime unavailable: extension shows a recoverable unavailable state and can rebuild/reconnect.
- Stale index: extension sends a full tab snapshot and Go rebuilds incrementally.
- Closed result before activation: remove it and select the next valid result.
- Chroma unavailable: normal lexical search remains fully available.
