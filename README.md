# Agently-Go

Go implementation of **Agently** core capabilities, aligned with the original Python project:
- Original project: https://github.com/AgentEra/Agently
- Target repository: https://github.com/AgentEra/Agently-Go

## Status

This repository focuses on semantic parity for:
- `core`
- `TriggerFlow` (signal-driven)
- default plugins and agent extensions
- regression tests + parity fixtures + runnable examples

## Install

```bash
go get github.com/AgentEra/Agently-Go
```

## Quick Start

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

## Repository Layout

- `agently/`: framework source (`core`, `triggerflow`, `builtins`, `utils`, `types`)
- `examples/`: runnable examples grouped by scenario
- `tests/`: semantic regression and online/offline validation
- `docs/`: parity specs, reports, and structure docs
- `scripts/`: test and verification scripts

Detailed structure and mapping to the Python repository:
- [`docs/project-structure.md`](docs/project-structure.md)

## Validation

```bash
go test ./...
./scripts/test-examples.sh
./scripts/verify-full-replication.sh
```

## License

Apache-2.0. See [LICENSE](LICENSE).

## Acknowledgements

Special thanks to **GPT-5.3-Codex**.  
This migration required a massive amount of deep parity work, refactoring, and validation.  
GPT-5.3-Codex made what felt impossible become possible.
