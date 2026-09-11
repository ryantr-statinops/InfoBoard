# Guide — Clean release verification

## Prerequisites

- A clean checkout of the candidate revision.
- Python 3.12 and `uv`.
- A disposable database copy for upgrade verification.
- A verified backup before migration testing.

## Steps

1. Run the core install from the repository README.
2. Run migrations against the disposable database copy.
3. Check `/api/health`, dashboard status, and basic create/list/search smoke.
4. If full mode is supported by the candidate, prepare the model explicitly and run the optional smoke suite.
5. Review release artifact contents and record test/lint/backup/restore evidence.

## Failure and rollback

Stop on migration, health, or smoke failure. Restore the disposable database from the verified backup and revert the candidate release; never downgrade the schema automatically.

## Evidence

Save commands, versions, output, artifact inventory, and rollback result in [execution.md](../execution.md).
