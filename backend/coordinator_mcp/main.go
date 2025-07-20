package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type CoordinatorServer struct {
	mcpServer       *server.Server
	anthropicAPIKey string
	fiMCPURL        string
	contextAgentURL string
	securityAgentURL string
	upgrader        websocket.Upgrader
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	Messages  []ChatMessage `json:"messages"`
}

type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
}

type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      string      `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func NewCoordinatorServer() *CoordinatorServer {
	return &CoordinatorServer{
		anthropicAPIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		fiMCPURL:        getEnvWithDefault("FI_MCP_URL", "http://fi-mcp-server:8080"),
		contextAgentURL:  getEnvWithDefault("CONTEXT_AGENT_URL", "http://context-agent-mcp:8082"),
		securityAgentURL: getEnvWithDefault("SECURITY_AGENT_URL", "http://security-agent-mcp:8083"),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}
}

func (cs *CoordinatorServer) setupMCPServer() {
	cs.mcpServer = server.NewMCPServer(
		"Coordinator MCP",
		"0.1.0",
		server.WithInstructions("Juno Coordinator MCP Server - Orchestrates multi-agent financial AI system"),
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	// Add coordination tools
	cs.mcpServer.AddTool(
		mcp.NewTool("process_user_query", mcp.WithDescription("Process user financial query through multi-agent system")),
		cs.handleProcessQuery,
	)

	cs.mcpServer.AddTool(
		mcp.NewTool("get_financial_context", mcp.WithDescription("Get user financial context from agents")),
		cs.handleGetContext,
	)
}

func (cs *CoordinatorServer) handleProcessQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract query from request
	params, ok := request.Params.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid parameters"), nil
	}

	query, ok := params["query"].(string)
	if !ok {
		return mcp.NewToolResultError("Missing query parameter"), nil
	}

	// Call Claude API for intelligent response
	response, err := cs.callClaudeAPI(query)
	if err != nil {
		log.Printf("Error calling Claude API: %v", err)
		return mcp.NewToolResultText(fmt.Sprintf("I'm having trouble processing your request: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

func (cs *CoordinatorServer) handleGetContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// For now, return static context
	contextData := map[string]interface{}{
		"user_status": "active",
		"last_sync":   time.Now().Format(time.RFC3339),
		"agents_available": []string{
			"fi-mcp-server",
			"context-agent-mcp",
			"security-agent-mcp",
		},
	}

	jsonData, err := json.Marshal(contextData)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal context data"), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (cs *CoordinatorServer) callClaudeAPI(query string) (string, error) {
	if cs.anthropicAPIKey == "" {
		return "Claude API key not configured", fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	requestBody := AnthropicRequest{
		Model:     "claude-3-5-sonnet-20241022",
		MaxTokens: 1000,
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf("You are Juno, a financial AI assistant. Please help with this query: %s", query),
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cs.anthropicAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(anthropicResp.Content) > 0 && anthropicResp.Content[0].Type == "text" {
		return anthropicResp.Content[0].Text, nil
	}

	return "No response from Claude", nil
}

func (cs *CoordinatorServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := cs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Println("WebSocket connection established")

	for {
		var msg MCPMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		log.Printf("Received message: %+v", msg)

		// Handle different MCP methods
		switch msg.Method {
		case "process_query":
			response := cs.processWebSocketQuery(msg)
			if err := conn.WriteJSON(response); err != nil {
				log.Printf("WebSocket write error: %v", err)
				break
			}
		default:
			// Echo back unknown methods
			response := MCPMessage{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Error:   map[string]string{"message": "Unknown method"},
			}
			if err := conn.WriteJSON(response); err != nil {
				log.Printf("WebSocket write error: %v", err)
				break
			}
		}
	}
}

func (cs *CoordinatorServer) processWebSocketQuery(msg MCPMessage) MCPMessage {
	params, ok := msg.Params.(map[string]interface{})
	if !ok {
		return MCPMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error:   map[string]string{"message": "Invalid parameters"},
		}
	}

	query, ok := params["query"].(string)
	if !ok {
		return MCPMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error:   map[string]string{"message": "Missing query parameter"},
		}
	}

	// Process query with Claude API
	response, err := cs.callClaudeAPI(query)
	if err != nil {
		return MCPMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Result:  map[string]string{"response": fmt.Sprintf("Error: %v", err)},
		}
	}

	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  map[string]string{"response": response},
	}
}

func (cs *CoordinatorServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "coordinator-mcp",
		"version": "0.1.0",
	})
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	coordinator := NewCoordinatorServer()
	coordinator.setupMCPServer()

	// Setup HTTP routes
	httpMux := http.NewServeMux()
	
	// Health check endpoint
	httpMux.HandleFunc("/health", coordinator.healthHandler)
	
	// WebSocket endpoint for mobile app
	httpMux.HandleFunc("/ws", coordinator.handleWebSocket)
	
	// MCP server endpoint
	streamableServer := server.NewStreamableHTTPServer(coordinator.mcpServer,
		server.WithEndpointPath("/mcp/"),
	)
	httpMux.Handle("/mcp/", streamableServer)

	port := getEnvWithDefault("PORT", "8081")
	log.Printf("Starting Coordinator MCP Server on port %s", port)
	log.Printf("WebSocket endpoint: ws://localhost:%s/ws", port)
	log.Printf("MCP endpoint: http://localhost:%s/mcp/", port)
	log.Printf("Health endpoint: http://localhost:%s/health", port)

	if err := http.ListenAndServe(":"+port, httpMux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}