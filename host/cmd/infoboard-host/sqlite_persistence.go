package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	hostruntime "github.com/ryantr-statinops/InfoBoard/host/internal/runtime"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

var errPersistenceFlush = errors.New("storage flush failed")
var errPersistenceClose = errors.New("storage close failed")

type sqlitePersistence struct {
	mu            sync.RWMutex
	dataDirectory string
	repository    *storage.Repository
	configuration storage.Configuration
}

func newDefaultSQLitePersistence() *sqlitePersistence {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return &sqlitePersistence{}
	}
	return &sqlitePersistence{dataDirectory: filepath.Join(base, "infoboard")}
}

func (persistence *sqlitePersistence) Prepare(ctx context.Context) (hostruntime.PersistenceStatus, error) {
	if persistence == nil {
		return hostruntime.PersistenceDegraded, nil
	}
	repository, result := storage.Open(ctx, persistence.dataDirectory)
	persistence.mu.Lock()
	previous := persistence.repository
	persistence.repository = repository
	persistence.configuration = storage.Configuration{}
	persistence.mu.Unlock()
	if previous != nil && previous != repository {
		_ = previous.Close()
	}
	return persistenceStatus(result.Status), nil
}

func (persistence *sqlitePersistence) PrepareProfile(ctx context.Context, binding hostruntime.Binding) (hostruntime.PersistenceStatus, error) {
	if persistence == nil {
		return hostruntime.PersistenceDegraded, nil
	}
	if binding.ContextKind != domain.ContextNormal {
		persistence.mu.Lock()
		persistence.configuration = storage.DefaultConfiguration(binding.ProfileID)
		persistence.mu.Unlock()
		return persistenceStatus(persistence.currentRepositoryStatus()), nil
	}
	persistence.mu.RLock()
	repository := persistence.repository
	persistence.mu.RUnlock()
	if repository == nil {
		return hostruntime.PersistenceDegraded, nil
	}
	configuration, result := repository.LoadConfiguration(ctx, string(binding.ProfileID), storage.CompatibilityVersions{
		ProtocolVersion: binding.ProtocolVersion,
	})
	persistence.mu.Lock()
	persistence.configuration = configuration
	persistence.mu.Unlock()
	return persistenceStatus(result.Status), nil
}

func (persistence *sqlitePersistence) Configuration() storage.Configuration {
	if persistence == nil {
		return storage.Configuration{}
	}
	persistence.mu.RLock()
	configuration := persistence.configuration
	persistence.mu.RUnlock()
	return configuration
}

func (persistence *sqlitePersistence) Flush(ctx context.Context) error {
	if persistence == nil {
		return nil
	}
	persistence.mu.RLock()
	repository := persistence.repository
	persistence.mu.RUnlock()
	if repository == nil {
		return nil
	}
	if result := repository.PruneRetention(ctx, time.Now()); result.Status != storage.StatusReady {
		return errPersistenceFlush
	}
	return nil
}

func (persistence *sqlitePersistence) Close() error {
	if persistence == nil {
		return nil
	}
	persistence.mu.Lock()
	repository := persistence.repository
	persistence.repository = nil
	persistence.mu.Unlock()
	if repository == nil {
		return nil
	}
	if err := repository.Close(); err != nil {
		return errPersistenceClose
	}
	return nil
}

func (persistence *sqlitePersistence) currentRepositoryStatus() storage.Status {
	persistence.mu.RLock()
	repository := persistence.repository
	persistence.mu.RUnlock()
	if repository == nil {
		return storage.StatusDegraded
	}
	return repository.CurrentResult().Status
}

func persistenceStatus(status storage.Status) hostruntime.PersistenceStatus {
	switch status {
	case storage.StatusReady, storage.StatusRebuilt:
		return hostruntime.PersistenceReady
	case storage.StatusDisabled:
		return hostruntime.PersistenceDisabled
	default:
		return hostruntime.PersistenceDegraded
	}
}

var _ hostruntime.Persistence = (*sqlitePersistence)(nil)
var _ hostruntime.ProfilePersistence = (*sqlitePersistence)(nil)
