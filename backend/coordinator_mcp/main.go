package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mark3labs/mcp-go/client"
)

type CoordinatorServer struct {
	mcpServer        *server.MCPServer
	anthropicAPIKey  string
	fiMCPURL         string
	contextAgentURL  string
	securityAgentURL string
	upgrader         websocket.Upgrader
	fiMCPClient      *client.Client // Persistent Fi MCP client
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
	cs := &CoordinatorServer{
		anthropicAPIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		fiMCPURL:        getEnvWithDefault("FI_MCP_URL", "http://fi-mcp-server:8080"),
		contextAgentURL:  getEnvWithDefault("CONTEXT_AGENT_URL", "http://context-agent-mcp:8082"),
		securityAgentURL: getEnvWithDefault("SECURITY_AGENT_URL", "http://security-agent-mcp:8083"),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}
	
	// Initialize persistent Fi MCP client
	cs.initializeFiClient()
	
	return cs
}

func (cs *CoordinatorServer) initializeFiClient() {
	var err error
	cs.fiMCPClient, err = client.NewStreamableHttpClient(cs.fiMCPURL + "/mcp/")
	if err != nil {
		log.Printf("Warning: Failed to create persistent Fi MCP client: %v", err)
		return
	}
	
	// Start and initialize the persistent Fi MCP client
	ctx := context.Background()
	if err := cs.fiMCPClient.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start persistent Fi MCP client: %v", err)
		cs.fiMCPClient = nil
		return
	}
	
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "coordinator-mcp",
				Version: "0.1.0",
			},
		},
	}
	_, err = cs.fiMCPClient.Initialize(ctx, initRequest)
	if err != nil {
		log.Printf("Warning: Failed to initialize persistent Fi MCP client: %v", err)
		cs.fiMCPClient.Close()
		cs.fiMCPClient = nil
		return
	}
	
	log.Printf("Successfully initialized persistent Fi MCP client")
}

func (cs *CoordinatorServer) setupMCPServer() {
	cs.mcpServer = server.NewMCPServer(
		"coordinator-mcp",
		"0.1.0",
		server.WithInstructions("Juno Coordinator MCP Server - Orchestrates multi-agent financial AI system"),
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	// Add coordination tools
	cs.mcpServer.AddTool(
		mcp.NewTool("process_user_query",
			mcp.WithDescription("Process user financial query through multi-agent system"),
			mcp.WithString("query",
				mcp.Description("User's financial question or request"),
				mcp.Required(),
			),
		),
		cs.handleProcessQuery,
	)

	cs.mcpServer.AddTool(
		mcp.NewTool("get_financial_context",
			mcp.WithDescription("Get user financial context from agents"),
			mcp.WithString("user_id",
				mcp.Description("User ID for context retrieval"),
			),
		),
		cs.handleGetContext,
	)

	cs.mcpServer.AddTool(
		mcp.NewTool("fetch_financial_data",
			mcp.WithDescription("Fetch financial data from Fi MCP server"),
			mcp.WithString("tool_name",
				mcp.Description("Name of Fi tool to call (e.g., fetch_net_worth, fetch_bank_transactions)"),
				mcp.Required(),
			),
		),
		cs.handleFetchFinancialData,
	)
}

func (cs *CoordinatorServer) handleProcessQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	query, ok := arguments["query"].(string)
	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Missing or invalid query parameter",
				},
			},
			IsError: true,
		}, nil
	}

	// Call Claude API with tools for intelligent response
	response, err := cs.callClaudeAPIWithTools(query)
	if err != nil {
		log.Printf("Error calling Claude API: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("I'm having trouble processing your request: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: response,
			},
		},
	}, nil
}

func (cs *CoordinatorServer) handleGetContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	userID, _ := arguments["user_id"].(string)
	if userID == "" {
		userID = "default_user"
	}

	// For MVP, return static context
	contextData := map[string]interface{}{
		"user_id":      userID,
		"user_status":  "active",
		"last_sync":    time.Now().Format(time.RFC3339),
		"agents_available": []string{
			"fi-mcp-server",
			"context-agent-mcp",
			"security-agent-mcp",
		},
	}

	jsonData, err := json.Marshal(contextData)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Failed to marshal context data",
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

func (cs *CoordinatorServer) handleFetchFinancialData(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	toolName, ok := arguments["tool_name"].(string)
	if !ok || toolName == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Missing or invalid tool_name parameter",
				},
			},
			IsError: true,
		}, nil
	}

	// Call Fi MCP server
	response, err := cs.callFiMCP(toolName)
	if err != nil {
		log.Printf("Error calling Fi MCP: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error fetching financial data: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: response,
			},
		},
	}, nil
}

// Fi MCP tool call using persistent client to maintain session
func (cs *CoordinatorServer) callFiMCPTool(toolName string) (*mcp.CallToolResult, error) {
	// Check if persistent client is available
	if cs.fiMCPClient == nil {
		return nil, fmt.Errorf("persistent Fi MCP client not available")
	}

	// Call the Fi tool using persistent client (maintains session)
	ctx := context.Background()
	toolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: map[string]interface{}{},
		},
	}
	result, err := cs.fiMCPClient.CallTool(ctx, toolRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to call Fi MCP tool %s: %w", toolName, err)
	}

	return result, nil
}

func (cs *CoordinatorServer) callFiMCP(toolName string) (string, error) {
	// Call Fi tool (will return login_required if not authenticated)
	result, err := cs.callFiMCPTool(toolName)
	if err != nil {
		return "", err
	}

	// Extract text content from result
	var responseText strings.Builder
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			responseText.WriteString(textContent.Text)
		}
	}

	resultText := responseText.String()
	if resultText == "" {
		return fmt.Sprintf("Fi MCP Result: %v", result.Content), nil
	}

	// Check if this is a login_required response
	if strings.Contains(resultText, "login_required") {
		log.Printf("Fi requires authentication for tool: %s", toolName)
		// Return the login_required response as-is so mobile app can handle it
		return resultText, nil
	}

	// Regular successful response
	return resultText, nil
}

// Claude API request with tool calls support
func (cs *CoordinatorServer) callClaudeAPIWithTools(query string) (string, error) {
	if cs.anthropicAPIKey == "" {
		return "Hello! I'm Juno, your financial AI assistant. I'm currently running in demo mode. How can I help you with your finances today?", nil
	}

	// Define Fi tools available to Claude
	tools := []map[string]interface{}{
		{
			"name": "fetch_net_worth",
			"description": "Fetch user's comprehensive net worth including assets, liabilities, and total wealth",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
				"required": []string{},
			},
		},
		{
			"name": "fetch_bank_transactions",
			"description": "Fetch user's bank transaction history and account details",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
				"required": []string{},
			},
		},
		{
			"name": "fetch_mf_transactions",
			"description": "Fetch user's mutual fund transactions and investment details",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
				"required": []string{},
			},
		},
		{
			"name": "fetch_credit_report",
			"description": "Fetch user's credit report including credit score, loan details, and account history",
			"input_schema": map[string]interface{}{
				"type": "object", 
				"properties": map[string]interface{}{},
				"required": []string{},
			},
		},
		{
			"name": "fetch_epf_details",
			"description": "Fetch user's Employee Provident Fund (EPF) details and balance",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
				"required": []string{},
			},
		},
	}

	requestBody := map[string]interface{}{
		"model":      "claude-3-5-sonnet-20241022",
		"max_tokens": 1000,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": fmt.Sprintf("You are Juno, a helpful financial AI assistant with access to the user's financial data through Fi Money. Please provide a helpful response to this query: %s", query),
			},
		},
		"tools": tools,
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

	var anthropicResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Handle tool calls if Claude wants to call Fi tools
	if content, ok := anthropicResp["content"].([]interface{}); ok {
		var finalResponse strings.Builder
		
		for _, item := range content {
			if contentItem, ok := item.(map[string]interface{}); ok {
				if contentItem["type"] == "text" {
					if text, exists := contentItem["text"]; exists {
						finalResponse.WriteString(fmt.Sprintf("%v", text))
					}
				} else if contentItem["type"] == "tool_use" {
					// Claude wants to call a tool
					toolName, _ := contentItem["name"].(string)
					toolId, _ := contentItem["id"].(string)
					
					// Call Fi MCP tool
					toolResult, err := cs.callFiMCP(toolName)
					if err != nil {
						log.Printf("Error calling Fi tool %s: %v", toolName, err)
						toolResult = fmt.Sprintf("Error accessing %s data", toolName)
					}
					
					// Check if Fi returned login_required - if so, return it directly without Claude processing
					if strings.Contains(toolResult, "login_required") {
						return toolResult, nil
					}
					
					// Continue conversation with tool result
					followUpResponse, err := cs.callClaudeAPIWithToolResult(query, toolName, toolId, toolResult)
					if err != nil {
						log.Printf("Error in follow-up call: %v", err)
						finalResponse.WriteString(fmt.Sprintf("\nI retrieved your %s data but had trouble processing it.", toolName))
					} else {
						finalResponse.WriteString(followUpResponse)
					}
				}
			}
		}
		
		result := finalResponse.String()
		if result != "" {
			return result, nil
		}
	}

	return "I'm having trouble generating a response right now. Please try again.", nil
}

// Follow-up call to Claude with tool result
func (cs *CoordinatorServer) callClaudeAPIWithToolResult(originalQuery, toolName, toolId, toolResult string) (string, error) {
	requestBody := map[string]interface{}{
		"model":      "claude-3-5-sonnet-20241022", 
		"max_tokens": 1000,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": fmt.Sprintf("You are Juno, a helpful financial AI assistant. The user asked: %s", originalQuery),
			},
			{
				"role": "assistant",
				"content": []map[string]interface{}{
					{
						"type": "tool_use",
						"id":   toolId,
						"name": toolName,
						"input": map[string]interface{}{},
					},
				},
			},
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type":       "tool_result",
						"tool_use_id": toolId,
						"content":    toolResult,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal follow-up request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create follow-up request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cs.anthropicAPIKey) 
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make follow-up request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("follow-up API returned status %d", resp.StatusCode)
	}

	var anthropicResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("failed to decode follow-up response: %w", err)
	}

	// Extract text response
	if content, ok := anthropicResp["content"].([]interface{}); ok {
		for _, item := range content {
			if contentItem, ok := item.(map[string]interface{}); ok {
				if contentItem["type"] == "text" {
					if text, exists := contentItem["text"]; exists {
						return fmt.Sprintf("%v", text), nil
					}
				}
			}
		}
	}

	return "I retrieved your financial data but had trouble processing it.", nil
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

	// Process query with Claude API + Fi tools available
	// Fi will handle authentication via login flow
	response, err := cs.callClaudeAPIWithTools(query)
	if err != nil {
		log.Printf("Error calling Claude API with tools: %v", err)
		response = "I'm having trouble processing your request right now. Please try again."
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
		"status":  "healthy",
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