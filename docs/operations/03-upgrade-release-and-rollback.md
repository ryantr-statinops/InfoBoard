# Operations — Upgrade, release and rollback

## Current state

Packaging, upgrade và rollback plans đang ở draft; chưa có tagged MVP release hoặc clean-checkout evidence.

## Target release artifact

- App version, lockfile, complete migration set và compatibility matrix.
- Setup docs cho core/full modes, changelog và known limitations.
- Backup/restore/rebuild và health/troubleshooting procedures.
- Test/lint/type/migration/security/benchmark evidence.

## Upgrade flow

1. Đọc release notes và compatibility requirements.
2. Backup authoritative data và verify integrity.
3. Cài locked dependencies/artifact.
4. Chạy migrations transactionally.
5. Rebuild derived indexes khi schema/model contract yêu cầu.
6. Chạy health và core smoke test.

## Rollback

- Code rollback qua revert/reinstall release trước, không dùng destructive worktree reset.
- Data rollback qua verified pre-upgrade backup.
- Không tự động down-migrate schema.
- Failed upgrade phải giữ log/evidence an toàn và không xóa backup.

## Release gate

Không release khi migration, backup/restore, core-mode install, security suite hoặc requirement traceability chưa pass/được waiver rõ ràng.
