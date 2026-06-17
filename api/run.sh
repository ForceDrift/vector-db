#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

# Start Ollama if not running
if ! pgrep -q ollama; then
    echo "Starting Ollama..."
    ollama serve &
    sleep 2
fi

# Wait for Ollama to be ready
echo "Waiting for Ollama..."
for i in $(seq 1 30); do
    if curl -s http://127.0.0.1:11434/api/tags > /dev/null 2>&1; then
        echo "Ollama ready"
        break
    fi
    sleep 1
done

# Build and start API server
echo "Starting Vector DB API on :8080..."
DYLD_LIBRARY_PATH=../engine go run .
