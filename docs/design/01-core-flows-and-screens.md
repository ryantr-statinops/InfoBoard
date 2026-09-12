# Design — Core flows and screens

## Screen inventory

| Screen/surface | Milestone | Primary purpose |
| --- | --- | --- |
| Dashboard | M1; M3 extensions | Shell/list in M1; KPI and analytics surfaces in M3 |
| Add dialog/form | M1 | Create a text, URL, or file item |
| Item list | M1; M3 extensions | Browse items in M1; retrieval filters/search in M3 |
| Detail panel | M1 | Read content/source; edit metadata, notes, collections, and status |
| Keyword search results | M3 core | Excerpt, keyword mode, shared filters, and item opening |
| Semantic/related/cluster surfaces | M3 optional full mode | Semantic score/mode and source-provenance discovery after explicit setup |
| Collections | M1 | Create, rename, delete, and filter collections |
| Settings/maintenance | M4 | Explicit full-mode setup, health, diagnostics, and backup/rebuild for advanced users |

## Primary flow

```mermaid
flowchart LR
    Add[Add Text/URL/File] --> Process[Validate + processing status]
    Process --> List[Dashboard/list]
    List --> Detail[Detail panel]
    Detail --> Organize[Note/collection/status]
    Search[Search + filters] --> Results[Item results]
    Results --> Detail
```

## Interaction rules

- Add creates an item in `inbox` by default; title and collections are optional.
- Detail shows the full snapshot, provenance, current job state, and up to five related items when available.
- Only text sources edit content directly; URL/file sources retain their snapshot.
- Delete always requires confirmation and hides the item immediately after success.
- Clicking a collection/chart opens the list with the corresponding filter.
- Search results are item-level; users do not need to understand chunks.

## Simple and advanced mode boundary

- Simple mode is the default and includes capture, organization, keyword retrieval, item status, and understandable recovery guidance.
- Advanced mode contains full-mode setup, provider/model metadata, diagnostics, backup/rebuild controls, and detailed job steps.
- Full-mode search and discovery controls appear only after successful explicit setup.
- A core-only installation does not show missing-full-mode warnings. If an enabled full-mode component later fails, affected optional surfaces show degraded state and keyword retrieval remains available.
