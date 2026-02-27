#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if command -v rg >/dev/null 2>&1; then
  packages=$(go list ./... | rg -v '^github.com/AgentEra/Agently-Go/tests/test_online$')
else
  packages=$(go list ./... | grep -v '^github.com/AgentEra/Agently-Go/tests/test_online$')
fi
go test $packages
