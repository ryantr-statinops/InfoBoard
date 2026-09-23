# Decision log

This is a product-facing summary. The binding record is [`../plan/refactor/decisions.md`](../plan/refactor/decisions.md).

## Accepted decisions

- The product is a complete Chrome/Edge desktop extension, not an MVP-only contract.
- The normal search path is local lexical search over the current open-tab projection.
- The runtime boundary is a Go Native Messaging host, not a loopback service.
- The host owns in-memory indexing and bounded local persistence; the extension owns browser authority and activation.
- SQLite stores configuration, installation state, and bounded activation metadata; current tabs are rebuildable.
- Ranking is deterministic, explainable, and versioned; semantic retrieval is not an implicit dependency.
- Privacy excludes cookies, history, page contents, cloud indexing, network interception, and arbitrary page injection.
- Chrome/Edge and Linux/macOS/Windows are product compatibility boundaries.

## Change rule

A change that adds a data source, permission, process boundary, persistence responsibility, ranking model, browser family, operating system, or privacy exception requires an updated product contract, decision record, and acceptance criteria before implementation planning.

## Rejected implicit directions

A faster-looking implementation shortcut is not a product decision. Permanent process residency, semantic indexing, broad browser access, history search, cloud search, and durable current-tab storage remain outside the contract unless explicitly reconsidered with a threat model and measurable acceptance signal.
