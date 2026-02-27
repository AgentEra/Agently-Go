package core_test

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/testkit"
	"github.com/AgentEra/Agently-Go/agently/types"
)

func TestResponseGetDataAndEnsureKeysRetry(t *testing.T) {
	requestCount := &atomic.Int32{}
	script := func(call int) []types.ResponseMessage {
		done := `{"foo":"bar"}`
		if call >= 2 {
			done = `{"foo":"bar","must":"ok"}`
		}
		return []types.ResponseMessage{
			{Event: types.ResponseEventOriginalDelta, Data: fmt.Sprintf(`{"attempt":%d}`, call)},
			{Event: types.ResponseEventDelta, Data: done},
			{Event: types.ResponseEventDone, Data: done},
			{Event: types.ResponseEventOriginalDone, Data: map[string]any{"attempt": call, "raw": done}},
			{Event: types.ResponseEventMeta, Data: map[string]any{"attempt": call}},
		}
	}
	manager := newRegressionPluginManager(script, requestCount)

	req := core.NewModelRequest(manager, "ensure-keys", core.NewDefaultSettings(nil), nil, nil)
	req.Input("regression")
	req.Output(map[string]any{"foo": "string", "must": "string"})

	ctx, cancel := testkit.TestContext(t, 5*time.Second)
	defer cancel()

	data, err := req.GetData(ctx, core.GetDataOptions{Type: "parsed", EnsureKeys: []string{"must"}, MaxRetries: 2})
	if err != nil {
		t.Fatalf("GetData parsed failed: %v", err)
	}
	parsed, ok := data.(map[string]any)
	if !ok {
		t.Fatalf("expected parsed map, got %T", data)
	}
	if parsed["must"] != "ok" {
		t.Fatalf("expected ensure key must=ok, got %#v", parsed)
	}
	if requestCount.Load() < 2 {
		t.Fatalf("expected ensure_keys retry to trigger 2 requests, got %d", requestCount.Load())
	}

	req2 := core.NewModelRequest(manager, "data-types", core.NewDefaultSettings(nil), nil, nil)
	req2.Input("regression")
	req2.Output(map[string]any{"foo": "string", "must": "string"})

	original, err := req2.GetData(ctx, core.GetDataOptions{Type: "original"})
	if err != nil {
		t.Fatalf("GetData original failed: %v", err)
	}
	originalMap, ok := original.(map[string]any)
	if !ok || originalMap["attempt"] == nil {
		t.Fatalf("unexpected original data: %#v", original)
	}

	allData, err := req2.GetData(ctx, core.GetDataOptions{Type: "all"})
	if err != nil {
		t.Fatalf("GetData all failed: %v", err)
	}
	if _, ok := allData.(types.ModelResult); !ok {
		t.Fatalf("expected ModelResult for all type, got %T", allData)
	}
}

func TestResponseGeneratorTypes(t *testing.T) {
	script := func(call int) []types.ResponseMessage {
		_ = call
		return []types.ResponseMessage{
			{Event: types.ResponseEventOriginalDelta, Data: `{"delta":"one"}`},
			{Event: types.ResponseEventDelta, Data: `{"answer":"he`},
			{Event: types.ResponseEventToolCalls, Data: []any{map[string]any{"name": "sum"}}},
			{Event: types.ResponseEventDelta, Data: `llo","status":"ok"}`},
			{Event: types.ResponseEventDone, Data: `{"answer":"hello","status":"ok"}`},
			{Event: types.ResponseEventOriginalDone, Data: map[string]any{"done": true}},
		}
	}
	manager := newRegressionPluginManager(script, nil)

	newResponse := func(name string) *core.ModelResponse {
		req := core.NewModelRequest(manager, name, core.NewDefaultSettings(nil), nil, nil)
		req.Input("streaming regression")
		req.Output(map[string]any{"answer": "string", "status": "string"})
		return req.GetResponse()
	}

	{
		ctx, cancel := testkit.TestContext(t, 5*time.Second)
		defer cancel()
		stream, err := newResponse("delta").Result.GetGenerator(ctx, "delta")
		if err != nil {
			t.Fatalf("delta stream failed: %v", err)
		}
		items := testkit.CollectAny(t, stream, 5*time.Second)
		if len(items) == 0 {
			t.Fatalf("expected delta items")
		}
	}

	{
		ctx, cancel := testkit.TestContext(t, 5*time.Second)
		defer cancel()
		stream, err := newResponse("specific").Result.GetGenerator(ctx, "specific", []string{"tool_calls"})
		if err != nil {
			t.Fatalf("specific stream failed: %v", err)
		}
		items := testkit.CollectAny(t, stream, 5*time.Second)
		if len(items) == 0 {
			t.Fatalf("expected specific tool_calls items")
		}
		if first, ok := items[0].(types.ResponseMessage); !ok || first.Event != types.ResponseEventToolCalls {
			t.Fatalf("unexpected specific item: %#v", items[0])
		}
	}

	{
		ctx, cancel := testkit.TestContext(t, 5*time.Second)
		defer cancel()
		stream, err := newResponse("original").Result.GetGenerator(ctx, "original")
		if err != nil {
			t.Fatalf("original stream failed: %v", err)
		}
		items := testkit.CollectAny(t, stream, 5*time.Second)
		if len(items) < 2 {
			t.Fatalf("expected original delta and done items, got %#v", items)
		}
	}

	{
		ctx, cancel := testkit.TestContext(t, 5*time.Second)
		defer cancel()
		stream, err := newResponse("all").Result.GetGenerator(ctx, "all")
		if err != nil {
			t.Fatalf("all stream failed: %v", err)
		}
		items := testkit.CollectAny(t, stream, 5*time.Second)
		if len(items) == 0 {
			t.Fatalf("expected all stream items")
		}
	}

	{
		ctx, cancel := testkit.TestContext(t, 5*time.Second)
		defer cancel()
		stream, err := newResponse("instant").Result.GetGenerator(ctx, "instant")
		if err != nil {
			t.Fatalf("instant stream failed: %v", err)
		}
		items := testkit.CollectAny(t, stream, 5*time.Second)
		if len(items) == 0 {
			t.Fatalf("expected instant(streaming_parse) items")
		}
		foundAnswerPath := false
		for _, item := range items {
			evt, ok := item.(types.StreamingData)
			if !ok {
				continue
			}
			if evt.Path == "answer" {
				foundAnswerPath = true
				break
			}
		}
		if !foundAnswerPath {
			t.Fatalf("expected instant stream event path=answer, got %#v", items)
		}
	}
}

func TestSessionResizeAndSerializeLoad(t *testing.T) {
	settings := core.NewDefaultSettings(nil)
	settings.Set("session.max_length", 120)
	session := core.NewSession("session-regression", true, settings)

	history := []types.ChatMessage{
		{Role: "user", Content: strings.Repeat("A", 60)},
		{Role: "assistant", Content: strings.Repeat("B", 60)},
		{Role: "user", Content: strings.Repeat("C", 60)},
	}
	session.SetChatHistory(history)

	if len(session.ContextWindow()) >= len(session.FullContext()) {
		t.Fatalf("expected context window trimmed by max_length, full=%d window=%d", len(session.FullContext()), len(session.ContextWindow()))
	}

	jsonPayload, err := session.ToJSON()
	if err != nil {
		t.Fatalf("session ToJSON failed: %v", err)
	}
	yamlPayload, err := session.ToYAML()
	if err != nil {
		t.Fatalf("session ToYAML failed: %v", err)
	}

	loadedFromJSON := core.NewSession("loaded-json", false, settings)
	if err := loadedFromJSON.LoadJSON(jsonPayload); err != nil {
		t.Fatalf("session LoadJSON failed: %v", err)
	}
	if len(loadedFromJSON.FullContext()) != len(session.FullContext()) {
		t.Fatalf("LoadJSON full context mismatch")
	}

	loadedFromYAML := core.NewSession("loaded-yaml", false, settings)
	if err := loadedFromYAML.LoadYAML(yamlPayload); err != nil {
		t.Fatalf("session LoadYAML failed: %v", err)
	}
	if len(loadedFromYAML.ContextWindow()) != len(session.ContextWindow()) {
		t.Fatalf("LoadYAML context window mismatch")
	}
}
