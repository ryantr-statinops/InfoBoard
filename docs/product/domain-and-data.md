# Domain and data

## Tab projection

The searchable domain record is a profile-scoped projection of an eligible open tab. It may contain:

- stable browser tab and window identity;
- title and display URL/domain;
- normalized searchable title, URL, and domain fields;
- window and tab-group labels;
- pinned state;
- current activation metadata required for bounded contextual ranking;
- projection revision and synchronization sequence.

The display form is separate from the searchable form. URL query strings that may contain secrets are redacted in the UI by default even when the URL contributes searchable tokens.

## Lifecycle

Tab creation, title/URL update, move, group change, pin change, activation, window change, and removal update the projection. A full snapshot is authoritative after startup, reconnect, missed events, or sequence mismatch. Deltas are an optimization and never override a newer snapshot.

## Ownership

| Data | Owner | Retention |
| --- | --- | --- |
| Current open-tab projection | Extension/host memory | Rebuildable; not durable source of truth |
| Configuration | SQLite | Until profile reset or uninstall |
| Installation state | SQLite/package layer | Until uninstall or cleanup |
| Activation metadata | SQLite | Bounded retention defined by lifecycle policy |
| Browser tabs and history | Browser | Not owned or deleted by InfoBoard |

## Privacy and trust

The extension has browser authority; the host has local computation authority. The host receives only the eligible projection needed for local search. No cookies, page content, whole history, cloud service, or network request enters the product contract.

Incognito/private profiles remain isolated from normal profiles and follow browser permission rules. A new data source requires a product-value explanation, privacy impact, retention rule, removal behavior, decision record, and acceptance signal.

See the [canonical domain and privacy contract](../plan/refactor/domain-and-privacy.md).
