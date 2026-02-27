# Agently-Go Project Structure (GitHub Release Layout)

This document defines the publish-ready repository structure for:
- `github.com/AgentEra/Agently-Go`

It is aligned with the original Python repository:
- `github.com/AgentEra/Agently`

## 1. Top-level Layout

```text
Agently-Go/
  agently/
    builtins/
    core/
    triggerflow/
    types/
    utils/
    testkit/
  examples/
    basic/
    configure_prompt/
    prompt_generation/
    model_configures/
    step_by_step/
    trigger_flow/
    tools_using/
    applied_cases/
    chromadb/      # v1 excluded (README)
    fastapi/       # v1 excluded (README)
    mcp/           # v1 excluded (README)
    builtin_tools/ # v1 excluded (README)
    vlm_support/   # v1 excluded (README)
  tests/
    fixtures/
    test_cores/
    test_plugins/
    test_extensions/
    test_trigger_flow/
    test_utils/
    test_online/
    testkit/
  scripts/
  docs/
  README.md
  README_CN.md
  LICENSE
  CLA.md
  TRADEMARK.md
  go.mod
  go.sum
```

## 2. Python -> Go Mapping

- Python `agently/core/*` -> Go `agently/core/*`
- Python `agently/core/TriggerFlow/*` -> Go `agently/triggerflow/*`
- Python `agently/builtins/*` -> Go `agently/builtins/*`
- Python `agently/utils/*` -> Go `agently/utils/*`
- Python `tests/*` -> Go `tests/*` (grouped by capability)
- Python `examples/*` -> Go `examples/*` (scenario-equivalent, Go idiomatic code)

## 3. Package Naming Rules

- Import path root: `github.com/AgentEra/Agently-Go`
- Keep stable package segmentation: `core`, `triggerflow`, `utils`, `types`, `builtins`
- Example local variable naming convention:
  - prefer `agentlyApp := agently.NewAgently()`

## 4. Release Notes

For GitHub release readiness:
- root-level readmes and legal files are required
- scripts remain executable entrypoints for validation
- tests and examples should pass from repository root

Recommended pre-release checks:

```bash
go test ./...
go test ./... -race
./scripts/test-examples.sh
./scripts/verify-full-replication.sh
```
