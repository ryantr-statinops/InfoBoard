# InfoBoard — Capability map

| Tầng | Capability | Product boundary |
| --- | --- | --- |
| Core MVP | text/URL/file capture, snapshot, collections, notes, keyword search, local dashboard | Content-oriented, local-first |
| Near-term | bookmark import, browser source adapters, export, migration preview | Portable content và metadata |
| Advanced | browser extension, profile discovery, history migration, conflict resolution | Có consent, preview và compatibility gate |
| Long-term | cookies/session/storage portability, encrypted transfer, cross-device sync | Threat model, feasibility, recovery và privacy review bắt buộc |
| Ecosystem | collaboration, AI/provider integrations | Chỉ sau discovery/decision gate |

## Explicit non-goals

- Âm thầm lấy credential hoặc session token.
- Bypass browser security.
- Cloud upload mặc định.
- Background sync không có consent.
- Gọi browser profile clone là capability portable mặc định.
