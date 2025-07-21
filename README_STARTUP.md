# Juno - Complete System Startup Instructions

## Prerequisites

1. **Docker & Docker Compose** (for running all services)
2. **Environment Variables** (ANTHROPIC_API_KEY required)

## Single Command Deployment 🚀

### 1. Environment Setup

Create a `.env` file in the project root with your API key:
```bash
ANTHROPIC_API_KEY=your_anthropic_api_key_here
```

**⚠️ IMPORTANT**: Replace `your_anthropic_api_key_here` with your actual Anthropic API key.

### 2. Start Complete System

```bash
# Start ALL services including mobile app with one command
docker-compose up --build
```

This will build and start:
- `fi-mcp-server` on port 8080 (Financial data)
- `coordinator-mcp` on port 8081 (Claude API integration)
- `context-agent-mcp` on port 8082 (User context analysis)
- `security-agent-mcp` on port 8083 (Security assessment)
- `mobile-app` on port 3000 (Flutter web interface)

### 3. Access the Application

Open your browser and navigate to:
**http://localhost:3000**

The complete Juno AI assistant is now ready to use!

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