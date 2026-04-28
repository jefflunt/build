#!/usr/bin/env bash
set -e

# Setup
go build -o bin/build cmd/build/main.go
./bin/build teardown || true
./bin/build init

# Create test plan
echo "# Test Plan
This is a test plan." > test_plan.md

# Run enqueue
./bin/build enqueue test_plan.md

# Verify
SESSION_NAME="test_plan"
OUTPUT_DIR="/tmp/build/breakdowns/$SESSION_NAME"

if [ -d "$OUTPUT_DIR" ]; then
    echo "Test passed: Output directory $OUTPUT_DIR exists."
else
    echo "Test failed: Output directory $OUTPUT_DIR does not exist."
    exit 1
fi

# Clean up
./bin/build teardown
rm test_plan.md
