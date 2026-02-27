package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	agently "github.com/AgentEra/Agently-Go/agently"
	"github.com/AgentEra/Agently-Go/agently/core"
)

const requestTimeout = 90 * time.Second

func configureMain() *agently.Main {
	baseURL := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434/v1"
	}
	model := strings.TrimSpace(os.Getenv("OLLAMA_MODEL"))
	if model == "" {
		model = "qwen2.5:7b"
	}

	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url":   baseURL,
		"model":      model,
		"model_type": "chat",
	})
	agentlyApp.SetSettings("debug", false)
	return agentlyApp
}

func evaluateCodeExpression(code string, expr string) (any, error) {
	e := strings.TrimSpace(expr)
	e = strings.Trim(e, "`")

	switch {
	case e == "len(code_string)":
		return len(code), nil
	case e == "(\"_\" in code_string)" || e == "('_' in code_string)" || e == "_ in code_string":
		return strings.Contains(code, "_"), nil
	case strings.HasPrefix(e, "code_string[") && strings.HasSuffix(e, "]"):
		body := strings.TrimSuffix(strings.TrimPrefix(e, "code_string["), "]")
		if strings.Contains(body, ":") {
			parts := strings.SplitN(body, ":", 2)
			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid slice start in expr %q: %w", expr, err)
			}
			end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid slice end in expr %q: %w", expr, err)
			}
			if start < 0 || end < start || end > len(code) {
				return nil, fmt.Errorf("slice out of range in expr %q for code length %d", expr, len(code))
			}
			return code[start:end], nil
		}
		index, err := strconv.Atoi(strings.TrimSpace(body))
		if err != nil {
			return nil, fmt.Errorf("invalid index in expr %q: %w", expr, err)
		}
		if index < 0 || index >= len(code) {
			return nil, fmt.Errorf("index out of range in expr %q for code length %d", expr, len(code))
		}
		return string(code[index]), nil
	default:
		return nil, fmt.Errorf("unsupported expression: %q", expr)
	}
}

func toStringMap(value any) map[string]string {
	out := map[string]string{}
	typed, ok := value.(map[string]any)
	if !ok {
		return out
	}
	for k, v := range typed {
		out[k] = strings.TrimSpace(fmt.Sprint(v))
	}
	return out
}

func main() {
	agentlyApp := configureMain()
	agent := agentlyApp.CreateAgent("id-code-analyst")

	codeString := "21553270020250017013_001"
	rule := `
Extract various parameters from the user-provided code:
1. Code length
2. Characters from position 6 to 9
3. First character of the code
4. Whether it contains an underscore (boolean)
`

	planRaw, err := agent.
		Input(map[string]any{
			"code_string": codeString,
			"rule":        rule,
		}).
		Instruct("Based on {rule}, provide output keys and Python expressions to extract each value from {code_string}. Use only code_string variable.").
		Output(map[string]any{
			"output_keys": map[string]any{
				"<output_key_name>": "string",
			},
			"output_method_dict": map[string]any{
				"<output_key_name>": "string",
			},
		}).
		GetData(
			core.GetDataOptions{
				Type:               "parsed",
				EnsureKeys:         []string{"output_keys", "output_method_dict"},
				MaxRetries:         2,
				RaiseEnsureFailure: true,
			},
			core.WithTimeout(requestTimeout),
		)
	if err != nil {
		panic(err)
	}

	plan, _ := planRaw.(map[string]any)
	outputKeys := toStringMap(plan["output_keys"])
	methods := toStringMap(plan["output_method_dict"])

	results := map[string]any{}
	for key, expr := range methods {
		value, evalErr := evaluateCodeExpression(codeString, expr)
		if evalErr != nil {
			results[key] = fmt.Sprintf("eval_error: %v", evalErr)
			continue
		}
		label := key
		if mapped, ok := outputKeys[key]; ok && strings.TrimSpace(mapped) != "" {
			label = mapped
		}
		results[label] = value
	}

	fmt.Printf("PLAN:\n%#v\n", plan)
	fmt.Printf("\nRESULT:\n%#v\n", results)
}
