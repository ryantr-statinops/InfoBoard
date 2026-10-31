package main

import (
	"context"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/index"
	"github.com/ryantr-statinops/InfoBoard/host/internal/projection"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/internal/query"
	"github.com/ryantr-statinops/InfoBoard/host/internal/runtime"
	hostsync "github.com/ryantr-statinops/InfoBoard/host/internal/sync"
)

type liveProjectionAdapter struct {
	orchestrator *hostsync.Orchestrator
}

func (p *liveProjectionAdapter) Synchronize(ctx context.Context, target runtime.Target) (runtime.ProjectionStatus, error) {
	state := p.orchestrator.State()
	epoch := state.Epoch
	if epoch == "" {
		epoch = "1"
	}
	rev := state.Revision
	if rev == 0 {
		rev = 1
	}
	return runtime.ProjectionStatus{
		Known:    true,
		Epoch:    epoch,
		Revision: rev,
		Count:    state.Count,
		Ready:    true,
	}, nil
}

func newRuntimeDependencies(store runtime.Persistence, version string) (runtime.Dependencies, error) {
	publisher := index.NewPublisher()
	partition := projection.Partition{ProfileID: "default", ContextKind: domain.ContextNormal}
	orchestrator, err := hostsync.NewOrchestrator(partition, publisher)
	if err != nil {
		return runtime.Dependencies{Persistence: store}, err
	}

	queryConfig := query.Config{
		FallbackResultLimit: 20,
		MaxResultLimit:      50,
		RankingModelVersion: "rank-1",
	}
	queryService, err := query.New(publisher, nil, nil, nil, nil, queryConfig)
	if err != nil {
		return runtime.Dependencies{Persistence: store}, err
	}

	protoConfig := protocol.Config{
		HostVersion:         version,
		RankingModelVersion: "rank-1",
		Limits:              protocol.DefaultLimits(),
		Budgets:             protocol.DefaultBudgets(),
	}
	protoSession, err := protocol.NewSession(protoConfig, orchestrator, queryService, nil)
	if err != nil {
		return runtime.Dependencies{Persistence: store}, err
	}

	return runtime.Dependencies{
		Protocol:    protoSession,
		Projection:  &liveProjectionAdapter{orchestrator: orchestrator},
		Persistence: store,
	}, nil
}
