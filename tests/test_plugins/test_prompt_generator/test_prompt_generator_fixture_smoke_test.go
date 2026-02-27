package test_prompt_generator_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	entry "github.com/AgentEra/Agently-Go/agently"
)

type fixture struct {
	PromptData    map[string]any `json:"prompt_data"`
	ExpectedText  string         `json:"expected_text"`
	ExpectedError string         `json:"expected_text_error"`
}

func TestPromptGeneratorFixtureSmoke(t *testing.T) {
	path := fixturePath(t, "prompt_003_full_slots_with_json_output.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture failed: %v", err)
	}
	var f fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("unmarshal fixture failed: %v", err)
	}

	main := entry.NewAgently()
	agent := main.CreateAgent("prompt-generator-fixture-smoke")

	if reqData, ok := f.PromptData[".request"].(map[string]any); ok {
		for key, value := range reqData {
			agent.SetRequestPrompt(key, value)
		}
	}
	if agentData, ok := f.PromptData[".agent"].(map[string]any); ok {
		for key, value := range agentData {
			agent.SetAgentPrompt(key, value)
		}
	}

	text, err := agent.Prompt().ToText()
	if strings.TrimSpace(f.ExpectedError) != "" {
		if err == nil {
			t.Fatalf("expected ToText error from fixture")
		}
		return
	}
	if err != nil {
		t.Fatalf("ToText failed: %v", err)
	}

	if normalize(text) != normalize(f.ExpectedText) {
		t.Fatalf("prompt text mismatch\nexpected:\n%s\nactual:\n%s", normalize(f.ExpectedText), normalize(text))
	}
}

func normalize(input string) string {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func fixturePath(t *testing.T, file string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	for {
		candidate := filepath.Join(dir, "tests", "fixtures", "prompt_parity", file)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("unable to locate fixture file %s", file)
		}
		dir = parent
	}
}
