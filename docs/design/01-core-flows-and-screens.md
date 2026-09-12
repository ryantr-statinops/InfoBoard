# Design — Core flows and screens

## Screen inventory

| Screen/surface | Primary purpose |
| --- | --- |
| Dashboard | Overview, filters, recent items, KPIs, and analytics |
| Add dialog/form | Create a text, URL, or file item |
| Item list | Browse filtered/search results |
| Detail panel | Read content/source; edit metadata, notes, collections, and status |
| Search results | Excerpt, score/mode, and item opening |
| Collections | Create, rename, delete, and filter collections |
| Settings/maintenance | Full-mode config, health, and backup/rebuild for advanced users |

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
