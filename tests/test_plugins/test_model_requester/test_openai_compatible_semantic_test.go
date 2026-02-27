package modelrequester_test

import (
	"context"
	"testing"
	"time"

	entry "github.com/AgentEra/Agently-Go/agently"
	mr "github.com/AgentEra/Agently-Go/agently/builtins/plugins/model_requester"
	"github.com/AgentEra/Agently-Go/agently/types"
)

func collectMessages(t *testing.T, ch <-chan types.ResponseMessage, timeout time.Duration) []types.ResponseMessage {
	t.Helper()
	out := make([]types.ResponseMessage, 0)
	deadline := time.After(timeout)
	for {
		select {
		case item, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, item)
		case <-deadline:
			t.Fatalf("collect timeout after %s", timeout)
		}
	}
}

func TestOpenAICompatibleGenerateRequestDataChat(t *testing.T) {
	main := entry.NewAgently()
	main.SetSettings("OpenAICompatible", map[string]any{
		"base_url":   "http://127.0.0.1:11434/v1",
		"model":      "qwen2.5:7b",
		"model_type": "chat",
		"stream":     true,
	}, false)

	req := main.CreateRequest("model-request-data")
	req.Input("Hello")
	req.Output(map[string]any{"answer": "string"})

	requester := mr.New(req.Prompt(), req.Settings())
	data, err := requester.GenerateRequestData()
	if err != nil {
		t.Fatalf("GenerateRequestData failed: %v", err)
	}

	if data.RequestURL != "http://127.0.0.1:11434/v1/chat/completions" {
		t.Fatalf("unexpected request url: %s", data.RequestURL)
	}
	if !data.Stream {
		t.Fatalf("expected stream=true")
	}
	if got := data.RequestOpts["model"]; got != "qwen2.5:7b" {
		t.Fatalf("expected model=qwen2.5:7b, got %#v", got)
	}
	if _, ok := data.Data["messages"]; !ok {
		t.Fatalf("expected chat messages payload, got %#v", data.Data)
	}
}

func TestOpenAICompatibleBroadcastResponseSemantic(t *testing.T) {
	main := entry.NewAgently()
	main.SetSettings("OpenAICompatible", map[string]any{
		"model_type": "chat",
	}, false)

	req := main.CreateRequest("broadcast-semantic")
	req.Input("hello")
	req.Output(map[string]any{"answer": "string"})

	requester := mr.New(req.Prompt(), req.Settings())
	source := make(chan types.ResponseMessage, 3)
	source <- types.ResponseMessage{Event: types.ResponseEventOriginalDelta, Data: `{"id":"abc","choices":[{"delta":{"content":"Hel"}}]}`}
	source <- types.ResponseMessage{Event: types.ResponseEventOriginalDelta, Data: `{"id":"abc","choices":[{"delta":{"content":"lo"},"finish_reason":"stop"}],"usage":{"total_tokens":12}}`}
	source <- types.ResponseMessage{Event: types.ResponseEventOriginalDone, Data: "[DONE]"}
	close(source)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stream, err := requester.BroadcastResponse(ctx, source)
	if err != nil {
		t.Fatalf("BroadcastResponse failed: %v", err)
	}

	messages := collectMessages(t, stream, 3*time.Second)
	if len(messages) == 0 {
		t.Fatalf("expected non-empty broadcast messages")
	}

	foundDelta := false
	foundDone := false
	foundMeta := false
	for _, msg := range messages {
		switch msg.Event {
		case types.ResponseEventDelta:
			foundDelta = true
		case types.ResponseEventDone:
			foundDone = true
		case types.ResponseEventMeta:
			foundMeta = true
		}
	}
	if !foundDelta || !foundDone || !foundMeta {
		t.Fatalf("missing semantic events: delta=%v done=%v meta=%v messages=%#v", foundDelta, foundDone, foundMeta, messages)
	}
}
