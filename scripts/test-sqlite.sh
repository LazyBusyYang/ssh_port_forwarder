#!/bin/bash
set -e

echo "Running SQLite integration tests..."

TEMP_DIR=$(mktemp -d)
export TEST_DB_PATH="$TEMP_DIR/spf_test.db"

go test ./internal/... -v -tags=integration -run=SQLite

rm -rf "$TEMP_DIR"
echo "SQLite tests passed!"
