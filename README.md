# InfoBoard

InfoBoard is a keyboard-first Chrome/Edge desktop extension for searching and activating eligible open tabs across windows and browser contexts.

The extension owns browser permissions, tab observation, the focused search surface, and tab activation. A local Go Native Messaging host owns protocol handling, projection reconciliation, in-memory lexical search, deterministic ranking, bounded SQLite persistence, health, and diagnostics. The normal path is local and metadata-only: no page-content indexing, browsing-history access, cloud service, or loopback transport.

## Project status

IP-01 and IP-02 are complete in dev. IP-03 is merged into dev as a browser-toolbar popup with open (Ctrl+Shift+Y / Command+Shift+Y) and close (Ctrl+Shift+X / Command+Shift+X) shortcuts. Build, unit/domain checks, corpus, fixtures, and targeted Go checks pass. Chrome/Edge popup acceptance remains unverified; it was skipped at the user's direction.

- [Canonical product specification](docs/plan/refactor/README.md)
- [Implementation execution dashboard](docs/plan/implementation/README.md)
- [Phase inventory, dependency DAG, and branch protocol](docs/plan/implementation/index.md)
- [Workspace and agent context](CONTEXT.md)

## Project map

| Path | Responsibility |
| --- | --- |
| [`docs/plan/refactor/`](docs/plan/refactor/README.md) | Binding product behavior, requirements, privacy, architecture, runtime, and acceptance |
| [`docs/plan/implementation/`](docs/plan/implementation/README.md) | Phase plans, commit-to-task mapping, execution status, and evidence tracking |
| [`fixtures/`](fixtures/catalog.json) | Shared contract fixture catalog and schema |
| [`tests/implementation/validate_corpus.py`](tests/implementation/validate_corpus.py) | Standard-library phase, ownership, link, and fixture-contract validator |
| [`docs/architecture/`](docs/architecture/README.md), [`docs/design/`](docs/design/README.md), [`docs/quality/`](docs/quality/README.md), [`docs/operations/`](docs/operations/README.md) | Preserved detailed views and supporting design/quality/operations references |
| [`.agent/skills/`](.agent/skills/) | Repository agent skills and templates |

## Starting implementation

Follow the dependency waves in `docs/plan/implementation/index.md`. Start from the latest `origin/dev` on a dedicated phase branch, follow Section 6 task IDs and Section 7 commit instructions, and run Section 8 acceptance checks. The coordinator alone updates the README execution dashboard. IP-01 provides the corpus check `python3 tests/implementation/validate_corpus.py`; runtime source scaffolds and their setup/test commands are established by their owning phases.

Implementation work integrates into `dev` only after dependencies and phase acceptance checks pass. The release PR from `dev` to `main` stays open for the user to merge manually.
