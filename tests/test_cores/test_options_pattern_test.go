package core_test

import (
	"strings"
	"testing"

	entry "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
)

func TestFunctionalOptionsPatternForRequestAndAgent(t *testing.T) {
	main := entry.NewAgently()

	req := main.CreateRequest("options-request")
	req.Input("Hello ${name}", core.WithMappings(map[string]any{"name": "Agently"}))
	req.Output(map[string]any{"answer": "string"})

	input := req.Prompt().Get("input", "", true)
	if input != "Hello Agently" {
		t.Fatalf("request functional options mapping failed, got %#v", input)
	}

	agent := main.CreateAgent("options-agent")
	agent.System("You are ${role}", core.WithMappings(map[string]any{"role": "assistant"}), core.Always())
	agent.Input("Say hello", core.RequestOnly())
	agent.Output(map[string]any{"answer": "string"})

	agentSystem := agent.AgentPrompt().Get("system", "", true)
	if agentSystem != "You are assistant" {
		t.Fatalf("agent always+mapping failed, got %#v", agentSystem)
	}
	requestInput := agent.Prompt().Get("input", "", true)
	if requestInput != "Say hello" {
		t.Fatalf("agent request-only input failed, got %#v", requestInput)
	}

	text, err := agent.GetPromptText()
	if err != nil {
		t.Fatalf("GetPromptText failed: %v", err)
	}
	if !strings.Contains(text, "You are assistant") {
		t.Fatalf("prompt text missing expected system content: %s", text)
	}
}

func TestLegacyArgsRemainCompatible(t *testing.T) {
	main := entry.NewAgently()
	agent := main.CreateAgent("legacy-compatible")

	// legacy style: (prompt, mappings, always)
	agent.System("Legacy ${name}", map[string]any{"name": "ok"}, true)
	agent.Input("Legacy input")

	agentSystem := agent.AgentPrompt().Get("system", "", true)
	if agentSystem != "Legacy ok" {
		t.Fatalf("legacy compatibility broken for agent.System, got %#v", agentSystem)
	}
	requestInput := agent.Prompt().Get("input", "", true)
	if requestInput != "Legacy input" {
		t.Fatalf("legacy compatibility broken for agent.Input, got %#v", requestInput)
	}

	req := main.CreateRequest("legacy-request")
	// legacy style: (prompt, mappings)
	req.Input("Hi ${name}", map[string]any{"name": "legacy"})
	if got := req.Prompt().Get("input", "", true); got != "Hi legacy" {
		t.Fatalf("legacy compatibility broken for request.Input, got %#v", got)
	}
}
