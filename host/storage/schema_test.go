package storage

import (
	"database/sql"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"reflect"
	"testing"

	_ "modernc.org/sqlite"
)

func TestInitialSchemaCreatesOnlyOwnedTablesAndSafeDefaults(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	for _, statement := range SchemaStatements() {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("apply schema statement: %v", err)
		}
	}

	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	var created []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		created = append(created, name)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	wantTables := []string{tableActivationMetadata, tableConfiguration, tableDiagnostics, tableInstallationState, tableSchemaMeta, tableSchemaMigrations}
	wantOwned := []string{tableSchemaMeta, tableSchemaMigrations, tableConfiguration, tableInstallationState, tableActivationMetadata, tableDiagnostics}
	if !reflect.DeepEqual(created, wantTables) || !reflect.DeepEqual(OwnedTableNames(), wantOwned) {
		t.Fatalf("schema tables=%v owned=%v want tables=%v owned=%v", created, OwnedTableNames(), wantTables, wantOwned)
	}

	const profileA = "123e4567-e89b-42d3-a456-426614174000"
	if _, err := db.Exec(`INSERT INTO configuration(profile_id) VALUES (?)`, profileA); err != nil {
		t.Fatalf("insert configuration defaults: %v", err)
	}
	var resultLimit int
	var density, theme, rankingVersion string
	var showLabels, persistRecency, protocolVersion int
	if err := db.QueryRow(`SELECT result_limit, density, theme, show_context_labels, persist_activation_recency, protocol_version, ranking_model_version FROM configuration WHERE profile_id = ?`, profileA).Scan(&resultLimit, &density, &theme, &showLabels, &persistRecency, &protocolVersion, &rankingVersion); err != nil {
		t.Fatal(err)
	}
	if resultLimit != DefaultResultLimit || density != DefaultDensity || theme != DefaultTheme || showLabels != 1 || persistRecency != 0 || protocolVersion != protocol.CurrentVersion || rankingVersion != "" {
		t.Fatalf("configuration defaults = limit:%d density:%q theme:%q labels:%d recency:%d protocol:%d ranking:%q", resultLimit, density, theme, showLabels, persistRecency, protocolVersion, rankingVersion)
	}
}

func TestInitialSchemaRejectsOutOfContractDurableRows(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range SchemaStatements() {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("apply schema statement: %v", err)
		}
	}

	const profileID = "123e4567-e89b-42d3-a456-426614174000"
	if _, err := db.Exec(`INSERT INTO configuration(profile_id, result_limit) VALUES (?, ?)`, profileID, protocol.MaxResults+1); err == nil {
		t.Fatal("configuration accepted a result limit above the IP-07 bound")
	}
	if _, err := db.Exec(`INSERT INTO activation_metadata(profile_id, context_kind, tab_id, domain, activated_at, source) VALUES (?, 'private', 1, 'example.com', 1, 'keyboard_enter')`, profileID); err == nil {
		t.Fatal("private-context activation entered durable storage")
	}
	if _, err := db.Exec(`INSERT INTO activation_metadata(profile_id, context_kind, tab_id, domain, activated_at, source) VALUES (?, 'normal', 1, 'example.com', 1, 'unknown_source')`, profileID); err == nil {
		t.Fatal("activation metadata accepted an undeclared source")
	}
	if _, err := db.Exec(`INSERT INTO diagnostics(occurred_at, event_kind, error_class, retryable, duration_ms, schema_version, payload_bytes, redacted_payload) VALUES (1, 'storage.failure', 'busy', 1, 0, 1, 1, 'é')`); err == nil {
		t.Fatal("diagnostics accepted a character count instead of the UTF-8 byte count")
	}
}
