# Requirement traceability

This matrix assigns delivery ownership. Detailed feature, architecture, quality, task, and evidence links live in each package reference file.

| Requirement | Primary packages |
| --- | --- |
| CAP-01 | CAP-URL, CAP-SNAPSHOT |
| CAP-02 | CAP-SNAPSHOT, CAP-WORKER |
| CAP-03 | CAP-SNAPSHOT, CAP-WORKER |
| CAP-04 | CAP-URL |
| CAP-05 | DB-MIGRATION, ORG-LEGACY |
| ORG-01 | ORG-CONTEXT |
| ORG-02 | ORG-CONTEXT, ORG-UI |
| ORG-03 | ORG-CONTEXT, REL-RECOVERY |
| ORG-04 | API-CANONICAL, RET-FTS, VEC-CHROMA, ANA-PROJECTION |
| RET-01 | RET-KEYWORD, VEC-HYBRID, ANA-API |
| RET-02 | RET-FTS, RET-KEYWORD |
| RET-03 | SEM-CONFIG, SEM-EMBED, VEC-CHROMA, VEC-HYBRID |
| RET-04 | RET-FTS, RET-KEYWORD, VEC-CHROMA, VEC-HYBRID |
| RET-05 | RET-KEYWORD, VEC-HYBRID, REL-HEALTH |
| ANA-01 | ANA-PROJECTION, ANA-API |
| ANA-02 | ANA-PROJECTION, ANA-API |
| PRI-01 | SEM-CONFIG, REL-SECURITY |
| PRI-02 | API-CANONICAL, SEM-CONFIG, REL-HEALTH, REL-SECURITY |
| REL-01 | DB-MIGRATION, RET-FTS, VEC-CHROMA, CACHE-ROCKS, ANA-PROJECTION, REL-RECOVERY |
| REL-02 | CAP-WORKER, REL-RECOVERY |
| UX-01 | ORG-UI, RET-KEYWORD, VEC-HYBRID |
| UX-02 | ORG-UI, REL-HEALTH |
| UX-03 | ORG-UI, RELEASE-TEST |
| UX-04 | ORG-UI, RELEASE-TEST |

## Coverage rule

A requirement is covered only when its package reference file links the requirement, feature behavior, architecture/API/schema contract, quality scenario, task IDs, and execution anchors. Package assignment alone is not completion.
