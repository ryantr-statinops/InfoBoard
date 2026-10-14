// Package storage owns InfoBoard's durable local metadata. The live tab
// projection and lexical index remain in memory and are never persisted here.
package storage

import (
	"fmt"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/storage/migrations"
)

const (
	SupportedSchemaVersion uint64 = 1
	DatabaseApplicationID  int64  = 0x494E464F // "INFO"
	DatabaseOwnerID               = "infoboard"

	MaxActivationRowsPerProfile     = 500
	ActivationRetentionSeconds      = int64(30 * 24 * 60 * 60)
	DiagnosticsRetentionMillis      = int64(7 * 24 * 60 * 60 * 1000)
	MaxDiagnosticsPayloadBytes      = 10 * 1024 * 1024
	MaxDiagnosticRowBytes           = protocol.MaxPayloadBytes
	MaxStoredVersionBytes           = protocol.MaxOpaqueValueBytes
	MaxStoredProfileIDBytes         = 36
	DefaultResultLimit              = 5
	DefaultDensity                  = "comfortable"
	DefaultTheme                    = "system"
	DefaultShowContextLabels        = 1
	DefaultPersistActivationRecency = 0
	DefaultRankingModelVersion      = ""
)

const (
	tableSchemaMeta         = "schema_meta"
	tableSchemaMigrations   = "schema_migrations"
	tableConfiguration      = "configuration"
	tableInstallationState  = "installation_state"
	tableActivationMetadata = "activation_metadata"
	tableDiagnostics        = "diagnostics"
)

var ownedTables = [...]string{
	tableSchemaMeta,
	tableSchemaMigrations,
	tableConfiguration,
	tableInstallationState,
	tableActivationMetadata,
	tableDiagnostics,
}

// OwnedTableNames returns the SQLite objects the repository may reset or drop.
// SQLite's internal sqlite_sequence table is deliberately not in the ownership
// manifest; it is managed only as a side effect of the owned AUTOINCREMENT tables.
func OwnedTableNames() []string {
	return append([]string(nil), ownedTables[:]...)
}

// SchemaStatements returns the deterministic initial schema. Migration version
// and application_id are committed separately by the migration runner only after
// every statement and postcondition succeeds.
func SchemaStatements() []string {
	return append([]string(nil), schemaStatements[:]...)
}

// InitialMigrations returns a detached registry for a new repository. Later
// schema changes append migrations; an existing version is never rewritten.
func InitialMigrations() []migrations.Migration {
	return []migrations.Migration{{Version: 1, ID: "initial_schema_v1", Statements: SchemaStatements()}}
}

var schemaStatements = buildSchemaStatements()

func buildSchemaStatements() [8]string {
	return [8]string{
		fmt.Sprintf(`CREATE TABLE schema_meta (
			key TEXT PRIMARY KEY NOT NULL CHECK(length(CAST(key AS BLOB)) BETWEEN 1 AND %d),
			value TEXT NOT NULL CHECK(length(CAST(value AS BLOB)) <= %d)
		) WITHOUT ROWID`, MaxStoredVersionBytes, MaxStoredVersionBytes),
		fmt.Sprintf(`CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY NOT NULL CHECK(version > 0),
			migration_id TEXT NOT NULL UNIQUE CHECK(length(CAST(migration_id AS BLOB)) BETWEEN 1 AND %d),
			checksum TEXT NOT NULL CHECK(length(CAST(checksum AS BLOB)) = 64 AND lower(checksum) = checksum AND checksum NOT GLOB '*[^0-9a-f]*'),
			applied_at INTEGER NOT NULL CHECK(applied_at >= 0)
		)`, MaxStoredVersionBytes),
		fmt.Sprintf(`CREATE TABLE configuration (
			profile_id TEXT PRIMARY KEY NOT NULL CHECK(length(CAST(profile_id AS BLOB)) = %d),
			result_limit INTEGER NOT NULL DEFAULT %d CHECK(result_limit BETWEEN 1 AND %d),
			density TEXT NOT NULL DEFAULT '%s' CHECK(density IN ('compact', 'comfortable')),
			theme TEXT NOT NULL DEFAULT '%s' CHECK(length(CAST(theme AS BLOB)) BETWEEN 1 AND %d),
			show_context_labels INTEGER NOT NULL DEFAULT %d CHECK(show_context_labels IN (0, 1)),
			persist_activation_recency INTEGER NOT NULL DEFAULT %d CHECK(persist_activation_recency IN (0, 1)),
			protocol_version INTEGER NOT NULL DEFAULT %d CHECK(protocol_version > 0),
			ranking_model_version TEXT NOT NULL DEFAULT '%s' CHECK(length(CAST(ranking_model_version AS BLOB)) <= %d),
			updated_at INTEGER NOT NULL DEFAULT (unixepoch()) CHECK(updated_at >= 0)
		) WITHOUT ROWID`, MaxStoredProfileIDBytes, DefaultResultLimit, protocol.MaxResults, DefaultDensity, DefaultTheme, MaxStoredVersionBytes, DefaultShowContextLabels, DefaultPersistActivationRecency, protocol.CurrentVersion, DefaultRankingModelVersion, MaxStoredVersionBytes),
		fmt.Sprintf(`CREATE TABLE installation_state (
			installation_key TEXT PRIMARY KEY NOT NULL CHECK(installation_key = '%s'),
			extension_version TEXT NOT NULL DEFAULT '' CHECK(length(CAST(extension_version AS BLOB)) <= %d),
			host_version TEXT NOT NULL DEFAULT '' CHECK(length(CAST(host_version AS BLOB)) <= %d),
			browser_family TEXT NOT NULL DEFAULT '' CHECK(length(CAST(browser_family AS BLOB)) <= %d),
			protocol_version INTEGER NOT NULL DEFAULT 0 CHECK(protocol_version >= 0),
			ranking_model_version TEXT NOT NULL DEFAULT '' CHECK(length(CAST(ranking_model_version AS BLOB)) <= %d),
			host_registration_state TEXT NOT NULL DEFAULT 'unknown' CHECK(length(CAST(host_registration_state AS BLOB)) BETWEEN 1 AND %d),
			last_successful_migration INTEGER NOT NULL DEFAULT 0 CHECK(last_successful_migration >= 0),
			repair_state TEXT NOT NULL DEFAULT 'normal' CHECK(length(CAST(repair_state AS BLOB)) BETWEEN 1 AND %d),
			updated_at INTEGER NOT NULL DEFAULT (unixepoch()) CHECK(updated_at >= 0)
		) WITHOUT ROWID`, DatabaseOwnerID, MaxStoredVersionBytes, MaxStoredVersionBytes, MaxStoredVersionBytes, MaxStoredVersionBytes, MaxStoredVersionBytes, MaxStoredVersionBytes),
		fmt.Sprintf(`CREATE TABLE activation_metadata (
			activation_id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id TEXT NOT NULL CHECK(length(CAST(profile_id AS BLOB)) = %d),
			context_kind TEXT NOT NULL DEFAULT 'normal' CHECK(context_kind = 'normal'),
			tab_id INTEGER NOT NULL CHECK(tab_id BETWEEN 0 AND %d),
			domain TEXT NOT NULL CHECK(length(CAST(domain AS BLOB)) BETWEEN 1 AND %d),
			activated_at INTEGER NOT NULL CHECK(activated_at >= 0),
			source TEXT NOT NULL CHECK(source IN ('keyboard_enter', 'surface_select'))
		)`, MaxStoredProfileIDBytes, domain.MaxProjectionRevision, protocol.MaxDomainBytes),
		`CREATE INDEX activation_profile_retention ON activation_metadata(profile_id, activated_at ASC, activation_id ASC)`,
		fmt.Sprintf(`CREATE TABLE diagnostics (
			diagnostic_id INTEGER PRIMARY KEY AUTOINCREMENT,
			occurred_at INTEGER NOT NULL CHECK(occurred_at >= 0),
			profile_id TEXT CHECK(profile_id IS NULL OR length(CAST(profile_id AS BLOB)) = %d),
			event_kind TEXT NOT NULL CHECK(length(CAST(event_kind AS BLOB)) BETWEEN 1 AND %d),
			error_class TEXT NOT NULL CHECK(length(CAST(error_class AS BLOB)) BETWEEN 1 AND %d),
			retryable INTEGER NOT NULL CHECK(retryable IN (0, 1)),
			duration_ms INTEGER NOT NULL CHECK(duration_ms >= 0),
			projection_revision INTEGER NOT NULL DEFAULT 0 CHECK(projection_revision BETWEEN 0 AND %d),
			protocol_version INTEGER NOT NULL DEFAULT 0 CHECK(protocol_version >= 0),
			ranking_model_version TEXT NOT NULL DEFAULT '' CHECK(length(CAST(ranking_model_version AS BLOB)) <= %d),
			schema_version INTEGER NOT NULL CHECK(schema_version >= 0),
			payload_bytes INTEGER NOT NULL CHECK(payload_bytes BETWEEN 0 AND %d),
			redacted_payload TEXT NOT NULL CHECK(payload_bytes = length(CAST(redacted_payload AS BLOB)))
		)`, MaxStoredProfileIDBytes, MaxStoredVersionBytes, MaxStoredVersionBytes, domain.MaxProjectionRevision, MaxStoredVersionBytes, MaxDiagnosticRowBytes),
		`CREATE INDEX diagnostics_retention ON diagnostics(occurred_at ASC, diagnostic_id ASC)`,
	}
}
