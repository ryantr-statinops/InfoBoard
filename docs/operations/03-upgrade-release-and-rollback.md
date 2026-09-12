# Operations — Upgrade, release and rollback

## Current state

Packaging, upgrade, and rollback plans are draft; there is no tagged MVP release or clean-checkout evidence yet.

## Target release artifact

- App version, lockfile, complete migration set, and compatibility matrix.
- Setup docs for core/full modes, changelog, and known limitations.
- Backup/restore/rebuild and health/troubleshooting procedures.
- Test/lint/type/migration/security/benchmark evidence.

## Upgrade flow

1. Read the release notes and compatibility requirements.
2. Back up authoritative data and verify integrity.
3. Install locked dependencies/artifact.
4. Run migrations transactionally.
5. Rebuild derived indexes when the schema/model contract requires it.
6. Run health and the core smoke test.

## Rollback

- Roll back code by reverting/reinstalling the previous release; do not use a destructive worktree reset.
- Roll back data through the verified pre-upgrade backup.
- Automatic schema down-migration is not supported.
- A failed upgrade must preserve safe logs/evidence and must not delete the backup.

## Release gate

Do not release until migrations, backup/restore, core-mode installation, the security suite, and requirement traceability pass or have an explicit waiver.
