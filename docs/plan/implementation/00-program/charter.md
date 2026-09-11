# 00 — Program charter

**Status:** `ready`
**Milestone:** program-wide
**Owner:** project maintainer
**Canonical dependencies:** [Internal PRD](../../../product/internal-prd/README.md), [Architecture](../../../architecture/README.md)

## Objective

InfoBoard is an Advanced Bookmark Manager for one local user: it stores URLs and content snapshots, accepts text/files, organizes information with notes/collections, and supports full-text or semantic retrieval. SQLite is the system of record; indexes, caches, and analytics are rebuildable derived stores.

## Outcomes

- Users can import, filter, open, edit, annotate, organize, and delete items.
- The dashboard is responsive, exposes indexing state, and survives restart without data loss.
- Keyword search works independently; semantic/hybrid search is an advanced capability with safe fallback.
- Backup, restore, migration, tests, and runbooks support long-term local use.
- Post-MVP directions remain explicit without expanding MVP scope.

## Scope by phase

### MVP

Python 3.12, FastAPI/Uvicorn, Jinja2 + HTMX, SQLite/FTS5, text/Markdown/TXT/PDF/public URL ingestion, sequential worker, optional semantic/cache/analytics stores, backup, and quality gates.

### Post-MVP

Browser extension, browser portability, migration assistant, cloud sync, collaboration, and AI/provider ecosystem. These require the discovery packages and gates in `50-post-mvp/`.

## Invariants

1. SQLite is authoritative for items, content, relations, notes, and recoverable job state.
2. Derived-store failure never removes readable canonical data or keyword search.
3. Models and network calls are never loaded silently by a dashboard request.
4. Every input has limits, normalization, and safe rendering.
5. Every feature uses small commits, verification, push, and user review.

## Program definition of done

All requirements in [traceability](../90-governance/requirement-traceability.md) are verified or explicitly waived; test/lint/benchmark pass; README/runbooks match actual behavior; backup/restore is verified; and the MVP release is tagged from a clean main branch.
