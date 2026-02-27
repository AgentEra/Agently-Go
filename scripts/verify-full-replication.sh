#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "[1/7] capability matrix fixture validation"
go test ./tests/test_cores -run TestCapabilityMatrixFixtures -count=1

echo "[2/7] offline semantic suite"
./scripts/test-offline.sh

echo "[3/7] online semantic suite"
./scripts/test-online.sh

echo "[4/7] examples smoke suite"
./scripts/test-examples.sh

echo "[5/7] full default suite"
go test ./...

echo "[6/7] stability and race"
go test ./... -count=3
go test ./... -race

echo "[7/7] report path"
echo "Update report: $ROOT_DIR/docs/spec/full-replication-report-2026-02-26.md"
