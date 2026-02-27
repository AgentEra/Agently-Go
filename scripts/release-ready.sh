#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "[1/5] checking module path"
expected="module github.com/AgentEra/Agently-Go"
actual="$(head -n 1 go.mod)"
if [[ "$actual" != "$expected" ]]; then
  echo "go.mod module mismatch"
  echo "expected: $expected"
  echo "actual:   $actual"
  exit 1
fi

echo "[2/5] checking required root files"
for f in README.md README_CN.md LICENSE CLA.md TRADEMARK.md .gitignore; do
  [[ -f "$f" ]] || { echo "missing $f"; exit 1; }
done

echo "[3/5] checking old import path leftovers"
if rg -n "github\\.com/moxin/agently-go" -g '*.go' -g 'go.mod' -g '*.md' -g '*.sh' >/dev/null; then
  echo "found old import path references"
  exit 1
fi

echo "[4/5] running tests"
go test ./...

echo "[5/5] release readiness checks passed"
