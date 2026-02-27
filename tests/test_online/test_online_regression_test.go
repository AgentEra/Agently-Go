package online_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	entry "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/testkit"
	"github.com/AgentEra/Agently-Go/agently/triggerflow"
	"github.com/AgentEra/Agently-Go/agently/types"
)

func newOnlineMain(t *testing.T) *entry.Main {
	t.Helper()
	testkit.RequireOllamaHealth(t)
	main := entry.NewAgently()
	testkit.ApplyOllamaDefaults(func(key string, value any, autoLoadEnv bool) {
		main.SetSettings(key, value, autoLoadEnv)
	})
	return main
}

func TestOpenAICompatibleStreamingPath(t *testing.T) {
	main := newOnlineMain(t)
	request := main.CreateRequest("online-stream")
	request.Input("Reply with exactly OK.")

	ctx, cancel := testkit.TestContext(t, 90*time.Second)
	defer cancel()
	stream, err := request.GetGenerator(ctx, "delta")
	if err != nil {
		t.Fatalf("delta stream failed: %v", err)
	}
	items := testkit.CollectAny(t, stream, 90*time.Second)
	if len(items) == 0 {
		t.Fatalf("expected non-empty streaming response (%s)", testkit.OnlineConfigSummary())
	}
}

func TestOnlineInstantParserAndEnsureKeys(t *testing.T) {
	main := newOnlineMain(t)
	request := main.CreateRequest("online-json")
	request.Input("Give a concise answer and identify the answer language.")
	request.Output(map[string]any{"answer": "string", "language": "string"})

	response := request.GetResponse()
	ctx, cancel := testkit.TestContext(t, 120*time.Second)
	defer cancel()

	instantStream, err := response.Result.GetGenerator(ctx, "instant")
	if err != nil {
		t.Fatalf("instant stream failed: %v", err)
	}
	instantItems := testkit.CollectAny(t, instantStream, 120*time.Second)
	if len(instantItems) == 0 {
		t.Fatalf("expected instant(streaming_parse) items")
	}

	data, err := response.Result.GetData(ctx, core.GetDataOptions{Type: "parsed", EnsureKeys: []string{"answer", "language"}, MaxRetries: 2})
	if err != nil {
		t.Fatalf("GetData(parsed+ensure_keys) failed: %v", err)
	}
	parsed, ok := data.(map[string]any)
	if !ok {
		t.Fatalf("expected parsed map, got %T", data)
	}
	if parsed["answer"] == nil || parsed["language"] == nil {
		t.Fatalf("parsed map missing keys: %#v", parsed)
	}
}

func TestOnlineConcurrentResponsesIndependence(t *testing.T) {
	main := newOnlineMain(t)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	type result struct {
		data map[string]any
		err  error
	}
	results := make([]result, 2)
	inputs := []string{
		"Respond with token A1 in the answer.",
		"Respond with token B2 in the answer.",
	}

	var wg sync.WaitGroup
	for i := range inputs {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := main.CreateRequest(fmt.Sprintf("parallel-%d", i))
			req.Input(inputs[i])
			req.Output(map[string]any{"answer": "string"})
			data, err := req.GetData(ctx, core.GetDataOptions{Type: "parsed", EnsureKeys: []string{"answer"}, MaxRetries: 2})
			if err != nil {
				results[i] = result{err: err}
				return
			}
			parsed, _ := data.(map[string]any)
			results[i] = result{data: parsed}
		}()
	}
	wg.Wait()

	for i, item := range results {
		if item.err != nil {
			t.Fatalf("parallel response %d failed: %v", i, item.err)
		}
		if item.data["answer"] == nil {
			t.Fatalf("parallel response %d missing answer: %#v", i, item.data)
		}
	}
}

func TestSessionAndToolExtensionOnline(t *testing.T) {
	main := newOnlineMain(t)
	agent := main.CreateAgent("online-ext")
	agent.ActivateSession("online-session")
	agent.Options(map[string]any{"temperature": 0}, true)

	if err := agent.RegisterTool(types.ToolInfo{
		Name:   "sum",
		Desc:   "sum two integers",
		Kwargs: map[string]any{"a": "number", "b": "number"},
	}, func(kwargs map[string]any) (any, error) {
		return kwargs["a"], nil
	}); err != nil {
		t.Fatalf("register tool failed: %v", err)
	}
	toolList := agent.Tool().Manager().GetToolList([]string{"agent-" + agent.Name()})
	if len(toolList) == 0 {
		t.Fatalf("expected tool extension to tag tool for agent")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	agent.Input("What is 12 + 34? Use the available tool if needed, then explain briefly.")
	agent.Output(map[string]any{"answer": "string"})
	if _, err := agent.GetData(ctx, core.GetDataOptions{Type: "parsed", EnsureKeys: []string{"answer"}, MaxRetries: 4}); err != nil {
		t.Fatalf("online tool extension request failed: %v", err)
	}

	agent.Input("What was the previous calculation result? Provide a one-sentence follow-up answer.")
	agent.Output(map[string]any{"answer": "string"})
	if _, err := agent.GetData(ctx, core.GetDataOptions{Type: "parsed", EnsureKeys: []string{"answer"}, MaxRetries: 4}); err != nil {
		t.Fatalf("online session extension second request failed: %v", err)
	}
}

func TestTriggerFlowAndAgentOnlineIntegration(t *testing.T) {
	main := newOnlineMain(t)
	agent := main.CreateAgent("online-flow-agent")
	flow := triggerflow.New(nil, "online-flow")

	flow.When("START", "").To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
		req := agent.CreateTempRequest()
		req.Input(fmt.Sprintf("Answer this question briefly: %v", data.Value))
		req.Output(map[string]any{"answer": "string"})

		parsed, err := req.GetData(context.Background(), core.GetDataOptions{Type: "parsed", EnsureKeys: []string{"answer"}, MaxRetries: 2})
		if err != nil {
			return nil, err
		}
		_ = data.PutIntoStream(parsed)
		_ = data.StopStream()
		return parsed, nil
	}), false, "agent-llm").End()

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	stream, err := flow.GetRuntimeStream(ctx, "Explain Go language.", triggerflow.WithRunTimeout(120*time.Second))
	if err != nil {
		t.Fatalf("runtime stream from triggerflow failed: %v", err)
	}
	streamItems := make([]any, 0)
	for item := range stream {
		streamItems = append(streamItems, item)
	}
	if len(streamItems) == 0 {
		t.Fatalf("expected runtime stream item from triggerflow+agent")
	}

	result, err := flow.Start("Explain concurrency.", triggerflow.WithRunTimeout(120*time.Second))
	if err != nil {
		t.Fatalf("triggerflow+agent start failed: %v", err)
	}
	parsed, ok := result.(map[string]any)
	if !ok || parsed["answer"] == nil {
		t.Fatalf("unexpected triggerflow+agent result: %#v", result)
	}
}
