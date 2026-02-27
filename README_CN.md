# Agently-Go

**Agently 的 Go 语言实现**，目标与原始 Python 项目能力语义对齐：
- 原始项目：https://github.com/AgentEra/Agently
- 目标仓库：https://github.com/AgentEra/Agently-Go

## 当前范围

本仓库当前重点覆盖：
- `core`
- `TriggerFlow`（信号驱动）
- 默认插件与 Agent 扩展
- 回归测试、对齐夹具与可运行示例

## 安装

```bash
go get github.com/AgentEra/Agently-Go
```

## 快速开始

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
        Input("用一句话解释递归").
        GetText()
    if err != nil {
        panic(err)
    }

    fmt.Println(text)
}
```

## 仓库结构

- `agently/`：框架源码（`core`、`triggerflow`、`builtins`、`utils`、`types`）
- `examples/`：按主题组织的示例
- `tests/`：离线/在线语义回归
- `docs/`：对齐规格与报告文档
- `scripts/`：测试与验证脚本

与 Python 原项目的目录映射见：
- [`docs/project-structure.md`](docs/project-structure.md)

## 验证命令

```bash
go test ./...
./scripts/test-examples.sh
./scripts/verify-full-replication.sh
```

## 许可证

Apache-2.0，见 [LICENSE](LICENSE)。

## 特别鸣谢

特别感谢 **GPT-5.3-Codex**。  
本项目迁移涉及大量深度对齐、重构与验证工作。  
GPT-5.3-Codex 让不可能成为可能。
