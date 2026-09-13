# Idea inbox

This is the only entry point for capabilities that are not in the accepted MVP. An idea is not a promise, requirement, or roadmap item.

## Workflow

```text
inbox -> evaluating -> accepted -> specified -> planned
                  \-> rejected
                  \-> parked
```

An idea may move to `accepted` only when its user value, scope impact, privacy boundary, architecture impact, failure behavior, and acceptance signal are understood. Acceptance requires a record in [decisions](decisions.md). The same change must then update the product scope, requirements, feature specification, architecture, quality gate, and roadmap in that order.

## Inbox

| Idea | Status | Question to answer before evaluation |
| --- | --- | --- |
| Browser extension | inbox | Does capture speed justify browser permission and distribution complexity? |
| Browser bookmark import | inbox | Which formats and duplicate semantics can be supported safely? |
| File and document ingestion | inbox | Which user job requires non-URL sources in this product? |
| Cloud synchronization | inbox | What conflict, identity, encryption, and recovery model preserves local ownership? |
| Collaboration and sharing | inbox | Who owns shared data and how is access revoked? |
| AI summaries and chat | inbox | What task improves beyond retrieval, and what content may leave the device? |
| Automatic tag suggestions | inbox | Can suggestions remain optional, explainable, and non-destructive? |
| Native mobile application | inbox | Which offline and sync behavior is required? |
| Authenticated-page capture | inbox | Can credentials and page content be handled without violating the trust model? |

## Proposal template

```markdown
### Idea: <name>

- Status: inbox
- User problem:
- Proposed outcome:
- Why current features are insufficient:
- Data and consent impact:
- Architecture and migration impact:
- Failure and fallback:
- Evidence needed for a decision:
```
