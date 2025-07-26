#!/bin/bash

echo "Stopping Juno services on localhost ports..."

# Kill processes on specific localhost ports
echo "Killing processes on port 8090 (Fi MCP Server)..."
lsof -ti:8090 | xargs kill -9 2>/dev/null && echo "Port 8090 freed" || echo "Port 8090 already free"

echo "Killing processes on port 8091 (Coordinator)..."
lsof -ti:8091 | xargs kill -9 2>/dev/null && echo "Port 8091 freed" || echo "Port 8091 already free"

echo "Killing processes on port 8092 (Context Agent)..."
lsof -ti:8092 | xargs kill -9 2>/dev/null && echo "Port 8092 freed" || echo "Port 8092 already free"

echo "Killing processes on port 8093 (Security Agent)..."
lsof -ti:8093 | xargs kill -9 2>/dev/null && echo "Port 8093 freed" || echo "Port 8093 already free"

echo "Killing processes on port 3000 (Mobile App)..."
lsof -ti:3000 | xargs kill -9 2>/dev/null && echo "Port 3000 freed" || echo "Port 3000 already free"

# Stop processes from PID file if it exists
if [ -f .juno_pids ]; then
    echo "Cleaning up PID file..."
    for pid in $(cat .juno_pids); do
        if kill -0 $pid 2>/dev/null; then
            kill -9 $pid 2>/dev/null || true
        fi
    done
    rm .juno_pids
fi

echo "All Juno services stopped!"
echo "Localhost ports 8090, 8091, 8092, 8093, and 3000 are now free."