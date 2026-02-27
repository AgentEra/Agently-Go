# Agently-Go

用 Go 语言构建稳定输出、可维护工作流的生产级 AI 应用。

[English](README.md) | [中文](README_CN.md)

Python 基线项目：[AgentEra/Agently](https://github.com/AgentEra/Agently)  
Go 目标仓库：[AgentEra/Agently-Go](https://github.com/AgentEra/Agently-Go)

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8.svg)](go.mod)
[![GitHub Stars](https://img.shields.io/github/stars/AgentEra/Agently-Go.svg?style=social)](https://github.com/AgentEra/Agently-Go/stargazers)

## 快速链接

- 仓库结构与 Python 映射：[`docs/project-structure.md`](docs/project-structure.md)
- 示例目录：[`examples/`](examples)
- 回归测试目录：[`tests/`](tests)
- 发布前检查脚本：[`scripts/release-ready.sh`](scripts/release-ready.sh)

## 为什么是 Agently-Go

| 常见生产问题 | Agently-Go 方案 |
|---|---|
| 输出结构漂移导致解析失败 | `Output(...)` + `EnsureKeys` 契约式约束 |
| Prompt 与请求行为难以维护 | 明确的 `Prompt -> Request -> Response` 管线 |
| 流式结果难以结构化消费 | 支持 `delta`、`specific`、`instant` 三类流 |
| 流程编排逻辑容易失控 | 信号驱动 `TriggerFlow` 路由与算子 |
| 多轮会话行为不稳定 | 内置 Session 扩展与上下文控制 |

## 核心能力

- 核心运行时：`agently/core`
- 信号驱动编排：`agently/triggerflow`
- 默认插件链路：
  - `PromptGenerator`
  - `ModelRequester (OpenAICompatible)`
  - `ResponseParser`
  - `ToolManager`
- 默认 Agent 扩展：
  - `Session`
  - `Tool`
  - `ConfigurePrompt`
  - `KeyWaiter`
  - `AutoFunc`

## 快速开始

### 安装

```bash
go get github.com/AgentEra/Agently-Go@v1.0.0
```

### 最小示例

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

## 更多示例

### 结构化输出 + ensure keys

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

### 结构化流式（instant）

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

### TriggerFlow（信号驱动）

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

## 示例矩阵

| 分类 | 示例 | 运行方式 |
|---|---|---|
| 基础 API | `examples/basic/core_api_basics` | `go run ./examples/basic/core_api_basics` |
| ensure_keys | `examples/basic/ensure_keys_in_output` | `go run ./examples/basic/ensure_keys_in_output` |
| 流式模式 | `examples/basic/response_and_streaming` | `go run ./examples/basic/response_and_streaming` |
| 会话模式 | `examples/basic/session_modes` | `go run ./examples/basic/session_modes` |
| ConfigurePrompt | `examples/configure_prompt/loaders_and_roundtrip` | `go run ./examples/configure_prompt/loaders_and_roundtrip` |
| Prompt 生成 | `examples/prompt_generation/text_messages_schema` | `go run ./examples/prompt_generation/text_messages_schema` |
| 工具使用 | `examples/tools_using/tool_agent` | `go run ./examples/tools_using/tool_agent` |
| TriggerFlow 基础 | `examples/trigger_flow/signal_basics` | `go run ./examples/trigger_flow/signal_basics` |
| Step-by-step core | `examples/step_by_step/core_01_to_08` | `go run ./examples/step_by_step/core_01_to_08` |
| 应用案例 | `examples/applied_cases/id_code_analyst` | `go run ./examples/applied_cases/id_code_analyst` |

## 验证

默认入口包含在线回归测试：

```bash
go test ./...
```

常用脚本：

```bash
./scripts/test-offline.sh
./scripts/test-online.sh
./scripts/test-examples.sh
./scripts/verify-full-replication.sh
./scripts/release-ready.sh
```

在线默认环境：

- `OLLAMA_BASE_URL=http://127.0.0.1:11434/v1`
- `OLLAMA_MODEL=qwen2.5:7b`

## 发布范围（v1.0.0）

已包含：

- `core`
- `triggerflow`
- 默认插件与扩展
- 语义回归测试与对齐夹具

已跟踪但不纳入 v1 阻塞范围：

- `Storage/AsyncStorage`
- `PythonSandbox`
- integrations（`fastapi/chromadb/mcp/vlm_support`）
- built-in tools（`search/browse/cmd`）

## 仓库结构

- `agently/`：框架源码（`core`、`triggerflow`、`builtins`、`utils`、`types`）
- `examples/`：可运行场景示例
- `tests/`：离线/在线语义回归
- `docs/`：对外项目文档
- `scripts/`：自动化与验证脚本

## 许可证

Apache-2.0，见 [LICENSE](LICENSE)。

## 特别鸣谢

特别感谢 **GPT-5.3-Codex**。  
本项目迁移涉及大量深度对齐、重构与验证工作。  
GPT-5.3-Codex 让不可能成为可能。
