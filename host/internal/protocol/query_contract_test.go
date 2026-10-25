package protocol

import (
	"context"
	"encoding/json"
	"testing"
)

func TestDecodeQueryAllowsOmittedLimitAndOptionalContext(t *testing.T) {
	query, failure := DecodeQuery([]byte(`{"query":"go","current_window_id":2,"current_group_id":3,"query_sequence":9}`), DefaultLimits())
	if failure != nil {
		t.Fatalf("decode query: %v", failure)
	}
	if query.HasResultLimit || query.ResultLimit != 0 {
		t.Fatalf("omitted limit = present:%v value:%d", query.HasResultLimit, query.ResultLimit)
	}
	if query.CurrentWindowID != 2 || query.CurrentGroupID == nil || *query.CurrentGroupID != 3 || query.QuerySequence != 9 {
		t.Fatalf("query context = %+v", query)
	}
}

func TestDecodeQueryRejectsInvalidLimitsAndContext(t *testing.T) {
	for _, payload := range []string{
		`{"query":"go","result_limit":0}`,
		`{"query":"go","result_limit":-1}`,
		`{"query":"go","result_limit":51}`,
	} {
		if _, failure := DecodeQuery([]byte(payload), DefaultLimits()); failure == nil || failure.Code != CodePayloadLimit {
			t.Fatalf("limit payload %s = %+v", payload, failure)
		}
	}
	for _, payload := range []string{
		`{"query":"go","current_window_id":-1}`,
		`{"query":"go","current_group_id":-1}`,
	} {
		if _, failure := DecodeQuery([]byte(payload), DefaultLimits()); failure == nil || failure.Code != CodeInvalidFrame {
			t.Fatalf("context payload %s = %+v", payload, failure)
		}
	}
}

func TestHandleQueryMapsTypedOwnerFailure(t *testing.T) {
	out := &recorder{}
	session := newSession(t, &fakeSync{}, &fakeQuery{fail: NewError(CodePayloadLimit)})
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-q1", 1, snapshotPayload(2))); err != nil {
		t.Fatal(err)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-q2", 1, `{"query":"go","result_limit":5}`)); err == nil {
		t.Fatal("typed owner failure was answered")
	}
	if code := lastErrorCode(t, out); code != CodePayloadLimit {
		t.Fatalf("typed owner failure code = %s", code)
	}
}

func TestHandleQueryDefaultsStatusAndBindsEpoch(t *testing.T) {
	out := &recorder{}
	owner := &fakeQuery{result: QueryResult{Results: []ResultRow{{
		ResultID: "result-1", TabID: 10, WindowID: 1, TitleDisplay: "one", DomainDisplay: "example.test",
	}}}}
	session := newSession(t, &fakeSync{}, owner)
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-q3", 1, snapshotPayload(2))); err != nil {
		t.Fatal(err)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-q4", 1, `{"query":"go","result_limit":5,"query_sequence":7}`)); err != nil {
		t.Fatal(err)
	}
	frames := out.envelopes(t)
	if len(frames) != 1 || frames[0].Type != TypeQueryResult {
		t.Fatalf("query frames = %+v", frames)
	}
	var result QueryResult
	if err := json.Unmarshal(frames[0].Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != QueryStatusResults || result.ProjectionEpoch != testEpoch {
		t.Fatalf("query metadata = %+v", result)
	}
	if result.AcceptedLimit != 5 || result.QuerySequence != 7 || result.IndexState != "ready" {
		t.Fatalf("query bounds = %+v", result)
	}
}

func lastErrorPayload(t *testing.T, out *recorder) ErrorPayload {
	t.Helper()
	frames := out.envelopes(t)
	for index := len(frames) - 1; index >= 0; index-- {
		if frames[index].Type != TypeError {
			continue
		}
		var payload ErrorPayload
		if err := json.Unmarshal(frames[index].Payload, &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}
	t.Fatalf("no error frame among %d frames", len(frames))
	return ErrorPayload{}
}

func TestHandleQueryPreservesRevisionExpectation(t *testing.T) {
	out := &recorder{}
	expected := uint64(5)
	owner := &fakeQuery{fail: NewError(CodeRevisionMismatch).WithExpectation(&expected, nil)}
	session := newSession(t, &fakeSync{}, owner)
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-q7", 1, snapshotPayload(2))); err != nil {
		t.Fatal(err)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-q8", 1, `{"query":"go","result_limit":5}`)); err == nil {
		t.Fatal("typed revision failure was answered")
	}
	payload := lastErrorPayload(t, out)
	if payload.Code != CodeRevisionMismatch || payload.ExpectedRevision == nil || *payload.ExpectedRevision != 5 {
		t.Fatalf("revision failure payload = %+v", payload)
	}
}

func TestHandleQueryRejectsRevisionBeforeOwner(t *testing.T) {
	out := &recorder{}
	owner := &fakeQuery{}
	session := newSession(t, &fakeSync{}, owner)
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-q9", 1, snapshotPayload(2))); err != nil {
		t.Fatal(err)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-q10", 2, `{"query":"go","result_limit":5}`)); err == nil {
		t.Fatal("unknown revision query was answered")
	}
	if owner.seen != 0 {
		t.Fatalf("owner was called for unknown revision: %d", owner.seen)
	}
	if code := lastErrorCode(t, out); code != CodeRevisionMismatch {
		t.Fatalf("unknown revision code = %s", code)
	}
}

func TestHandleQueryReportsNoResultsAsSuccess(t *testing.T) {
	out := &recorder{}
	owner := &fakeQuery{result: QueryResult{Results: []ResultRow{}}}
	session := newSession(t, &fakeSync{}, owner)
	mustBind(t, session, out)
	if err := session.Accept(context.Background(), out, frame(t, TypeSnapshot, "r-q5", 1, snapshotPayload(2))); err != nil {
		t.Fatal(err)
	}
	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeQuery, "r-q6", 1, `{"query":"missing","result_limit":5}`)); err != nil {
		t.Fatal(err)
	}
	frames := out.envelopes(t)
	if len(frames) != 1 || frames[0].Type != TypeQueryResult {
		t.Fatalf("query frames = %+v", frames)
	}
	var result QueryResult
	if err := json.Unmarshal(frames[0].Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != QueryStatusNoResults || len(result.Results) != 0 {
		t.Fatalf("no-result response = %+v", result)
	}
}
