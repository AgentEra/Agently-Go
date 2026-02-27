package core_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	entry "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/builtins/agent_extensions"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/testkit"
	"github.com/AgentEra/Agently-Go/agently/types"
)

func TestExampleSettingsInheritanceAndOverride(t *testing.T) {
	main := entry.NewAgently()
	main.SetSettings("OpenAICompatible", map[string]any{
		"base_url": "http://127.0.0.1:11434/v1",
		"model":    "qwen2.5:7b",
	}, false)

	agent := main.CreateAgent("example-settings")
	agent.SetSettings("OpenAICompatible", map[string]any{
		"model": "qwen3:latest",
	}, false)

	baseURL := fmt.Sprint(agent.Settings().Get("plugins.ModelRequester.OpenAICompatible.base_url", "", true))
	model := fmt.Sprint(agent.Settings().Get("plugins.ModelRequester.OpenAICompatible.model", "", true))
	if baseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("expected base_url inherited from global settings, got %q", baseURL)
	}
	if model != "qwen3:latest" {
		t.Fatalf("expected model overridden in agent settings, got %q", model)
	}
}

func TestExampleConfigurePromptLoadersAndMappings(t *testing.T) {
	manager := newRegressionPluginManager(func(_ int) []types.ResponseMessage { return nil }, nil)
	agent := agentextensions.NewAgent(manager, core.NewDefaultSettings(nil), "example-configure")

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "prompt.yaml")
	yamlPrompt := `
.agent:
  system: You are ${name}
.request:
  input: Ask ${topic}
  output:
    answer: string
$${agent_key}: ${agent_value}
${request_key}: ${request_value}
`
	if err := os.WriteFile(yamlPath, []byte(yamlPrompt), 0o644); err != nil {
		t.Fatalf("write yaml prompt failed: %v", err)
	}

	mappings := map[string]any{
		"name":          "Go Assistant",
		"topic":         "recursion",
		"agent_key":     "persona",
		"agent_value":   "teacher",
		"request_key":   "extra_note",
		"request_value": "from-yaml",
	}
	if err := agent.LoadYAMLPrompt(yamlPath, mappings, ""); err != nil {
		t.Fatalf("LoadYAMLPrompt(path) failed: %v", err)
	}

	if got := fmt.Sprint(agent.AgentPrompt().Get("system", "", true)); got != "You are Go Assistant" {
		t.Fatalf("unexpected agent system from yaml: %q", got)
	}
	if got := fmt.Sprint(agent.AgentPrompt().Get("persona", "", true)); got != "teacher" {
		t.Fatalf("unexpected mapped agent key from yaml: %q", got)
	}
	if got := fmt.Sprint(agent.Prompt().Get("input", "", true)); got != "Ask recursion" {
		t.Fatalf("unexpected request input from yaml: %q", got)
	}
	if got := fmt.Sprint(agent.Prompt().Get("extra_note", "", true)); got != "from-yaml" {
		t.Fatalf("unexpected mapped request key from yaml: %q", got)
	}

	jsonPrompt := `{
  "prompt_1": {
    ".request": { "input": "first prompt" }
  },
  "prompt_2": {
    ".agent": { "system": "System ${name2}" },
    ".request": {
      "input": "Topic: ${topic2}",
      "output": { "reply": "string" }
    }
  }
}`
	if err := agent.LoadJSONPrompt(jsonPrompt, map[string]any{
		"name2":  "FromJSON",
		"topic2": "TriggerFlow",
	}, "prompt_2"); err != nil {
		t.Fatalf("LoadJSONPrompt(content+key path) failed: %v", err)
	}

	if got := fmt.Sprint(agent.AgentPrompt().Get("system", "", true)); got != "System FromJSON" {
		t.Fatalf("unexpected agent system from json key path: %q", got)
	}
	if got := fmt.Sprint(agent.Prompt().Get("input", "", true)); got != "Topic: TriggerFlow" {
		t.Fatalf("unexpected request input from json key path: %q", got)
	}
	schema, err := agent.Prompt().ToOutputModelSchema()
	if err != nil {
		t.Fatalf("ToOutputModelSchema failed after json load: %v", err)
	}
	schemaMap, ok := schema.(map[string]any)
	if !ok || schemaMap["reply"] != "string" {
		t.Fatalf("unexpected schema after json load: %#v", schema)
	}
}

func TestExampleConfigurePromptRoundTrip(t *testing.T) {
	manager := newRegressionPluginManager(func(_ int) []types.ResponseMessage { return nil }, nil)
	agent := agentextensions.NewAgent(manager, core.NewDefaultSettings(nil), "example-roundtrip")

	agent.System("SYS", core.Always())
	agent.Input("IN")
	agent.Output(map[string]any{"answer": "string"})

	yamlPrompt, err := agent.GetYAMLPrompt()
	if err != nil {
		t.Fatalf("GetYAMLPrompt failed: %v", err)
	}
	jsonPrompt, err := agent.GetJSONPrompt()
	if err != nil {
		t.Fatalf("GetJSONPrompt failed: %v", err)
	}

	fromYAML := agentextensions.NewAgent(manager, core.NewDefaultSettings(nil), "from-yaml")
	if err := fromYAML.LoadYAMLPrompt(yamlPrompt, nil, ""); err != nil {
		t.Fatalf("LoadYAMLPrompt(roundtrip) failed: %v", err)
	}

	fromJSON := agentextensions.NewAgent(manager, core.NewDefaultSettings(nil), "from-json")
	if err := fromJSON.LoadJSONPrompt(jsonPrompt, nil, ""); err != nil {
		t.Fatalf("LoadJSONPrompt(roundtrip) failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	yamlText, err := fromYAML.GetPromptText(ctx)
	if err != nil {
		t.Fatalf("GetPromptText(from yaml) failed: %v", err)
	}
	jsonText, err := fromJSON.GetPromptText(ctx)
	if err != nil {
		t.Fatalf("GetPromptText(from json) failed: %v", err)
	}
	if !strings.Contains(yamlText, "SYS") || !strings.Contains(yamlText, "IN") {
		t.Fatalf("roundtrip yaml prompt text missing key content: %s", yamlText)
	}
	if !strings.Contains(jsonText, "SYS") || !strings.Contains(jsonText, "IN") {
		t.Fatalf("roundtrip json prompt text missing key content: %s", jsonText)
	}
}

func TestExampleResponseSnapshotAndReuse(t *testing.T) {
	requestCount := &atomic.Int32{}
	manager := newRegressionPluginManager(func(call int) []types.ResponseMessage {
		return []types.ResponseMessage{
			{Event: types.ResponseEventDelta, Data: `{"answer":"hel`},
			{Event: types.ResponseEventDelta, Data: `lo","call":` + fmt.Sprint(call) + `}`},
			{Event: types.ResponseEventDone, Data: `{"answer":"hello","call":` + fmt.Sprint(call) + `}`},
			{Event: types.ResponseEventOriginalDone, Data: map[string]any{"call": call}},
			{Event: types.ResponseEventMeta, Data: map[string]any{"call": call}},
		}
	}, requestCount)

	req := core.NewModelRequest(manager, "example-response", core.NewDefaultSettings(nil), nil, nil)
	req.Input("Explain recursion")
	req.Output(map[string]any{"answer": "string", "call": "number"})
	resp := req.GetResponse()

	ctx, cancel := testkit.TestContext(t, 5*time.Second)
	defer cancel()

	text, err := resp.Result.GetText(ctx)
	if err != nil {
		t.Fatalf("GetText failed: %v", err)
	}
	if !strings.Contains(text, "hello") {
		t.Fatalf("unexpected text: %s", text)
	}

	data, err := resp.Result.GetData(ctx, core.GetDataOptions{Type: "parsed"})
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	parsed, ok := data.(map[string]any)
	if !ok || parsed["answer"] == nil {
		t.Fatalf("unexpected parsed data: %#v", data)
	}

	meta, err := resp.Result.GetMeta(ctx)
	if err != nil {
		t.Fatalf("GetMeta failed: %v", err)
	}
	if meta["call"] == nil {
		t.Fatalf("expected meta call, got %#v", meta)
	}

	if requestCount.Load() != 1 {
		t.Fatalf("expected one underlying model request for one response snapshot, got %d", requestCount.Load())
	}
}

func TestExampleChatHistoryPromptOperations(t *testing.T) {
	manager := newRegressionPluginManager(func(_ int) []types.ResponseMessage { return nil }, nil)
	agent := agentextensions.NewAgent(manager, core.NewDefaultSettings(nil), "example-chat-history")

	agent.SetChatHistory([]types.ChatMessage{
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello"},
	})
	agent.AddChatHistory([]types.ChatMessage{
		{Role: "user", Content: "What did I ask?"},
	})

	history, _ := agent.AgentPrompt().Get("chat_history", []any{}, true).([]any)
	if len(history) != 3 {
		t.Fatalf("expected 3 chat history items, got %#v", history)
	}

	agent.ResetChatHistory()
	historyAfterReset, _ := agent.AgentPrompt().Get("chat_history", []any{}, true).([]any)
	if len(historyAfterReset) != 0 {
		t.Fatalf("expected empty chat history after reset, got %#v", historyAfterReset)
	}
}
