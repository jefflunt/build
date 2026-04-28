#!/bin/bash
set -e

# Build the CLI
go build -o build ./cmd/build/main.go

# Verify 'design' fails
if ./build design 2>&1 | grep -q "Unknown subcommand: design"; then
    echo "Successfully verified 'design' is removed."
else
    echo "Failed: 'design' subcommand still exists or did not return expected error."
    exit 1
fi

# Verify 'help' works and contains 'enqueue'
if ./build help | grep -q "enqueue"; then
    echo "Successfully verified 'enqueue' is in help."
else
    echo "Failed: 'enqueue' not found in help."
    exit 1
fi

# Verify 'enqueue' without arguments fails
if ./build enqueue 2>&1 | grep -q "Usage: build enqueue <plan file>"; then
    echo "Successfully verified 'enqueue' without args fails."
else
    echo "Failed: 'enqueue' without args did not fail as expected."
    exit 1
fi

# Verify 'enqueue' with argument succeeds
# We create a dummy plan file first
echo "# Test Plan" > myplan.json
if ./build enqueue myplan.json; then
    echo "Successfully verified 'enqueue' with arg works."
    
    # Verify the output directory was created
    # The session name derived from 'myplan.json' is 'myplan'
    OUTPUT_DIR="/tmp/build/breakdowns/myplan"
    if [ -d "$OUTPUT_DIR" ]; then
        echo "Successfully verified output directory $OUTPUT_DIR was created."
        rm -rf /tmp/build/breakdowns/myplan
    else
        echo "Failed: Output directory $OUTPUT_DIR was not created."
        exit 1
    fi
else
    echo "Failed: 'enqueue' with arg did not work as expected."
    exit 1
fi
rm myplan.json
