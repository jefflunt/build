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

# Verify 'help' works
if ./build help > /dev/null; then
    echo "Successfully verified 'help' works."
else
    echo "Failed: 'help' subcommand failed."
    exit 1
fi
