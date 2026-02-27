# Agently-Go

Build production-grade AI apps in Go with stable outputs and maintainable workflows.

[English](README.md) | [Chinese](README_CN.md)

Python baseline project: [AgentEra/Agently](https://github.com/AgentEra/Agently)  
Target Go repository: [AgentEra/Agently-Go](https://github.com/AgentEra/Agently-Go)

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8.svg)](go.mod)
[![GitHub Stars](https://img.shields.io/github/stars/AgentEra/Agently-Go.svg?style=social)](https://github.com/AgentEra/Agently-Go/stargazers)

## Quick Links

- Repository structure and Python mapping: [`docs/project-structure.md`](docs/project-structure.md)
- Examples root: [`examples/`](examples)
- Regression tests root: [`tests/`](tests)
- Release readiness script: [`scripts/release-ready.sh`](scripts/release-ready.sh)

## Why Agently-Go

| Common production issue | Agently-Go approach |
|---|---|
| Output schema drift causes parser failures | Contract-first output with `Output(...)` + `EnsureKeys` |
| Prompt and request behavior become hard to reason about | Clear core pipeline: `Prompt -> Request -> Response` |
| Streaming UX is difficult to structure | Supports `delta`, `specific`, and `instant` streaming |
| Workflow logic becomes tangled | Signal-driven `TriggerFlow` operators and routing |
| Multi-turn behavior is unstable | Built-in session extension and context management |

## Core Capabilities

- Core runtime: `agently/core`
- Signal-driven orchestration: `agently/triggerflow`
- Default plugin chain:
  - `PromptGenerator`
  - `ModelRequester (OpenAICompatible)`
  - `ResponseParser`
  - `ToolManager`
- Default agent extensions:
  - `Session`
  - `Tool`
  - `ConfigurePrompt`
  - `KeyWaiter`
  - `AutoFunc`

## Quickstart

### Install

```bash
go get github.com/AgentEra/Agently-Go@v1.0.0
```

### Minimal agent

```go
package main

import (
	"fmt"

	agently "github.com/AgentEra/Agently-Go/agently"
)

func main() {
	agentlyApp := agently.NewAgently()
	agentlyApp.SetSettings("OpenAICompatible", map[string]any{
		"base_url": "http://127.0.0.1:11434/v1",
		"model":    "qwen2.5:7b",
	})

	text, err := agentlyApp.
		CreateAgent("hello").
		Input("Introduce recursion in one sentence.").
		GetText()
	if err != nil {
		panic(err)
	}

	fmt.Println(text)
}
```

## More Examples

### Structured output with ensure keys

```go
result, err := agentlyApp.
	CreateAgent("ensure-keys").
	Input("Explain recursion and provide two practical tips.").
	Output(map[string]any{
		"definition": "string",
		"tips":       []any{"string"},
	}).
	GetDataObject(core.GetDataOptions{
		EnsureKeys: []string{"definition", "tips[*]"},
		KeyStyle:   "dot",
		MaxRetries: 2,
	})
if err != nil {
	panic(err)
}
fmt.Printf("%#v\n", result)
```

### Structured streaming (instant)

```go
stream, err := agentlyApp.
	CreateAgent("streaming").
	Input("Explain recursion with a short definition and two tips.").
	Output(map[string]any{
		"definition": "string",
		"tips":       []any{"string"},
	}).
	GetGenerator("instant")
if err != nil {
	panic(err)
}

for item := range stream {
	evt, ok := item.(types.StreamingData)
	if !ok || evt.Delta == "" {
		continue
	}
	fmt.Printf("path=%s delta=%q\n", evt.Path, evt.Delta)
}
```

### TriggerFlow (signal-driven)

```go
flow := triggerflow.New(nil, "demo")
flow.To(triggerflow.Handler(func(data *triggerflow.EventData) (any, error) {
	return fmt.Sprintf("Hello, %v", data.Value), nil
})).End()

result, err := flow.Start("Agently", triggerflow.WithRunTimeout(5*time.Second))
if err != nil {
	panic(err)
}
fmt.Println(result)
```

## Example Matrix

| Category | Example | Run |
|---|---|---|
| Basic core API | `examples/basic/core_api_basics` | `go run ./examples/basic/core_api_basics` |
| Ensure keys | `examples/basic/ensure_keys_in_output` | `go run ./examples/basic/ensure_keys_in_output` |
| Streaming modes | `examples/basic/response_and_streaming` | `go run ./examples/basic/response_and_streaming` |
| Session modes | `examples/basic/session_modes` | `go run ./examples/basic/session_modes` |
| Configure prompt | `examples/configure_prompt/loaders_and_roundtrip` | `go run ./examples/configure_prompt/loaders_and_roundtrip` |
| Prompt generation | `examples/prompt_generation/text_messages_schema` | `go run ./examples/prompt_generation/text_messages_schema` |
| Tool usage | `examples/tools_using/tool_agent` | `go run ./examples/tools_using/tool_agent` |
| TriggerFlow basics | `examples/trigger_flow/signal_basics` | `go run ./examples/trigger_flow/signal_basics` |
| Step-by-step core | `examples/step_by_step/core_01_to_08` | `go run ./examples/step_by_step/core_01_to_08` |
| Applied case | `examples/applied_cases/id_code_analyst` | `go run ./examples/applied_cases/id_code_analyst` |

## Validation

Default test entrypoint includes online regression tests:

```bash
go test ./...
```

Useful scripts:

```bash
./scripts/test-offline.sh
./scripts/test-online.sh
./scripts/test-examples.sh
./scripts/verify-full-replication.sh
./scripts/release-ready.sh
```

Online defaults:

- `OLLAMA_BASE_URL=http://127.0.0.1:11434/v1`
- `OLLAMA_MODEL=qwen2.5:7b`

## Release Scope (v1.0.0)

Included:

- `core`
- `triggerflow`
- default plugins and extensions
- semantic regression suite and parity fixtures

Tracked but out of v1 blocking scope:

- `Storage/AsyncStorage`
- `PythonSandbox`
- integrations (`fastapi/chromadb/mcp/vlm_support`)
- built-in tools (`search/browse/cmd`)

## Repository Layout

- `agently/`: framework source (`core`, `triggerflow`, `builtins`, `utils`, `types`)
- `examples/`: runnable scenario examples
- `tests/`: semantic regression and online/offline suites
- `docs/`: public project docs
- `scripts/`: automation and validation scripts

## License

Apache-2.0. See [LICENSE](LICENSE).

## Acknowledgements

Special thanks to **GPT-5.3-Codex**.  
This migration required deep parity analysis, extensive refactoring, and large-scale validation work.  
GPT-5.3-Codex made what felt impossible become possible.
