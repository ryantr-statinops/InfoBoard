-- IP-17 retention audit seed. The audit script creates a temporary database from
-- this schema and inserts synthetic, non-sensitive rows relative to the run time.
CREATE TABLE schema_meta (
  key TEXT PRIMARY KEY NOT NULL,
  value TEXT NOT NULL
) WITHOUT ROWID;

CREATE TABLE schema_migrations (
  version INTEGER PRIMARY KEY NOT NULL,
  migration_id TEXT NOT NULL UNIQUE,
  checksum TEXT NOT NULL,
  applied_at INTEGER NOT NULL
);

CREATE TABLE configuration (
  profile_id TEXT PRIMARY KEY NOT NULL,
  result_limit INTEGER NOT NULL DEFAULT 10,
  persist_activation_recency INTEGER NOT NULL DEFAULT 1,
  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
) WITHOUT ROWID;

CREATE TABLE installation_state (
  installation_key TEXT PRIMARY KEY NOT NULL,
  host_version TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
) WITHOUT ROWID;

CREATE TABLE activation_metadata (
  activation_id INTEGER PRIMARY KEY AUTOINCREMENT,
  profile_id TEXT NOT NULL,
  context_kind TEXT NOT NULL DEFAULT 'normal' CHECK(context_kind = 'normal'),
  tab_id INTEGER NOT NULL,
  domain TEXT NOT NULL,
  activated_at INTEGER NOT NULL,
  source TEXT NOT NULL CHECK(source IN ('keyboard_enter', 'surface_select'))
);
CREATE INDEX activation_profile_retention ON activation_metadata(profile_id, activated_at ASC, activation_id ASC);

CREATE TABLE diagnostics (
  diagnostic_id INTEGER PRIMARY KEY AUTOINCREMENT,
  occurred_at INTEGER NOT NULL,
  profile_id TEXT,
  event_kind TEXT NOT NULL,
  error_class TEXT NOT NULL,
  retryable INTEGER NOT NULL,
  duration_ms INTEGER NOT NULL,
  projection_revision INTEGER NOT NULL DEFAULT 0,
  protocol_version INTEGER NOT NULL DEFAULT 1,
  ranking_model_version TEXT NOT NULL DEFAULT '',
  schema_version INTEGER NOT NULL DEFAULT 1,
  payload_bytes INTEGER NOT NULL,
  redacted_payload TEXT NOT NULL
);
CREATE INDEX diagnostics_retention ON diagnostics(occurred_at ASC, diagnostic_id ASC);

-- Deliberately unrelated data: an ownership audit must never touch it.
CREATE TABLE browser_state (
  state_key TEXT PRIMARY KEY NOT NULL,
  state_value TEXT NOT NULL
) WITHOUT ROWID;
