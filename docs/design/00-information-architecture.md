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
└── Settings / advanced maintenance
```

## Dashboard structure

```text
┌──────────────┬──────────────────────────────────────────────┐
│ Navigation   │ Search                         + Add        │
│              ├──────────────────────────────────────────────┤
│              │ Collection · Source · Status · Date         │
│              ├──────────────────────────────────────────────┤
│              │ KPIs and processing status                  │
│              ├────────────────────────┬─────────────────────┤
│              │ Recent items          │ Analytics/insights  │
└──────────────┴────────────────────────┴─────────────────────┘
                         Item → detail panel
```

## URL state

Filter/search context includes `q`, collection, source, status, and date range. Opening/closing detail preserves the context; the list excludes archived items by default and uses a 30-day range with an all-time option.

## Information rules

- An item may belong to multiple collections, so collection totals may exceed the item total.
- The application-wide job count must be labelled when a filter applies only to items/analytics.
- Advanced maintenance must not interrupt the default capture/retrieve flow.
