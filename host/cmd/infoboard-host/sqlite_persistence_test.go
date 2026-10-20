package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	hostruntime "github.com/ryantr-statinops/InfoBoard/host/internal/runtime"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

func TestSQLitePersistencePreparesAndLoadsOnlyBoundProfile(t *testing.T) {
	persistence := &sqlitePersistence{dataDirectory: filepath.Join(t.TempDir(), "data")}
	status, err := persistence.Prepare(context.Background())
	if err != nil || status != hostruntime.PersistenceReady {
		t.Fatalf("prepare status=%s err=%v", status, err)
	}
	profileA := domain.ProfileID("123e4567-e89b-42d3-a456-426614174000")
	binding := hostruntime.Binding{ProfileID: profileA, ContextKind: domain.ContextNormal, ProtocolVersion: protocol.CurrentVersion}
	status, err = persistence.PrepareProfile(context.Background(), binding)
	if err != nil || status != hostruntime.PersistenceReady {
		t.Fatalf("profile prepare status=%s err=%v", status, err)
	}
	configuration := persistence.Configuration()
	if configuration.ProfileID != profileA || configuration.PersistActivationRecency {
		t.Fatalf("loaded profile defaults = %+v", configuration)
	}

	configuration.PersistActivationRecency = true
	versions := storage.CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion}
	if result := persistence.repository.SaveConfiguration(context.Background(), configuration, versions); result.Status != storage.StatusReady {
		t.Fatalf("save profile A configuration = %+v", result)
	}
	profileB := domain.ProfileID("123e4567-e89b-42d3-a456-426614174099")
	privateBinding := hostruntime.Binding{ProfileID: profileB, ContextKind: domain.ContextPrivate, ProtocolVersion: protocol.CurrentVersion}
	status, err = persistence.PrepareProfile(context.Background(), privateBinding)
	if err != nil || status != hostruntime.PersistenceReady {
		t.Fatalf("private profile prepare status=%s err=%v", status, err)
	}
	privateConfiguration := persistence.Configuration()
	if privateConfiguration.ProfileID != profileB || privateConfiguration.PersistActivationRecency {
		t.Fatalf("private profile reused normal profile state: %+v", privateConfiguration)
	}
	if err := persistence.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := persistence.Close(); err != nil {
		t.Fatal(err)
	}
	if err := persistence.Close(); err != nil {
		t.Fatalf("repeated close: %v", err)
	}
}
