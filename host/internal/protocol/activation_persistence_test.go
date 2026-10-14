package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/runtime"
)

type failedActivationHook struct{}

func (failedActivationHook) RecordActivation(context.Context, RequestContext, ActivationStatus) error {
	return errors.New("storage unavailable")
}

func TestActivationAckRemainsAcceptedWhenPersistenceFails(t *testing.T) {
	syncOwner := &fakeSync{}
	session, err := NewSession(testConfig(), syncOwner, &fakeQuery{}, failedActivationHook{})
	if err != nil {
		t.Fatal(err)
	}
	out := &recorder{}
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-2", 1, snapshotPayload(3))); err != nil {
		t.Fatal(err)
	}
	out.reset()
	payload := `{"result_reference":{"tab_id":10,"projection_epoch":"` + testEpoch + `","projection_revision":1,"result_id":"res-1"},"tab_id":10,"projection_revision":1,"outcome":"activated"}`
	if err := session.Accept(context.Background(), out, frame(t, TypeActivationObserved, "r-3", 1, payload)); err != nil {
		t.Fatal(err)
	}
	frames := out.envelopes(t)
	if len(frames) != 1 || frames[0].Type != TypeActivationAck {
		t.Fatalf("activation acknowledgement frames = %+v", frames)
	}
	var acknowledgement ActivationAck
	if err := json.Unmarshal(frames[0].Payload, &acknowledgement); err != nil {
		t.Fatal(err)
	}
	if !acknowledgement.Accepted {
		t.Fatal("durable metadata failure rejected browser-owned activation")
	}
	if session.persistenceState() != string(runtime.PersistenceDegraded) {
		t.Fatalf("persistence status after write failure = %s", session.persistenceState())
	}
}
