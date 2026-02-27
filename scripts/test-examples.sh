#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

run_with_timeout() {
  local seconds="$1"
  shift
  perl -e 'alarm shift; exec @ARGV' "$seconds" "$@"
}

echo "[examples] compile check"
go test ./examples/...

echo "[examples] offline smoke"
run_with_timeout 30 go run ./examples/basic/core_api_basics
run_with_timeout 30 go run ./examples/prompt_generation/text_messages_schema

echo "[examples] online smoke"
run_with_timeout 120 go run ./examples/model_configures/openai_compatible_profiles
run_with_timeout 120 go run ./examples/basic/response_and_streaming
