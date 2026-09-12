# Design — Information architecture

## Navigation model

```text
InfoBoard
├── Dashboard
├── All items
├── Collections
├── Inbox
├── Active
├── Archived
└── Settings / maintenance (advanced)
```

The default navigation exposes capture, organization, and retrieval without
requiring advanced setup. Settings and maintenance remain available through an
explicit advanced path.

## Dashboard structure

```text
┌──────────────┬──────────────────────────────────────────────┐
│ Navigation   │ Search                              + Add   │
│              ├──────────────────────────────────────────────┤
│              │ Collection · Source · Status · Date         │
│              ├──────────────────────────────────────────────┤
│              │ KPI and processing status                   │
│              ├────────────────────────┬─────────────────────┤
│              │ Recent/item list       │ Analytics/insights  │
└──────────────┴────────────────────────┴─────────────────────┘
                         Item → detail panel
```

## Surface ownership by milestone

| Surface | Milestone | Mode | Notes |
| --- | --- | --- | --- |
| Dashboard shell, add, list, and detail | M1 | Core | Primary user workspace |
| Collections, notes, and status views | M1 | Core | Organization and context |
| Keyword search and retrieval filters | M3 | Core | Must work without full mode |
| KPI and analytics | M3 | Core | SQLite fallback remains available |
| Semantic controls, related items, and clusters | M3 | Optional full | Visible only after explicit setup |
| Health, backup, rebuild, and maintenance | M4 | Core/advanced | Operational actions are advanced |

## URL and filter state

Filter/search context includes `q`, collection, source, status, and date range.
Opening or closing a detail panel preserves that context. The list excludes
archived items by default and offers an explicit all-time option.

## Information rules

- An item can belong to multiple collections, so collection totals can exceed
  the number of unique items.
- Application-wide job counts must be labelled separately when filters apply
  only to items or analytics.
- Advanced maintenance must not interrupt the default capture/retrieve flow.
- Full-mode controls must not appear as available until full mode is explicitly
  configured.
