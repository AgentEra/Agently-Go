package plugins_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	entry "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/builtins/agent_extensions"
	"github.com/AgentEra/Agently-Go/agently/core"
)

type promptParityFixture struct {
	CaseID                   string                 `json:"case_id"`
	Mode                     string                 `json:"mode"`
	Settings                 map[string]any         `json:"settings"`
	MessageOptions           map[string]any         `json:"message_options"`
	PromptData               map[string]any         `json:"prompt_data"`
	Configure                map[string]any         `json:"configure"`
	ExpectedText             string                 `json:"expected_text"`
	ExpectedTextError        string                 `json:"expected_text_error"`
	ExpectedMessages         any                    `json:"expected_messages"`
	ExpectedMessagesError    string                 `json:"expected_messages_error"`
	ExpectedOutputSchema     any                    `json:"expected_output_schema"`
	ExpectedSerializableData map[string]any         `json:"expected_serializable_prompt"`
	Raw                      map[string]interface{} `json:"-"`
}

func TestPromptParityFixtureSuite(t *testing.T) {
	fixtures := loadPromptParityFixtures(t)
	if len(fixtures) == 0 {
		t.Fatalf("no prompt parity fixtures found")
	}

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.CaseID, func(t *testing.T) {
			main := entry.NewAgently()
			agent := main.CreateAgent("prompt-parity-" + fixture.CaseID)
			applySettings(agent, fixture.Settings)

			switch fixture.Mode {
			case "direct":
				applyPromptData(agent, fixture.PromptData)
			case "configure":
				applyConfigurePrompt(t, agent, fixture.Configure)
			case "configure_roundtrip_yaml":
				applyPromptData(agent, fixture.PromptData)
				yamlPrompt, err := agent.GetYAMLPrompt()
				if err != nil {
					t.Fatalf("GetYAMLPrompt failed: %v", err)
				}
				agent = main.CreateAgent("prompt-parity-rt-yaml-" + fixture.CaseID)
				applySettings(agent, fixture.Settings)
				if err := agent.LoadYAMLPrompt(yamlPrompt, nil, ""); err != nil {
					t.Fatalf("LoadYAMLPrompt roundtrip failed: %v", err)
				}
			case "configure_roundtrip_json":
				applyPromptData(agent, fixture.PromptData)
				jsonPrompt, err := agent.GetJSONPrompt()
				if err != nil {
					t.Fatalf("GetJSONPrompt failed: %v", err)
				}
				agent = main.CreateAgent("prompt-parity-rt-json-" + fixture.CaseID)
				applySettings(agent, fixture.Settings)
				if err := agent.LoadJSONPrompt(jsonPrompt, nil, ""); err != nil {
					t.Fatalf("LoadJSONPrompt roundtrip failed: %v", err)
				}
			default:
				t.Fatalf("unknown fixture mode %q", fixture.Mode)
			}

			validateSerializablePrompt(t, fixture, agent)
			validatePromptText(t, fixture, agent)
			validatePromptMessages(t, fixture, agent)
			validateOutputSchema(t, fixture, agent)
		})
	}
}

func validateSerializablePrompt(t *testing.T, fixture promptParityFixture, agent *agentextensions.Agent) {
	t.Helper()
	actualAgent, err := agent.AgentPrompt().ToSerializablePromptData(false)
	if err != nil {
		t.Fatalf("ToSerializablePromptData(agent) failed: %v", err)
	}
	actualRequest, err := agent.Prompt().ToSerializablePromptData(false)
	if err != nil {
		t.Fatalf("ToSerializablePromptData(request) failed: %v", err)
	}
	actual := map[string]any{
		".agent":   actualAgent,
		".request": actualRequest,
	}

	expect := fixture.ExpectedSerializableData
	if !deepEqualCanonical(expect, actual) {
		t.Fatalf("serializable prompt mismatch\nexpected:\n%s\nactual:\n%s", prettyJSON(expect), prettyJSON(actual))
	}
}

func validatePromptText(t *testing.T, fixture promptParityFixture, agent *agentextensions.Agent) {
	t.Helper()
	actualText, err := agent.Prompt().ToText()
	expectedErr := unquotePythonError(fixture.ExpectedTextError)
	if expectedErr != "" {
		if err == nil {
			t.Fatalf("expected ToText error %q, got nil", expectedErr)
		}
		if !strings.Contains(err.Error(), expectedErr) {
			t.Fatalf("ToText error mismatch: expected contains %q, got %q", expectedErr, err.Error())
		}
		return
	}
	if err != nil {
		t.Fatalf("ToText failed: %v", err)
	}

	expected := normalizePromptText(fixture.ExpectedText)
	actual := normalizePromptText(actualText)
	if expected != actual {
		t.Fatalf("ToText mismatch\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func validatePromptMessages(t *testing.T, fixture promptParityFixture, agent *agentextensions.Agent) {
	t.Helper()
	options := core.PromptMessageOptions{
		RichContent:      boolValue(fixture.MessageOptions["rich_content"], false),
		StrictRoleOrders: boolValue(fixture.MessageOptions["strict_role_orders"], true),
	}
	if roleMapping, ok := fixture.MessageOptions["role_mapping"].(map[string]any); ok {
		options.RoleMapping = map[string]string{}
		for k, v := range roleMapping {
			options.RoleMapping[k] = fmt.Sprint(v)
		}
	}

	actual, err := agent.Prompt().ToMessages(options)
	expectedErr := unquotePythonError(fixture.ExpectedMessagesError)
	if expectedErr != "" {
		if err == nil {
			t.Fatalf("expected ToMessages error %q, got nil", expectedErr)
		}
		if !strings.Contains(err.Error(), expectedErr) {
			t.Fatalf("ToMessages error mismatch: expected contains %q, got %q", expectedErr, err.Error())
		}
		return
	}
	if err != nil {
		t.Fatalf("ToMessages failed: %v", err)
	}

	expected := fixture.ExpectedMessages
	if !deepEqualCanonical(expected, actual) {
		t.Fatalf("ToMessages mismatch\nexpected:\n%s\nactual:\n%s", prettyJSON(expected), prettyJSON(actual))
	}
}

func validateOutputSchema(t *testing.T, fixture promptParityFixture, agent *agentextensions.Agent) {
	t.Helper()
	actual, err := agent.Prompt().ToOutputModelSchema()
	if fixture.ExpectedOutputSchema == nil {
		// empty prompt case may return output error; both are acceptable for nil expected output.
		if err == nil && actual != nil {
			t.Fatalf("expected nil output schema, got %#v", actual)
		}
		return
	}
	if err != nil {
		t.Fatalf("ToOutputModelSchema failed: %v", err)
	}
	if !deepEqualCanonical(fixture.ExpectedOutputSchema, actual) {
		t.Fatalf("output schema mismatch\nexpected:\n%s\nactual:\n%s", prettyJSON(fixture.ExpectedOutputSchema), prettyJSON(actual))
	}
}

func applySettings(agent *agentextensions.Agent, settings map[string]any) {
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		agent.SetSettings(key, settings[key])
	}
}

func applyPromptData(agent *agentextensions.Agent, promptData map[string]any) {
	if agentData, ok := promptData[".agent"].(map[string]any); ok {
		keys := make([]string, 0, len(agentData))
		for k := range agentData {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			agent.SetAgentPrompt(key, agentData[key])
		}
	}
	if reqData, ok := promptData[".request"].(map[string]any); ok {
		keys := make([]string, 0, len(reqData))
		for k := range reqData {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			agent.SetRequestPrompt(key, reqData[key])
		}
	}
}

func applyConfigurePrompt(t *testing.T, agent *agentextensions.Agent, cfg map[string]any) {
	t.Helper()
	format := fmt.Sprint(cfg["format"])
	content := fmt.Sprint(cfg["content"])
	keyPath := fmt.Sprint(cfg["prompt_key_path"])
	if keyPath == "<nil>" {
		keyPath = ""
	}

	var mappings map[string]any
	if raw, ok := cfg["mappings"].(map[string]any); ok {
		mappings = raw
	} else {
		mappings = map[string]any{}
	}

	var err error
	if format == "yaml" {
		err = agent.LoadYAMLPrompt(content, mappings, keyPath)
	} else {
		err = agent.LoadJSONPrompt(content, mappings, keyPath)
	}
	if err != nil {
		t.Fatalf("configure load failed(format=%s): %v", format, err)
	}
}

func loadPromptParityFixtures(t *testing.T) []promptParityFixture {
	t.Helper()
	base := fixtureDir(t)
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("read fixture dir failed: %v", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	out := make([]promptParityFixture, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(base, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read fixture file %s failed: %v", path, err)
		}
		var f promptParityFixture
		if err := json.Unmarshal(raw, &f); err != nil {
			t.Fatalf("unmarshal fixture file %s failed: %v", path, err)
		}
		if f.CaseID == "" {
			t.Fatalf("fixture file %s missing case_id", path)
		}
		out = append(out, f)
	}
	return out
}

func fixtureDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	for {
		candidate := filepath.Join(dir, "tests", "fixtures", "prompt_parity")
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("unable to locate fixture directory from %s", dir)
		}
		dir = parent
	}
}

func boolValue(value any, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	switch typed := value.(type) {
	case bool:
		return typed
	default:
		text := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
		if text == "true" {
			return true
		}
		if text == "false" {
			return false
		}
		return defaultValue
	}
}

func unquotePythonError(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if unquoted, err := strconv.Unquote(text); err == nil {
		return unquoted
	}
	return strings.Trim(text, "\"'")
}

func normalizePromptText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	for idx := range lines {
		lines[idx] = strings.TrimRight(lines[idx], " \t")
	}
	return strings.Join(lines, "\n")
}

func deepEqualCanonical(expected any, actual any) bool {
	return prettyJSON(expected) == prettyJSON(actual)
}

func prettyJSON(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("<marshal error: %v> %#v", err, value)
	}
	var obj any
	if err := json.Unmarshal(b, &obj); err != nil {
		return string(b)
	}
	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return string(b)
	}
	return string(pretty)
}
