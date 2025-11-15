#!/bin/bash

# Backtide Version Script
# Simple script to show current version for development builds

set -e

# Get current version from git or use default
if command -v git >/dev/null 2>&1; then
    CURRENT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")
    COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    echo "Version: $CURRENT_TAG"
    echo "Commit: $COMMIT_HASH"
else
    echo "Version: dev (git not available)"
fi

# If backtide binary exists, show its version
if [ -f "./backtide" ]; then
    echo "Binary version: $(./backtide version 2>/dev/null | cut -d' ' -f3 || echo "unknown")"
fi
