# Product contract

## Problem

People lose time when many browser tabs are distributed across windows and groups. Their memory is often a fragment of a title, domain, URL, project name, or topic, while browser navigation favors spatial and recent movement. InfoBoard turns that fragment into a predictable switch to an already-open tab.

## Primary user and outcome

The primary user is a desktop browser user who keeps many tabs open across several windows and wants a fast keyboard-driven way to return to a known tab. The user values speed, predictable ordering, privacy, and recovery when the local runtime is unavailable.

For any eligible currently open tab, the user can:

1. invoke a configurable shortcut;
2. type a remembered fragment;
3. understand why results are ordered;
4. activate the intended tab and window; and
5. return to the previous browser task without losing context.

## Complete capability boundary

- Focused extension-owned search surface.
- Live projection of eligible tabs across windows in the active profile.
- Local lexical search over title, URL, domain, window, group, and bounded state labels.
- Deterministic contextual and recency ranking.
- Keyboard and pointer navigation with safe activation.
- Profile-scoped configuration and runtime status.
- Reconciliation after missed events, service-worker restart, browser restart, or host reconnect.
- Reset, uninstall, diagnostics, installation, update, rollback, and repair behavior.

## Explicit exclusions

The product does not require cookies, browser local storage, whole browsing history, network interception, page-content indexing, cloud indexing, arbitrary page injection, or a loopback listening service. Semantic retrieval and embedding experiments are separate proposals, not hidden dependencies.

## Product boundary

- Browsers: Chrome and Edge desktop.
- Operating systems: Linux, macOS, and Windows.
- Runtime: Manifest V3 extension plus a Go Native Messaging host.
- Persistence: local SQLite for configuration, installation state, and bounded activation metadata.
- Current tab state: rebuildable from browser APIs.
- Privacy: local-only lexical search with explicit permission and retention boundaries.

Read the [canonical product contract](../plan/refactor/idea-brief.md) for binding behavior.
