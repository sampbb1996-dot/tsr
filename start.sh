#!/bin/sh
set -e
echo "Building server..."
go build -o server ./cmd/server
echo "Starting server..."
exec ./server
