#!/bin/bash

if [ -f .juno_pids ]; then
    echo "Stopping Juno services..."
    for pid in $(cat .juno_pids); do
        if kill -0 $pid 2>/dev/null; then
            kill $pid
            echo "Stopped process $pid"
        fi
    done
    rm .juno_pids
    echo "All services stopped."
else
    echo "No running services found."
fi