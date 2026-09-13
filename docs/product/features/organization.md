# Feature specification: organization

## Outcome

The user can add durable personal context to a bookmark without coupling that context to capture or indexing lifecycle.

## Model

- **Collection:** a named project or topic; a bookmark may belong to many collections.
- **Tag:** a lightweight normalized label; a bookmark may have many tags.
- **Note:** user-authored context attached to one bookmark.
- **Status:** one of `inbox`, `active`, or `archived`.

## Behavior

- New bookmarks start in `inbox`.
- Collection deletion removes memberships but never bookmarks.
- Tag deletion removes assignments but never bookmarks.
- Notes are created, edited, and deleted independently of snapshots.
- Archive hides a bookmark from the default list without deleting it.
- Soft delete immediately excludes a bookmark from lists, public detail, search, related results, and analytics.
- Recapture, retry, reindex, provider changes, and derived-store rebuild do not mutate organization data.

## Acceptance

- Collection and tag many-to-many behavior survives restart.
- Notes and status survive snapshot version changes and complete rebuilds.
- Deleting a collection or tag preserves every bookmark.
- Soft-deleted data does not leak through any user-facing retrieval path.
