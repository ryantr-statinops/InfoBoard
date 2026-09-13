# RocksDB embedding cache: implementation plan

## Approach

Reuse compatible document embeddings through a revision-scoped cache that can be bypassed, cleaned, or rebuilt without affecting canonical state. Implement the boundary in rocksdict adapter, cache key/value codec, semantic pipeline integration, cache tests. Use explicit adapters so provider and store failure cannot leak into canonical repositories or keyword retrieval.

## Flow

1. Validate configuration, consent, revision, and canonical eligibility.
2. Perform bounded provider or derived-store work.
3. Validate every external/derived result before use.
4. Persist durable status/checkpoints and project safe degradation.
5. Test revocation, incompatibility, outage, restart, cleanup, and rebuild.

## Security and recovery

Recheck consent immediately before transfer. Never log API keys, query text, raw snapshots, or sensitive responses. Derived records carry bookmark, snapshot, chunk, and revision identity and are disposable.

## Rollout

Complete SEM-EMBED; deliver [tasks](tasks.md) in order; record real [evidence](execution.md); update rollups only after milestone acceptance.
