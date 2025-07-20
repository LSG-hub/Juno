# Juno - Startup Instructions

## Prerequisites

1. **Flutter** (for mobile app development)
2. **Go** (for backend MCP servers)
3. **Docker & Docker Compose** (for running all services)

## Quick Start

### 1. Environment Setup

A `.env` file has been created in the project root with the necessary environment variables. The file contains:
```bash
ANTHROPIC_API_KEY=your_anthropic_api_key_here
# ... other environment variables
```

**⚠️ IMPORTANT**: The `.env` file is gitignored and should never be committed to version control.

### 2. Start Backend Services

```bash
# Start all MCP services (automatically loads .env file)
docker-compose up --build
```

This will start:
- `fi-mcp-server` on port 8080
- `coordinator-mcp` on port 8081 (with Claude API integration)
- `context-agent-mcp` on port 8082
- `security-agent-mcp` on port 8083

### 2. Run Mobile App

```bash
# Navigate to mobile app directory
cd mobile_app

# Get dependencies
flutter pub get

# Run the app (requires Android/iOS emulator or device)
flutter run
```

## Service Endpoints

- **Coordinator WebSocket**: `ws://localhost:8081/ws` (for mobile app)
- **Coordinator MCP**: `http://localhost:8081/mcp/`
- **Fi MCP Server**: `http://localhost:8080/mcp/`
- **Context Agent**: `http://localhost:8082/mcp/`
- **Security Agent**: `http://localhost:8083/mcp/`

## Health Checks

All services expose health endpoints:
- `http://localhost:8080/health`
- `http://localhost:8081/health`
- `http://localhost:8082/health`
- `http://localhost:8083/health`

## Mobile App Features

- Modern Material 3 design
- Real-time WebSocket chat with Juno AI
- Claude Sonnet 4 integration for intelligent responses
- Connection status indicators
- Typing indicators and message status
- Auto-reconnection on network issues

## Demo Usage

1. Start all backend services with `docker-compose up`
2. Run the Flutter app
3. Chat with Juno about financial queries like:
   - "How's my spending this month?"
   - "What's my emergency fund status?"
   - "Show me my financial security assessment"

The system will provide intelligent responses powered by Claude and the multi-agent financial analysis system.