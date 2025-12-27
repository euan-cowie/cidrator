#!/bin/bash

# Find all Go files in cmd/mtu and its subdirectories
# Output format:
# --- [filename] ---
# [content]

find . -type f -name "*.go" | sort | while read -r file; do
    echo "========================================="
    echo "File: $file"
    echo "========================================="
    cat "$file"
    echo ""
    echo ""
done
