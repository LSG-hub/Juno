# Juno Individual Setup Guide - Running Without Docker

This guide provides complete step-by-step instructions for running the entire Juno Financial Assistant system locally without Docker, using virtual environments and manual service management.

**⚠️ Important Port Note**: This guide uses the correct external port numbers that match the Docker configuration. Fi MCP Server uses port 8090 (not 8080) and Coordinator uses port 8091 (not 8081) to match the external Docker port mapping.

## System Overview

The Juno system consists of:
- **Fi MCP Server** (Port 8090) - Financial data provider
- **Context Agent MCP Server** (Port 8092) - RAG intelligence hub
- **Security Agent MCP Server** (Port 8093) - Risk assessment
- **Coordinator MCP Server** (Port 8091) - Orchestration hub with Gemini API
- **Mobile App** (Port 3000) - Flutter web interface

## Prerequisites

### 1. Install System Dependencies

#### macOS:
```bash
# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Go 1.23+
brew install go

# Install Flutter (stable channel)
brew install flutter

# Install Node.js (for serving Flutter web)
brew install node

# Install Python (for virtual environment management)
brew install python@3.11
```

#### Linux (Ubuntu/Debian):
```bash
# Update package list
sudo apt update

# Install Go 1.23+
sudo rm -rf /usr/local/go
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Flutter
sudo snap install flutter --classic

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# Install Python 3.11+
sudo apt install python3.11 python3.11-venv python3-pip
```

#### Windows:
```powershell
# Install Go from https://golang.org/dl/
# Install Flutter from https://docs.flutter.dev/get-started/install/windows
# Install Node.js from https://nodejs.org/
# Install Python 3.11+ from https://www.python.org/downloads/windows/
```

### 2. Verify Installations
```bash
go version          # Should show Go 1.23+
flutter --version   # Should show Flutter stable
node --version      # Should show Node.js 18+
python3 --version   # Should show Python 3.11+
```

## Setup Instructions

### Step 1: Navigate to Project Directory
```bash
cd /Users/sreenivasg/Desktop/Projects/Juno
# OR wherever your Juno project is located
```

### Step 2: Create Python Virtual Environment (Optional - for future Python services)
```bash
# Create virtual environment
python3 -m venv juno_venv

# Activate virtual environment
# macOS/Linux:
source juno_venv/bin/activate
# Windows:
# juno_venv\Scripts\activate

# Verify activation (should show juno_venv in prompt)
which python  # Should point to juno_venv/bin/python
```

### Step 3: Setup Environment Variables

Create a `.env` file in the project root:
```bash
# Copy from existing .env or create new one
cp .env.example .env  # If available
# OR create manually:
touch .env
```

Add the following to `.env`:
```env
# Gemini API Configuration
GEMINI_API_KEY=AIzaSyBvIzIMpPcqUduNF6rSUL2o-ClYWO4GtTA

# Service Ports (External/Localhost ports)
FI_MCP_PORT=8090
CONTEXT_AGENT_PORT=8092
SECURITY_AGENT_PORT=8093
COORDINATOR_MCP_PORT=8091
MOBILE_APP_PORT=3000

# Service URLs (for local development)
FI_MCP_URL=http://localhost:8090
CONTEXT_AGENT_URL=http://localhost:8092
SECURITY_AGENT_URL=http://localhost:8093

# Fi MCP External Port (for browser access)
FI_MCP_EXTERNAL_PORT=8090
```

Export environment variables:
```bash
# macOS/Linux:
export $(cat .env | xargs)

# Windows (PowerShell):
Get-Content .env | ForEach-Object { 
    $key, $value = $_ -split '=', 2
    [Environment]::SetEnvironmentVariable($key, $value, "Process")
}
```

## Service Setup and Startup

### Step 4: Setup Go Services

#### A. Fi MCP Server (Port 8090)
```bash
# Navigate to Fi MCP server directory
cd fi-mcp-server

# Set the port for Fi MCP (use external port)
export PORT=8090

# Download Go dependencies
go mod tidy
go mod download

# Build the server
go build -o fi-mcp-server main.go

# Start the server (keep this terminal open)
./fi-mcp-server
```

**Expected Output:**
```
Starting Fi MCP Server on port 8090
MCP endpoint: http://localhost:8090/mcp/
Health endpoint: http://localhost:8090/health
```

#### B. Context Agent MCP Server (Port 8092)
**Open a new terminal window/tab:**
```bash
cd /Users/sreenivasg/Desktop/Projects/Juno/backend/context_agent_mcp

# Set environment variables again in new terminal
export $(cat ../../.env | xargs)

# Download Go dependencies
go mod tidy
go mod download

# Build the server
go build -o context-agent-mcp main.go

# Start the server (keep this terminal open)
./context-agent-mcp
```

**Expected Output:**
```
Starting Context Agent MCP Server on port 8092
MCP endpoint: http://localhost:8092/mcp/
Health endpoint: http://localhost:8092/health
```

#### C. Security Agent MCP Server (Port 8093)
**Open a new terminal window/tab:**
```bash
cd /Users/sreenivasg/Desktop/Projects/Juno/backend/security_agent_mcp

# Set environment variables again in new terminal
export $(cat ../../.env | xargs)

# Download Go dependencies
go mod tidy
go mod download

# Build the server
go build -o security-agent-mcp main.go

# Start the server (keep this terminal open)
./security-agent-mcp
```

**Expected Output:**
```
Starting Security Agent MCP Server on port 8093
MCP endpoint: http://localhost:8093/mcp/
Health endpoint: http://localhost:8093/health
```

#### D. Coordinator MCP Server (Port 8091)
**Open a new terminal window/tab:**
```bash
cd /Users/sreenivasg/Desktop/Projects/Juno/backend/coordinator_mcp

# Set environment variables again in new terminal
export $(cat ../../.env | xargs)
export PORT=8091
export GEMINI_API_KEY=AIzaSyBvIzIMpPcqUduNF6rSUL2o-ClYWO4GtTA

# Download Go dependencies
go mod tidy
go mod download

# Build the server
go build -o coordinator-mcp main.go

# Start the server (keep this terminal open)
./coordinator-mcp
```

**Expected Output:**
```
Starting Coordinator MCP Server on port 8091
MCP endpoint: http://localhost:8091/mcp/
Health endpoint: http://localhost:8091/health
Connecting to Fi MCP Server at: http://localhost:8090
Connecting to Context Agent at: http://localhost:8092
Connecting to Security Agent at: http://localhost:8093
```

### Step 5: Setup Flutter Mobile App (Port 3000)

#### A. Install Flutter Dependencies
**Open a new terminal window/tab:**
```bash
cd /Users/sreenivasg/Desktop/Projects/Juno/mobile_app

# Get Flutter dependencies
flutter pub get

# Verify no issues
flutter analyze
```

#### B. Configure Firebase (Required for Authentication)
```bash
# Make sure Firebase configuration files exist:
# - web/firebase-config.js
# - lib/firebase_options.dart

# If missing, you'll need to set up Firebase project:
# 1. Go to https://console.firebase.google.com/
# 2. Create/select project: juno-financial-assistant
# 3. Add web app and download config
# 4. Enable Authentication with Email, Google, Anonymous
# 5. Create Firestore database in test mode
```

#### C. Build and Serve Flutter Web App
```bash
# Build for web with correct coordinator port
flutter build web --release --dart-define=COORDINATOR_PORT=8091

# Install a simple HTTP server (if not already installed)
npm install -g http-server

# Serve the built web app on port 3000
cd build/web
http-server -p 3000 -c-1

# Alternative: Use Flutter's built-in server (with environment variable)
# flutter run -d web-server --web-hostname 0.0.0.0 --web-port 3000 --dart-define=COORDINATOR_PORT=8091
```

**Expected Output:**
```
Starting up http-server, serving ./
http-server version: 14.1.1
Available on:
  http://127.0.0.1:3000
  http://localhost:3000
```

## Health Check and Testing

### Step 6: Verify All Services

Open a new terminal and test each service:

```bash
# Test Fi MCP Server
curl http://localhost:8090/health
# Expected: {"status":"healthy","service":"fi-mcp-server","version":"..."}

# Test Context Agent
curl http://localhost:8092/health
# Expected: {"status":"healthy","service":"context-agent-mcp","version":"0.1.0"}

# Test Security Agent
curl http://localhost:8093/health
# Expected: {"status":"healthy","service":"security-agent-mcp","version":"0.1.0"}

# Test Coordinator
curl http://localhost:8091/health
# Expected: {"status":"healthy","service":"coordinator-mcp","version":"0.1.0"}

# Test Mobile App
curl http://localhost:3000
# Expected: HTML content of the Flutter web app
```

### Step 7: Test Complete System

1. **Open Mobile App:**
   ```
   http://localhost:3000
   ```

2. **Authenticate:**
   - Click "Quick Demo Access" for anonymous login
   - OR create account with email/Google

3. **Select Fi User:**
   - Use dropdown to select test user (e.g., 1111111111)

4. **Test Financial Query:**
   - Type: "What's my net worth?"
   - Should prompt for Fi login
   - Login with phone: 1111111111, OTP: 123456

5. **Test RAG Context:**
   - Ask follow-up questions
   - System should remember previous context

## Service Management

### Starting All Services (Quick Script)

Create a startup script `start_juno.sh`:
```bash
#!/bin/bash
set -e

# Load environment variables (filter out comments)
export $(grep -v '^#' .env | grep -v '^$' | xargs)

echo "Starting Juno Services..."

# Start Fi MCP Server
echo "Starting Fi MCP Server..."
cd fi-mcp-server
export PORT=8090
go build -o fi-mcp-server main.go
./fi-mcp-server &
FI_PID=$!
cd ..

# Wait for Fi server to start
sleep 3

# Start Context Agent
echo "Starting Context Agent..."
cd backend/context_agent_mcp
export PORT=8092
go build -o context-agent-mcp main.go
./context-agent-mcp &
CONTEXT_PID=$!
cd ../..

# Start Security Agent
echo "Starting Security Agent..."
cd backend/security_agent_mcp
export PORT=8093
go build -o security-agent-mcp main.go
./security-agent-mcp &
SECURITY_PID=$!
cd ../..

# Start Coordinator
echo "Starting Coordinator..."
cd backend/coordinator_mcp
export PORT=8091
export GEMINI_API_KEY=AIzaSyBvIzIMpPcqUduNF6rSUL2o-ClYWO4GtTA
go build -o coordinator-mcp main.go
./coordinator-mcp &
COORDINATOR_PID=$!
cd ../..

# Start Mobile App
echo "Starting Mobile App..."
cd mobile_app
flutter build web --release --dart-define=COORDINATOR_PORT=8091
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

# Save PIDs for cleanup
echo "$FI_PID $CONTEXT_PID $SECURITY_PID $COORDINATOR_PID $MOBILE_PID" > .juno_pids

echo "To stop all services, run: ./stop_juno.sh"
```

Create a stop script `stop_juno.sh`:
```bash
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
```

Make scripts executable:
```bash
chmod +x start_juno.sh stop_juno.sh
```

### Usage:
```bash
# Start all services
./start_juno.sh

# Stop all services
./stop_juno.sh
```

## Troubleshooting

### Common Issues:

1. **Port Already in Use:**
   ```bash
   # Find process using port (example for Fi MCP)
   lsof -i :8090
   # Kill process
   kill -9 <PID>
   ```

2. **Go Module Issues:**
   ```bash
   # Clean Go module cache
   go clean -modcache
   # Re-download dependencies
   go mod download
   ```

3. **Flutter Build Issues:**
   ```bash
   # Clean Flutter cache
   flutter clean
   flutter pub get
   ```

4. **Environment Variables Not Loaded:**
   ```bash
   # Manually export in each terminal
   export GEMINI_API_KEY=AIzaSyBvIzIMpPcqUduNF6rSUL2o-ClYWO4GtTA
   export FI_MCP_URL=http://localhost:8090
   export PORT=8090  # For Fi MCP
   # ... etc
   ```

5. **Juno Stuck in Demo Mode (Always Same Response):**
   - **Symptom**: Juno responds "I'm currently running in demo mode" to all queries
   - **Cause**: GEMINI_API_KEY not loaded properly
   - **Solution**: 
     ```bash
     # Check if API key is set
     echo $GEMINI_API_KEY
     
     # If empty, export manually
     export GEMINI_API_KEY=AIzaSyBvIzIMpPcqUduNF6rSUL2o-ClYWO4GtTA
     
     # Restart coordinator
     pkill coordinator-mcp
     cd backend/coordinator_mcp && ./coordinator-mcp
     ```

5. **Firebase Configuration Missing:**
   - Ensure `web/firebase-config.js` exists
   - Ensure `lib/firebase_options.dart` exists
   - Check Firebase project settings

### Service Dependencies:

**Startup Order (Important):**
1. Fi MCP Server (8090) - Must start first
2. Context Agent (8092) - Depends on Fi MCP
3. Security Agent (8093) - Depends on Fi MCP
4. Coordinator (8091) - Depends on all above
5. Mobile App (3000) - Depends on Coordinator

### Log Locations:

All services log to stdout/stderr. To save logs:
```bash
# Example for Fi MCP server
./fi-mcp-server > fi-mcp.log 2>&1 &

# View logs
tail -f fi-mcp.log
```

## Production Considerations

For production deployment:
1. Use process managers (PM2, systemd)
2. Implement proper logging (structured logs)
3. Add monitoring and health checks
4. Configure reverse proxy (nginx)
5. Set up SSL certificates
6. Configure environment-specific variables
7. Implement graceful shutdown handling

## Success Criteria

✅ All health endpoints return 200 OK  
✅ Mobile app loads at http://localhost:3000  
✅ Firebase authentication works  
✅ Fi login flow completes successfully  
✅ RAG context system remembers conversation history  
✅ Multi-user isolation works correctly  

The system is fully operational when you can:
- Login with different authentication methods
- Switch between Fi test users (1010101010 - 9999999999)
- Ask financial questions and get accurate data
- See conversation context maintained across sessions