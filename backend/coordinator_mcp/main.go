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
	"sync"
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
	fiMCPClient      *client.Client // Legacy single client (kept for backward compatibility)
	fiClients        map[string]*client.Client // Pool of Fi clients per user
	clientsMu        sync.Mutex                // Thread safety for concurrent users
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
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      string `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
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
		fiClients: make(map[string]*client.Client), // Initialize client pool
	}
	
	// Initialize legacy single Fi MCP client for backward compatibility
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

// getClientKey generates the appropriate client pool key based on Firebase UID and userId
func (cs *CoordinatorServer) getClientKey(userId string, firebaseUID string) string {
	if firebaseUID == "" {
		// Legacy mode - preserve existing functionality
		return userId
	}
	// Firebase mode - isolate by app user
	return fmt.Sprintf("%s_%s", firebaseUID, userId)
}

// getOrCreateFiClient returns or creates a persistent Fi MCP client for the given userId and optional firebaseUID
func (cs *CoordinatorServer) getOrCreateFiClient(userId string, firebaseUID string) (*client.Client, error) {
	cs.clientsMu.Lock()
	defer cs.clientsMu.Unlock()
	
	// Generate appropriate client key (supports both legacy and Firebase modes)
	clientKey := cs.getClientKey(userId, firebaseUID)
	
	// Check if client already exists for this client key
	if existingClient, exists := cs.fiClients[clientKey]; exists {
		return existingClient, nil
	}
	
	// Create new Fi MCP client
	var logMsg string
	if firebaseUID != "" {
		logMsg = fmt.Sprintf("Creating new Fi MCP client for Firebase user %s, Fi user: %s", firebaseUID, userId)
	} else {
		logMsg = fmt.Sprintf("Creating new Fi MCP client for user: %s (legacy mode)", userId)
	}
	log.Printf(logMsg)
	fiClient, err := client.NewStreamableHttpClient(cs.fiMCPURL + "/mcp/")
	if err != nil {
		return nil, fmt.Errorf("failed to create Fi MCP client for user %s: %w", userId, err)
	}
	
	// Start and initialize the client
	ctx := context.Background()
	if err := fiClient.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start Fi MCP client for user %s: %w", userId, err)
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
	_, err = fiClient.Initialize(ctx, initRequest)
	if err != nil {
		fiClient.Close()
		return nil, fmt.Errorf("failed to initialize Fi MCP client for user %s: %w", userId, err)
	}
	
	// Store the client in our pool using the appropriate key
	cs.fiClients[clientKey] = fiClient
	if firebaseUID != "" {
		log.Printf("Successfully created and stored Fi MCP client for Firebase user %s, Fi user: %s", firebaseUID, userId)
	} else {
		log.Printf("Successfully created and stored Fi MCP client for user: %s (legacy mode)", userId)
	}
	
	return fiClient, nil
}

// cleanupFiClients closes and removes inactive Fi clients to free resources
func (cs *CoordinatorServer) cleanupFiClients() {
	cs.clientsMu.Lock()
	defer cs.clientsMu.Unlock()
	
	log.Printf("Cleaning up Fi client pool, current clients: %d", len(cs.fiClients))
	for clientKey, client := range cs.fiClients {
		if client != nil {
			client.Close()
			log.Printf("Closed Fi client for key: %s", clientKey)
		}
	}
	cs.fiClients = make(map[string]*client.Client)
}

// cleanupFirebaseUserClients removes all Fi clients for a specific Firebase user
func (cs *CoordinatorServer) cleanupFirebaseUserClients(firebaseUID string) {
	cs.clientsMu.Lock()
	defer cs.clientsMu.Unlock()
	
	var removedClients []string
	for clientKey, client := range cs.fiClients {
		// Check if this client belongs to the Firebase user
		if strings.HasPrefix(clientKey, firebaseUID+"_") {
			if client != nil {
				client.Close()
			}
			delete(cs.fiClients, clientKey)
			removedClients = append(removedClients, clientKey)
		}
	}
	
	log.Printf("Cleaned up %d Fi clients for Firebase user %s: %v", len(removedClients), firebaseUID, removedClients)
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

	// Extract userId, default to 1111111111 for backward compatibility
	userId, _ := arguments["userId"].(string)
	if userId == "" {
		userId = "1111111111"
	}

	// Extract optional firebaseUID for Firebase-enabled clients
	firebaseUID, _ := arguments["firebaseUID"].(string)

	// Call Claude API with tools for intelligent response
	response, err := cs.callClaudeAPIWithTools(query, userId, firebaseUID)
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
	contextData := map[string]any{
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

	// Extract userId, default to 1111111111 for backward compatibility
	userId, _ := arguments["userId"].(string)
	if userId == "" {
		userId = "1111111111"
	}

	// Extract optional firebaseUID for Firebase-enabled clients
	firebaseUID, _ := arguments["firebaseUID"].(string)

	// Call Fi MCP server for specific user
	response, err := cs.callFiMCP(toolName, userId, firebaseUID)
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

// Fi MCP tool call using per-user persistent client to maintain session
func (cs *CoordinatorServer) callFiMCPTool(toolName string, userId string, firebaseUID string) (*mcp.CallToolResult, error) {
	// Get or create Fi client for this specific user (with Firebase isolation)
	fiClient, err := cs.getOrCreateFiClient(userId, firebaseUID)
	if err != nil {
		// Fallback to legacy single client if user-specific client fails
		log.Printf("Failed to get Fi client for user %s, falling back to legacy client: %v", userId, err)
		if cs.fiMCPClient == nil {
			return nil, fmt.Errorf("no Fi MCP client available for user %s", userId)
		}
		fiClient = cs.fiMCPClient
	}

	// Call the Fi tool using the user's persistent client (maintains per-user session)
	ctx := context.Background()
	toolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: map[string]any{},
		},
	}
	result, err := fiClient.CallTool(ctx, toolRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to call Fi MCP tool %s for user %s: %w", toolName, userId, err)
	}

	return result, nil
}

func (cs *CoordinatorServer) callFiMCP(toolName string, userId string, firebaseUID string) (string, error) {
	// Call Fi tool for specific user (will return login_required if not authenticated)
	result, err := cs.callFiMCPTool(toolName, userId, firebaseUID)
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
		log.Printf("Fi requires authentication for user %s, tool: %s", userId, toolName)
		// Return the login_required response as-is so mobile app can handle it
		return resultText, nil
	}

	// Regular successful response
	return resultText, nil
}

// Claude API request with tool calls support
func (cs *CoordinatorServer) callClaudeAPIWithTools(query string, userId string, firebaseUID string) (string, error) {
	if cs.anthropicAPIKey == "" {
		return "Hello! I'm Juno, your financial AI assistant. I'm currently running in demo mode. How can I help you with your finances today?", nil
	}

	// Define Fi tools available to Claude
	tools := []map[string]any{
		{
			"name": "fetch_net_worth",
			"description": "Fetch user's comprehensive net worth including assets, liabilities, and total wealth",
			"input_schema": map[string]any{
				"type": "object",
				"properties": map[string]any{},
				"required": []string{},
			},
		},
		{
			"name": "fetch_bank_transactions",
			"description": "Fetch user's bank transaction history and account details",
			"input_schema": map[string]any{
				"type": "object",
				"properties": map[string]any{},
				"required": []string{},
			},
		},
		{
			"name": "fetch_mf_transactions",
			"description": "Fetch user's mutual fund transactions and investment details",
			"input_schema": map[string]any{
				"type": "object",
				"properties": map[string]any{},
				"required": []string{},
			},
		},
		{
			"name": "fetch_credit_report",
			"description": "Fetch user's credit report including credit score, loan details, and account history",
			"input_schema": map[string]any{
				"type": "object", 
				"properties": map[string]any{},
				"required": []string{},
			},
		},
		{
			"name": "fetch_epf_details",
			"description": "Fetch user's Employee Provident Fund (EPF) details and balance",
			"input_schema": map[string]any{
				"type": "object",
				"properties": map[string]any{},
				"required": []string{},
			},
		},
	}

	requestBody := map[string]any{
		"model":      "claude-3-5-sonnet-20241022",
		"max_tokens": 1000,
		"messages": []map[string]any{
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

	var anthropicResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Handle tool calls if Claude wants to call Fi tools
	if content, ok := anthropicResp["content"].([]any); ok {
		var finalResponse strings.Builder
		
		for _, item := range content {
			if contentItem, ok := item.(map[string]any); ok {
				switch contentItem["type"] {
				case "text":
					if text, exists := contentItem["text"]; exists {
						finalResponse.WriteString(fmt.Sprintf("%v", text))
					}
				case "tool_use":
					// Claude wants to call a tool
					toolName, _ := contentItem["name"].(string)
					toolId, _ := contentItem["id"].(string)
					
					// Call Fi MCP tool for specific user (with Firebase isolation)
					toolResult, err := cs.callFiMCP(toolName, userId, firebaseUID)
					if err != nil {
						log.Printf("Error calling Fi tool %s for user %s (Firebase: %s): %v", toolName, userId, firebaseUID, err)
						toolResult = fmt.Sprintf("Error accessing %s data", toolName)
					}
					
					// Check if Fi returned login_required - if so, return it directly without Claude processing
					if strings.Contains(toolResult, "login_required") {
						return toolResult, nil
					}
					
					// Continue conversation with tool result
					followUpResponse, err := cs.callClaudeAPIWithToolResult(query, toolName, toolId, toolResult, userId, firebaseUID)
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
func (cs *CoordinatorServer) callClaudeAPIWithToolResult(originalQuery, toolName, toolId, toolResult, userId, firebaseUID string) (string, error) {
	log.Printf("Making follow-up Claude API call for user %s (Firebase: %s) with tool result from %s", userId, firebaseUID, toolName)
	requestBody := map[string]any{
		"model":      "claude-3-5-sonnet-20241022", 
		"max_tokens": 1000,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": fmt.Sprintf("You are Juno, a helpful financial AI assistant. The user asked: %s", originalQuery),
			},
			{
				"role": "assistant",
				"content": []map[string]any{
					{
						"type": "tool_use",
						"id":   toolId,
						"name": toolName,
						"input": map[string]any{},
					},
				},
			},
			{
				"role": "user",
				"content": []map[string]any{
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

	var anthropicResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("failed to decode follow-up response: %w", err)
	}

	// Extract text response
	if content, ok := anthropicResp["content"].([]any); ok {
		for _, item := range content {
			if contentItem, ok := item.(map[string]any); ok {
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

connectionLoop:
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
				break connectionLoop
			}
		case "cleanup_user":
			response := cs.processWebSocketCleanup(msg)
			if err := conn.WriteJSON(response); err != nil {
				log.Printf("WebSocket write error: %v", err)
				break connectionLoop
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
				break connectionLoop
			}
		}
	}
}

func (cs *CoordinatorServer) processWebSocketQuery(msg MCPMessage) MCPMessage {
	params, ok := msg.Params.(map[string]any)
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

	// Extract userId from parameters (sent by mobile app dropdown selection)
	userId, ok := params["userId"].(string)
	if !ok || userId == "" {
		// Fallback to default user if no userId provided (backward compatibility)
		userId = "1111111111"
		log.Printf("No userId provided, defaulting to: %s", userId)
	}

	// Extract optional firebaseUID from parameters (sent by Firebase-enabled mobile app)
	firebaseUID, _ := params["firebaseUID"].(string)
	
	if firebaseUID != "" {
		log.Printf("Processing query for Firebase user %s, Fi user: %s", firebaseUID, userId)
	} else {
		log.Printf("Processing query for user: %s (legacy mode)", userId)
	}

	// Process query with Claude API + Fi tools available for specific user
	// Each user will have their own Fi client and authentication session
	response, err := cs.callClaudeAPIWithTools(query, userId, firebaseUID)
	if err != nil {
		log.Printf("Error calling Claude API with tools for user %s: %v", userId, err)
		response = "I'm having trouble processing your request right now. Please try again."
	}

	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  map[string]string{"response": response},
	}
}

func (cs *CoordinatorServer) processWebSocketCleanup(msg MCPMessage) MCPMessage {
	params, ok := msg.Params.(map[string]any)
	if !ok {
		return MCPMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error:   map[string]string{"message": "Invalid parameters"},
		}
	}

	firebaseUID, ok := params["firebaseUID"].(string)
	if !ok || firebaseUID == "" {
		return MCPMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error:   map[string]string{"message": "Missing firebaseUID parameter"},
		}
	}

	// Clean up all Fi clients for this Firebase user
	cs.cleanupFirebaseUserClients(firebaseUID)

	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  map[string]string{"status": "cleanup_completed"},
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