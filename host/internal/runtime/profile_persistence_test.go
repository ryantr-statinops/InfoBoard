package runtime

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

type profilePersistenceProbe struct {
	prepared bool
	binding  Binding
}

func (persistence *profilePersistenceProbe) Prepare(context.Context) (PersistenceStatus, error) {
	return PersistenceReady, nil
}

func (persistence *profilePersistenceProbe) PrepareProfile(_ context.Context, binding Binding) (PersistenceStatus, error) {
	persistence.binding = binding
	persistence.prepared = true
	return PersistenceReady, nil
}

func (*profilePersistenceProbe) Flush(context.Context) error { return nil }
func (*profilePersistenceProbe) Close() error                { return nil }

type profileReadyProjection struct {
	persistence *profilePersistenceProbe
}

func (projection profileReadyProjection) Synchronize(_ context.Context, target Target) (ProjectionStatus, error) {
	if !projection.persistence.prepared || target.ProfileID != projection.persistence.binding.ProfileID || target.ContextKind != projection.persistence.binding.ContextKind {
		return ProjectionStatus{}, errors.New("profile persistence did not complete before synchronization")
	}
	return readyProjection().Synchronize(context.Background(), target)
}

func TestProfilePersistenceLoadsNegotiatedBindingBeforeProjectionSync(t *testing.T) {
	store := &profilePersistenceProbe{}
	config := DefaultConfig("test", NewBoundedSink(io.Discard, 4096), testLimits())
	config.AllowedOrigins = []string{testOrigin}
	protocol := &fakeProtocol{binding: validBinding()}
	session, err := Bootstrap(config, Invocation{Origin: testOrigin}, bytes.NewReader(frame("hello")), &frameRecorder{}, Dependencies{
		Protocol:    protocol,
		Projection:  profileReadyProjection{persistence: store},
		Persistence: store,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := session.Run(context.Background())
	if outcome.ExitCode != ExitClean || !outcome.Clean {
		t.Fatalf("session outcome = %+v", outcome)
	}
	if !store.prepared || store.binding != validBinding() {
		t.Fatalf("profile binding was not prepared correctly: prepared=%t binding=%+v", store.prepared, store.binding)
	}
}
