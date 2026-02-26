#!/bin/sh
set -e

echo "Installing TSR..."

if command -v go >/dev/null 2>&1; then
  go install github.com/sampbb1996-dot/tsr/cmd/tsr@latest
  echo ""
  echo "TSR installed. Run: tsr run <yourfile.tsr>"
else
  echo "Go is not installed. Install it from https://go.dev/dl/ then run:"
  echo "  go install github.com/sampbb1996-dot/tsr/cmd/tsr@latest"
  exit 1
fi
