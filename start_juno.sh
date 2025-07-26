#!/bin/bash
set -e


export $(grep -v '^#' .env | grep -v '^$' | xargs)

echo "Starting Juno Services..."


echo "Starting Fi MCP Server..."
cd fi-mcp-server
export FI_MCP_PORT=8090
go build -o fi-mcp-server main.go
./fi-mcp-server &
FI_PID=$!
cd ..

sleep 3


echo "Starting Context Agent..."
cd backend/context_agent_mcp
export PORT=8092
export GEMINI_API_KEY=$GEMINI_API_KEY
go build -o context-agent-mcp main.go
./context-agent-mcp &
CONTEXT_PID=$!
cd ../..


echo "Starting Security Agent..."
cd backend/security_agent_mcp
export PORT=8093
go build -o security-agent-mcp main.go
./security-agent-mcp &
SECURITY_PID=$!
cd ../..


echo "Starting Coordinator..."
cd backend/coordinator_mcp
export PORT=8091
export GEMINI_API_KEY=$GEMINI_API_KEY
go build -o coordinator-mcp main.go
./coordinator-mcp &
COORDINATOR_PID=$!
cd ../..


echo "Starting Mobile App..."
cd mobile_app
flutter build web --release
cd build/web
http-server -p 3000 -c-1 &
MOBILE_PID=$!
cd ../../..

echo "All services started!"
echo "Fi MCP Server: http://localhost:8090"
echo "Context Agent: http://localhost:8092"
echo "Security Agent: http://localhost:8093"
echo "Coordinator: http://localhost:8091"
echo "Mobile App: http://localhost:3000"


echo "$FI_PID $CONTEXT_PID $SECURITY_PID $COORDINATOR_PID $MOBILE_PID" > .juno_pids

echo "To stop all services, run: ./stop_juno.sh"