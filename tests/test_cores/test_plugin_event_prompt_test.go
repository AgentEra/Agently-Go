package core_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	entry "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
	"github.com/AgentEra/Agently-Go/agently/types"
	"github.com/AgentEra/Agently-Go/agently/utils"
)

func TestPluginManagerRegistrationLifecycle(t *testing.T) {
	settings := core.NewDefaultSettings(nil)
	manager := core.NewPluginManager(settings, nil, "plugin-manager-regression")

	registered := false
	unregistered := false

	err := manager.Register(core.PluginTypeToolManager, core.PluginSpec{
		Name: "RegressionToolManager",
		Creator: core.ToolManagerCreator(func(_ *utils.Settings) core.ToolManager {
			return nil
		}),
		DefaultSettings: map[string]any{
			"$global": map[string]any{
				"runtime": map[string]any{"show_model_logs": true},
			},
			"$mappings": map[string]any{
				"path_mappings": map[string]any{
					"RegressionTool": "plugins.ToolManager.RegressionToolManager",
				},
			},
			"enabled": true,
		},
		OnRegister: func() { registered = true },
		OnUnregister: func() {
			unregistered = true
		},
	}, true)
	if err != nil {
		t.Fatalf("register plugin failed: %v", err)
	}

	if !registered {
		t.Fatalf("expected register callback to run")
	}
	if got := settings.Get("plugins.ToolManager.activate", "", true); got != "RegressionToolManager" {
		t.Fatalf("unexpected activated plugin: %#v", got)
	}
	if got := settings.Get("runtime.show_model_logs", false, true); got != true {
		t.Fatalf("expected default global setting injected, got %#v", got)
	}
	settings.SetSettings("RegressionTool", map[string]any{"enabled": "mapped"}, false)
	if got := settings.Get("plugins.ToolManager.RegressionToolManager.enabled", nil, true); got != "mapped" {
		t.Fatalf("expected mapped path to work, got %#v", got)
	}

	if err := manager.Unregister(core.PluginTypeToolManager, "RegressionToolManager"); err != nil {
		t.Fatalf("unregister plugin failed: %v", err)
	}
	if !unregistered {
		t.Fatalf("expected unregister callback to run")
	}
}

func TestEventCenterAndMessenger(t *testing.T) {
	center := core.NewEventCenter()
	messenger := center.CreateMessenger("regression", map[string]any{"trace": "core"})

	logCh := make(chan types.EventMessage, 1)
	sysCh := make(chan types.EventMessage, 1)

	center.RegisterHook(types.EventNameLog, func(msg types.EventMessage) {
		logCh <- msg
	}, "log-hook")
	center.RegisterHook(types.EventNameSystem, func(msg types.EventMessage) {
		sysCh <- msg
	}, "sys-hook")

	if err := messenger.Info("hello"); err != nil {
		t.Fatalf("messenger info failed: %v", err)
	}
	select {
	case msg := <-logCh:
		if msg.Content != "hello" {
			t.Fatalf("unexpected log content: %#v", msg.Content)
		}
		if msg.Meta["trace"] != "core" {
			t.Fatalf("expected merged meta trace=core, got %#v", msg.Meta)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("wait log event timeout")
	}

	if err := center.SystemMessage(types.SystemEventTool, map[string]any{"name": "sum"}, nil); err != nil {
		t.Fatalf("system message failed: %v", err)
	}
	select {
	case msg := <-sysCh:
		content, ok := msg.Content.(map[string]any)
		if !ok {
			t.Fatalf("unexpected system content type: %T", msg.Content)
		}
		if content["type"] != types.SystemEventTool {
			t.Fatalf("unexpected system type: %#v", content["type"])
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("wait system event timeout")
	}
}

func TestPromptGeneratorKeyBehaviors(t *testing.T) {
	main := entry.NewAgently()
	prompt := main.CreatePrompt("prompt-regression")

	if _, err := prompt.ToText(); err == nil {
		t.Fatalf("expected empty prompt ToText error")
	}

	prompt.Set("system", "you are test")
	prompt.Set("input", "say hi")
	prompt.Set("output", map[string]any{"answer": "string"})

	text, err := prompt.ToText()
	if err != nil {
		t.Fatalf("ToText failed: %v", err)
	}
	if !strings.Contains(text, "[INPUT]:") {
		t.Fatalf("expected INPUT block in text prompt, got: %s", text)
	}

	messages, err := prompt.ToMessages(core.PromptMessageOptions{})
	if err != nil {
		t.Fatalf("ToMessages failed: %v", err)
	}
	if len(messages) == 0 {
		t.Fatalf("expected non-empty messages")
	}

	schema, err := prompt.ToOutputModelSchema()
	if err != nil {
		t.Fatalf("ToOutputModelSchema failed: %v", err)
	}
	schemaMap, ok := schema.(map[string]any)
	if !ok || schemaMap["answer"] != "string" {
		t.Fatalf("unexpected output schema: %#v", schema)
	}
}

func TestPromptGeneratorStrictRoleOrders(t *testing.T) {
	main := entry.NewAgently()
	prompt := main.CreatePrompt("prompt-strict-orders")
	prompt.Set("chat_history", []any{
		map[string]any{"role": "assistant", "content": "Hi, how can I help you today?"},
		map[string]any{"role": "user", "content": "?"},
	})

	prompt.Set("input", "hi")

	messages, err := prompt.ToMessages(core.PromptMessageOptions{
		RichContent:      false,
		StrictRoleOrders: true,
	})
	if err != nil {
		t.Fatalf("ToMessages strict role orders failed: %v", err)
	}
	expected := []map[string]any{
		{"role": "user", "content": "[CHAT HISTORY]"},
		{"role": "assistant", "content": "Hi, how can I help you today?"},
		{"role": "user", "content": "?"},
		{"role": "assistant", "content": "[User continue input]"},
		{"role": "user", "content": "hi"},
	}
	if len(messages) != len(expected) {
		t.Fatalf("strict role orders message length mismatch: got=%d want=%d, messages=%#v", len(messages), len(expected), messages)
	}
	for i := range expected {
		if fmt.Sprint(messages[i]["role"]) != fmt.Sprint(expected[i]["role"]) || fmt.Sprint(messages[i]["content"]) != fmt.Sprint(expected[i]["content"]) {
			t.Fatalf("strict role orders mismatch at %d: got=%#v want=%#v", i, messages[i], expected[i])
		}
	}
}
