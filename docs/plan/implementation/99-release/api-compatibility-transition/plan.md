# API compatibility transition: implementation plan

## Approach

Ship `/api/*` as thin one-release adapters over `/api/v1/*`, publish deprecation metadata, and remove them only in the following release. Work in compatibility routers, release manifest, HTTP contract tests, migration notes. Treat clean-environment reproducibility and evidence completeness as implementation outputs.

## Delivery flow

1. Define clean fixtures, environments, and acceptance mapping.
2. Implement the package boundary and automation.
3. Exercise success, failure, upgrade, rollback, and compatibility paths.
4. Preserve redacted artifacts and exact commands.
5. Review against MVP acceptance before changing status.

## Release control

A failed mandatory gate blocks release unless an accepted upstream decision changes scope. Evidence identifies commit, environment, command, result, artifact, and reviewer.

## Rollout

Complete API-CANONICAL and all API-owning packages; removal requires one released compatibility version; execute [tasks](tasks.md) in order; update all rollups with the evidence-bearing completion commit.
