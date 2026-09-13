# Product vision

InfoBoard is a private, local-first knowledge bookmark manager for people who save useful web pages and need to find them again with their original context intact.

## Problem

Browser bookmarks preserve a location but usually lose the content, context, and reason a page mattered. Links can change or disappear, folder-only organization becomes difficult to maintain, and exact-word search fails when a user remembers an idea rather than a title.

## Product promise

InfoBoard turns a public URL into a durable personal knowledge item:

```text
Capture -> Organize -> Retrieve
```

- **Capture:** save the URL, source metadata, and a content snapshot.
- **Organize:** add collections, tags, notes, and an inbox/active/archive state.
- **Retrieve:** browse, filter, use local keyword search, or use semantic/hybrid search through a deliberately configured provider.

The bookmark remains useful even when snapshot extraction or a derived database fails. Canonical user data stays local and recoverable.

## Positioning

InfoBoard is more capable than a browser bookmark list and narrower than a general-purpose note-taking or research platform. Its center is the saved web resource and the personal context around it.

## Principles

1. Preserve the bookmark before enriching it.
2. Keep canonical data on the user's machine.
3. Make ownership and external data transfer explicit.
4. Prefer recoverable pipelines over hidden automation.
5. Make keyword and semantic retrieval first-class MVP capabilities.
6. Keep derived stores replaceable and rebuildable.
