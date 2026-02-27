package extensions_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AgentEra/Agently-Go/agently/builtins/agent_extensions"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
)

func TestExtensionsCoreBehaviors(t *testing.T) {
	script := func(call int) []types.ResponseMessage {
		_ = call
		return []types.ResponseMessage{
			{Event: types.ResponseEventDelta, Data: `{"answer":"o`},
			{Event: types.ResponseEventDelta, Data: `k","num":2}`},
			{Event: types.ResponseEventDone, Data: `{"answer":"ok","num":2}`},
			{Event: types.ResponseEventOriginalDone, Data: map[string]any{"done": true}},
		}
	}
	manager := newRegressionPluginManager(script, nil)
	agent := agentextensions.NewAgent(manager, core.NewDefaultSettings(nil), "ext-regression")

	agent.ActivateSession("reg-1")
	agent.SetChatHistory([]types.ChatMessage{{Role: "user", Content: "hello"}})
	history, _ := agent.AgentPrompt().Get("chat_history", []any{}, true).([]any)
	if len(history) == 0 {
		t.Fatalf("expected session extension to sync chat_history into agent prompt")
	}

	err := agent.RegisterTool(types.ToolInfo{
		Name:   "sum",
		Desc:   "sum two numbers",
		Kwargs: map[string]any{"a": "number", "b": "number"},
	}, func(kwargs map[string]any) (any, error) {
		return kwargs["a"], nil
	})
	if err != nil {
		t.Fatalf("register tool failed: %v", err)
	}
	toolList := agent.Tool().Manager().GetToolList([]string{"agent-" + agent.Name()})
	if len(toolList) == 0 {
		t.Fatalf("expected tagged tool list for agent")
	}

	err = agent.LoadJSONPrompt(`{".agent":{"system":"SYS ${name}"},".request":{"input":"INPUT ${name}"}}`, map[string]any{"name": "Go"}, "")
	if err != nil {
		t.Fatalf("LoadJSONPrompt failed: %v", err)
	}
	if got := agent.AgentPrompt().Get("system", "", true); got != "SYS Go" {
		t.Fatalf("configure prompt did not update agent system, got %#v", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	agent.Input("trigger key waiter")
	agent.Output(map[string]any{"answer": "string", "num": "number"})
	keyValue, err := agent.GetKeyResult(ctx, "answer", false)
	if err != nil {
		t.Fatalf("GetKeyResult failed: %v", err)
	}
	if fmt.Sprint(keyValue) == "" {
		t.Fatalf("expected key waiter result")
	}

	auto := agent.AutoFunc("return answer and num", map[string]any{"answer": "string", "num": "number"})
	autoResult, err := auto(ctx, map[string]any{"input": "go"})
	if err != nil {
		t.Fatalf("AutoFunc failed: %v", err)
	}
	parsed, ok := autoResult.(map[string]any)
	if !ok || parsed["answer"] == nil {
		t.Fatalf("unexpected auto func result: %#v", autoResult)
	}

	agent.Input("session finally check")
	agent.Output(map[string]any{"answer": "string"})
	if _, err := agent.GetData(ctx, core.GetDataOptions{Type: "parsed"}); err != nil {
		t.Fatalf("agent GetData failed: %v", err)
	}
	updatedHistory, _ := agent.AgentPrompt().Get("chat_history", []any{}, true).([]any)
	if len(updatedHistory) == 0 {
		t.Fatalf("expected session finally to write back history")
	}
}

func TestSessionExtensionInputReplyKeyFiltering(t *testing.T) {
	script := func(call int) []types.ResponseMessage {
		_ = call
		done := `{"answer":{"text":"done"},"score":99,"ignored":"x"}`
		return []types.ResponseMessage{
			{Event: types.ResponseEventDelta, Data: done},
			{Event: types.ResponseEventDone, Data: done},
			{Event: types.ResponseEventOriginalDone, Data: map[string]any{"done": true}},
		}
	}
	manager := newRegressionPluginManager(script, nil)
	agent := agentextensions.NewAgent(manager, core.NewDefaultSettings(nil), "session-keys")
	agent.ActivateSession("session-keys")
	agent.System("base-system", core.Always())
	agent.SetSettings("session.input_keys", []any{"city", ".agent.system", "input.code"})
	agent.SetSettings("session.reply_keys", []any{"answer.text", "score"})

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	agent.Input(map[string]any{"city": "Shanghai", "code": 200, "ignored": true})
	agent.Output(map[string]any{"answer": map[string]any{"text": "string"}, "score": "number"})
	if _, err := agent.GetData(ctx, core.GetDataOptions{Type: "parsed"}); err != nil {
		t.Fatalf("session key filtering request failed: %v", err)
	}

	chatHistory, _ := agent.AgentPrompt().Get("chat_history", []any{}, true).([]any)
	if len(chatHistory) < 2 {
		t.Fatalf("expected at least user/assistant history entries, got %#v", chatHistory)
	}
	userContent := ""
	assistantContent := ""
	for _, item := range chatHistory {
		msg, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role := fmt.Sprint(msg["role"])
		if role == "user" {
			userContent = fmt.Sprint(msg["content"])
		}
		if role == "assistant" {
			assistantContent = fmt.Sprint(msg["content"])
		}
	}
	if !strings.Contains(userContent, "[city]:") || !strings.Contains(userContent, "Shanghai") {
		t.Fatalf("user keyed content missing city: %s", userContent)
	}
	if !strings.Contains(userContent, "[.agent.system]:") || !strings.Contains(userContent, "base-system") {
		t.Fatalf("user keyed content missing .agent.system: %s", userContent)
	}
	if !strings.Contains(userContent, "[input.code]:") || !strings.Contains(userContent, "200") {
		t.Fatalf("user keyed content missing input.code: %s", userContent)
	}
	if !strings.Contains(assistantContent, "[answer.text]:") || !strings.Contains(assistantContent, "done") {
		t.Fatalf("assistant keyed content missing answer.text: %s", assistantContent)
	}
	if !strings.Contains(assistantContent, "[score]:") || !strings.Contains(assistantContent, "99") {
		t.Fatalf("assistant keyed content missing score: %s", assistantContent)
	}
}
