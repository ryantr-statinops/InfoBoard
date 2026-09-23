# SKILLS integration

## Source provenance

- Source: `https://github.com/ryantr-statinops/SKILLS.git`
- Pinned revision: `b47f46307f9d7a140a961c729fb6d1b866e7dc2b`
- Export date: 2026-09-23
- Selected skills: 71 unique registry entries
- Destination: `.agent/skills/`
- Runtime adapter: none; `.agent/skills/` is the canonical portable root.
- Source checkout used for validation: `/tmp/infoboard-skills-main` at the pinned revision; it is not part of this repository.

The portable export includes skill entrypoints, linked local resources, templates, `.skill-sync.json`, and `.skill-catalog.json`. It does not copy the source registry's `docs/`, `scripts/`, `tests/`, `.github/`, or repository-level instruction files into this project.

The existing untracked export was reconciled against the pinned revision. Parent/child skill paths were checked with merge semantics: identical target files are retained once, and conflicting target contents are rejected before copying. The upstream full-registry sync command was not used unchanged because its dry-run does not detect parent/child target overlap.

## Routing policy

All 71 entries are installed for discovery. A task activates only the smallest set whose descriptions and boundaries match the task. Installation is not equivalent to loading every skill into every task.

### Active project-relevant personal skills

- `personal/workflow/brainstorming`
- `personal/workflow/personal-research`
- `personal/product/project-scoping`
- `personal/product/mvp-design`
- `personal/product/architecture-review`
- `personal/decision/technology-selection`
- `personal/decision/architecture-tradeoff`
- `personal/decision/scope-control`
- `personal/engineering/core`
- `personal/engineering/backend/go`
- `personal/engineering/backend/service-architecture`
- `personal/engineering/data/databases`
- `personal/engineering/infrastructure/linux`
- `personal/engineering/infrastructure/networking`
- `personal/engineering/frontend/lightweight-web`

### Conditional personal skills

- `personal/engineering/ai` and `personal/engineering/ai/core`: semantic search or embedding decisions.
- `personal/engineering/data/core`, `personal/engineering/data/pipelines`, and `personal/engineering/data/orchestration`: a durable indexing pipeline.
- `personal/engineering/infrastructure/docker`: container-based packaging or deployment.
- `personal/statistics`: measured ranking or performance evaluation.

### Installed but not routed initially

Python, Node.js, R, LaTeX, education, and quant skills remain installed for discovery. They are not routed for the current tab-search product unless a future task explicitly matches their boundaries.

## Validation record

Source checks run at the pinned revision:

```text
python3 scripts/validate_skills.py
All skills passed validation.

python3 scripts/generate_skill_index.py --check
Skill index check passed.

python3 scripts/check_markdown_links.py
Checked 152 local link(s); skipped 6 external link(s).
Markdown links passed.
```

Consumer reconciliation checks:

- Catalog: 71 unique IDs; every catalog path resolves below `.agent/skills/` to a readable `SKILL.md`.
- Manifest: 226 managed files present; all SHA-256 hashes match `.skill-sync.json`.
- Source registry: 71 unique IDs at the pinned revision; catalog has no missing or extra IDs.
- `git diff --check`: passed.
- Working-tree scope before publish: integration files are limited to `.agent/skills/`, `CONTEXT.md`, and `docs/agent/skill-integration.md`.

## Routing smoke result

The smoke task researched a warm Go daemon versus a Chromium Native Messaging host for the tab-search runtime and produced `docs/agent/decisions/runtime-boundary.md` during validation. The artifact was removed as throwaway output after its result was captured here.

- Activated route: `common/workflow/research-decision`, `common/research/research`, `common/research/comparison`, `personal/workflow/personal-research`, `personal/decision/technology-selection`, `personal/decision/architecture-tradeoff`, `personal/engineering/backend/go`, and `personal/engineering/backend/service-architecture`.
- Decision result: run a bounded Go Native Messaging host spike; do not settle long-term process residency or storage before measuring cold/warm latency, reconnect behavior, and installation friction.
- Boundary task: a quantitative-strategy query returned the quant candidates, but none of `personal/quant`, `personal/quant/research`, or `personal/statistics` was activated for the tab-search task.

Runtime discovery command:

```text
python3 /tmp/infoboard-skills-main/scripts/discover_skills.py \
  --registry .agent/skills/.skill-catalog.json \
  --format json "warm Go Native Messaging tab-search runtime"
```

This returned `personal/engineering/backend/go` as the top candidate and did not return a quant skill. `.agent/skills/` is the only runtime skill root; no `.agents/skills/`, `.codex/skills/`, or `.claude/skills/` adapter was needed.
