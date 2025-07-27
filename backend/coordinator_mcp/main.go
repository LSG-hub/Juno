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
	mcpServer         *server.MCPServer
	geminiAPIKey      string
	fiMCPURL          string
	contextAgentURL   string
	securityAgentURL  string
	upgrader          websocket.Upgrader
	fiMCPClient       *client.Client // Legacy single client (kept for backward compatibility)
	fiClients         map[string]*client.Client // Pool of Fi clients per user
	contextAgentClients map[string]*client.Client // Pool of Context Agent clients per user
	clientsMu         sync.Mutex                // Thread safety for concurrent users
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
	Tools    []GeminiTool    `json:"tools,omitempty"`
}

type GeminiContent struct {
	Role  string        `json:"role"`
	Parts []GeminiPart  `json:"parts"`
}

type GeminiPart struct {
	Text         string                 `json:"text,omitempty"`
	FunctionCall *GeminiFunctionCall    `json:"functionCall,omitempty"`
}

type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunction `json:"functionDeclarations"`
}

type GeminiFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

// RAG-related structures
type GeminiEmbeddingRequest struct {
	Model   string                    `json:"model"`
	Content GeminiEmbeddingContent    `json:"content"`
	TaskType string                   `json:"taskType,omitempty"`
	Title   string                    `json:"title,omitempty"`
}

type GeminiEmbeddingContent struct {
	Parts []GeminiEmbeddingPart `json:"parts"`
}

type GeminiEmbeddingPart struct {
	Text string `json:"text"`
}

type GeminiEmbeddingResponse struct {
	Embedding GeminiEmbeddingData `json:"embedding"`
}

type GeminiEmbeddingData struct {
	Values []float64 `json:"values"`
}

// Enhanced message structure with embeddings
type EnhancedMessage struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	IsUser    bool      `json:"isUser"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	
	// Embedding fields
	Embedding            []float64 `json:"embedding,omitempty"`
	EmbeddingGeneratedAt *time.Time `json:"embedding_generated_at,omitempty"`
	EmbeddingModel       string    `json:"embedding_model,omitempty"`
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
		geminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		fiMCPURL:        getEnvWithDefault("FI_MCP_URL", "http://localhost:8090"),
		contextAgentURL:  getEnvWithDefault("CONTEXT_AGENT_URL", "http://localhost:8092"),
		securityAgentURL: getEnvWithDefault("SECURITY_AGENT_URL", "http://localhost:8093"),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		fiClients: make(map[string]*client.Client), // Initialize Fi client pool
		contextAgentClients: make(map[string]*client.Client), // Initialize Context Agent client pool
	}
	
	// Debug: Check if Gemini API key is loaded
	if cs.geminiAPIKey == "" {
		log.Printf("WARNING: GEMINI_API_KEY environment variable is not set - running in demo mode")
	} else {
		log.Printf("Gemini API key loaded successfully (length: %d characters)", len(cs.geminiAPIKey))
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

// getOrCreateContextAgentClient returns or creates a persistent Context Agent MCP client for the given userId and optional firebaseUID
func (cs *CoordinatorServer) getOrCreateContextAgentClient(userId string, firebaseUID string) (*client.Client, error) {
	cs.clientsMu.Lock()
	defer cs.clientsMu.Unlock()
	
	// Generate appropriate client key (supports both legacy and Firebase modes)
	clientKey := cs.getClientKey(userId, firebaseUID)
	
	// Check if client already exists for this client key
	if existingClient, exists := cs.contextAgentClients[clientKey]; exists {
		return existingClient, nil
	}
	
	// Create new Context Agent MCP client
	var logMsg string
	if firebaseUID != "" {
		logMsg = fmt.Sprintf("Creating new Context Agent MCP client for Firebase user %s, Fi user: %s", firebaseUID, userId)
	} else {
		logMsg = fmt.Sprintf("Creating new Context Agent MCP client for user: %s (legacy mode)", userId)
	}
	log.Printf(logMsg)
	contextAgentClient, err := client.NewStreamableHttpClient(cs.contextAgentURL + "/mcp/")
	if err != nil {
		return nil, fmt.Errorf("failed to create Context Agent MCP client for user %s: %w", userId, err)
	}
	
	// Start and initialize the client
	ctx := context.Background()
	if err := contextAgentClient.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start Context Agent MCP client for user %s: %w", userId, err)
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
	_, err = contextAgentClient.Initialize(ctx, initRequest)
	if err != nil {
		contextAgentClient.Close()
		return nil, fmt.Errorf("failed to initialize Context Agent MCP client for user %s: %w", userId, err)
	}
	
	// Store the client in our pool using the appropriate key
	cs.contextAgentClients[clientKey] = contextAgentClient
	if firebaseUID != "" {
		log.Printf("Successfully created and stored Context Agent MCP client for Firebase user %s, Fi user: %s", firebaseUID, userId)
	} else {
		log.Printf("Successfully created and stored Context Agent MCP client for user: %s (legacy mode)", userId)
	}
	
	return contextAgentClient, nil
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

	// General conversation tool (routes to Context Agent)
	cs.mcpServer.AddTool(
		mcp.NewTool("general_conversation",
			mcp.WithDescription("Handle general conversation through Context Agent"),
			mcp.WithString("firebase_uid",
				mcp.Description("Firebase user ID for data isolation"),
				mcp.Required(),
			),
			mcp.WithString("fi_user_id",
				mcp.Description("Fi user ID (1010101010-9999999999)"),
				mcp.Required(),
			),
			mcp.WithString("query",
				mcp.Description("User's conversation query"),
				mcp.Required(),
			),
			mcp.WithString("message_id",
				mcp.Description("Unique message identifier"),
				mcp.Required(),
			),
		),
		cs.handleGeneralConversation,
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

	// Load chat history from Context Agent (for mobile app UI button)
	cs.mcpServer.AddTool(
		mcp.NewTool("load_chat_history",
			mcp.WithDescription("Load chat history for a specific Fi user from Context Agent storage"),
			mcp.WithString("firebase_uid",
				mcp.Description("Firebase user ID for data isolation"),
				mcp.Required(),
			),
			mcp.WithString("fi_user_id",
				mcp.Description("Fi user ID (1010101010-9999999999)"),
				mcp.Required(),
			),
		),
		cs.handleLoadChatHistory,
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

	// Get RAG context from Context Agent first
	contextResult := cs.getRAGContextFromContextAgent(query, userId, firebaseUID)

	// Call Gemini API with tools and RAG context
	response, err := cs.callGeminiAPIWithTools(query, userId, firebaseUID, contextResult)
	if err != nil {
		log.Printf("Error calling Gemini API: %v", err)
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

func (cs *CoordinatorServer) handleLoadChatHistory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	
	firebaseUID, ok := arguments["firebase_uid"].(string)
	if !ok || firebaseUID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: firebase_uid parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	fiUserID, ok := arguments["fi_user_id"].(string)
	if !ok || fiUserID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: fi_user_id parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	// Call Context Agent MCP to load chat history
	contextClient, err := cs.getOrCreateContextAgentClient(fiUserID, firebaseUID)
	if err != nil {
		log.Printf("Error getting Context Agent client: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error connecting to Context Agent: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Call load_chat_history tool on Context Agent
	toolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "load_chat_history",
			Arguments: map[string]interface{}{
				"firebase_uid": firebaseUID,
				"fi_user_id":   fiUserID,
			},
		},
	}

	result, err := contextClient.CallTool(ctx, toolRequest)
	if err != nil {
		log.Printf("Error calling Context Agent load_chat_history: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error loading chat history: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	log.Printf("Successfully loaded chat history for user %s/%s", firebaseUID, fiUserID)

	// Return the result from Context Agent
	return result, nil
}

// Handle general conversation by routing to Context Agent
func (cs *CoordinatorServer) handleGeneralConversation(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()

	firebaseUID, ok := arguments["firebase_uid"].(string)
	if !ok || firebaseUID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: firebase_uid parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	fiUserID, ok := arguments["fi_user_id"].(string)
	if !ok || fiUserID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: fi_user_id parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	query, ok := arguments["query"].(string)
	if !ok || query == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: query parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	messageID, ok := arguments["message_id"].(string)
	if !ok || messageID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: message_id parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	// Route to Context Agent MCP for general conversation
	contextClient, err := cs.getOrCreateContextAgentClient(fiUserID, firebaseUID)
	if err != nil {
		log.Printf("Error getting Context Agent client: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "I'm having trouble processing your request right now. Please try again.",
				},
			},
			IsError: true,
		}, nil
	}

	// Call general_conversation tool on Context Agent
	toolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "general_conversation",
			Arguments: map[string]interface{}{
				"firebase_uid": firebaseUID,
				"fi_user_id":   fiUserID,
				"query":        query,
				"message_id":   messageID,
			},
		},
	}

	result, err := contextClient.CallTool(ctx, toolRequest)
	if err != nil {
		log.Printf("Error calling Context Agent general_conversation: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "I'm having trouble generating a response right now. Please try again.",
				},
			},
			IsError: true,
		}, nil
	}

	log.Printf("Successfully routed general conversation for user %s/%s", firebaseUID, fiUserID)

	// Return the result from Context Agent
	return result, nil
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

// Context Agent MCP tool call using per-user persistent client
func (cs *CoordinatorServer) callContextAgentTool(toolName string, userId string, firebaseUID string, functionArgs map[string]interface{}) (*mcp.CallToolResult, error) {
	// Get or create Context Agent client for this specific user (with Firebase isolation)
	contextAgentClient, err := cs.getOrCreateContextAgentClient(userId, firebaseUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Context Agent client for user %s: %v", userId, err)
	}

	// Prepare arguments - always include user_id, and merge any Gemini function call args
	arguments := map[string]any{
		"user_id": userId, // Always pass user_id to context agent tools
	}
	
	// Merge in any additional arguments from Gemini's function call
	for key, value := range functionArgs {
		arguments[key] = value
	}

	// Call the Context Agent tool using the user's persistent client
	ctx := context.Background()
	toolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	}
	result, err := contextAgentClient.CallTool(ctx, toolRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to call Context Agent MCP tool %s for user %s: %w", toolName, userId, err)
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

func (cs *CoordinatorServer) callContextAgent(toolName string, userId string, firebaseUID string, functionArgs map[string]interface{}) (string, error) {
	// Call Context Agent tool for specific user
	result, err := cs.callContextAgentTool(toolName, userId, firebaseUID, functionArgs)
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
		return fmt.Sprintf("Context Agent MCP Result: %v", result.Content), nil
	}

	// Regular successful response
	return resultText, nil
}

// RAG functionality moved to Context Agent MCP - Coordinator now calls Context Agent for all RAG operations


// Gemini API request with function calls support and context from Context Agent
func (cs *CoordinatorServer) callGeminiAPIWithTools(query string, userId string, firebaseUID string, contextResult map[string]interface{}) (string, error) {
	if cs.geminiAPIKey == "" {
		return "Hello! I'm Juno, your helpful AI companion. I'm currently running in demo mode. How can I help you today?", nil
	}

	// Define Fi tools available to Gemini with better descriptions
	tools := []GeminiTool{
		{
			FunctionDeclarations: []GeminiFunction{
				// Fi MCP Tools - USE THESE WHEN USER ASKS ABOUT FINANCES
				{
					Name:        "fetch_net_worth",
					Description: "REQUIRED: Call this when user asks about net worth, total wealth, financial status, affordability, budget planning, or financial overview. Returns comprehensive financial picture including assets and liabilities.",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{},
						"required": []any{},
					},
				},
				{
					Name:        "fetch_bank_transactions",
					Description: "REQUIRED: Call this when user asks about bank accounts, transactions, spending patterns, cash flow, income, expenses, or banking details. Essential for budget analysis.",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{},
						"required": []any{},
					},
				},
				{
					Name:        "fetch_mf_transactions",
					Description: "REQUIRED: Call this when user asks about investments, mutual funds, portfolio, SIPs, returns, or investment planning. Critical for investment advice.",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{},
						"required": []any{},
					},
				},
				{
					Name:        "fetch_credit_report",
					Description: "REQUIRED: Call this when user asks about loans, credit score, debt, EMIs, credit history, or borrowing capacity. Essential for credit-related queries.",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{},
						"required": []any{},
					},
				},
				{
					Name:        "fetch_epf_details",
					Description: "REQUIRED: Call this when user asks about EPF, provident fund, retirement savings, or PF balance. Important for retirement planning.",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{},
						"required": []any{},
					},
				},
			},
		},
	}

	// Build prompt with context from Context Agent (passed as parameter)
	promptText := cs.buildPromptWithContext(query, contextResult)

	requestBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Role: "user",
				Parts: []GeminiPart{
					{
						Text: promptText,
					},
				},
			},
		},
		Tools: tools,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent?key=%s", cs.geminiAPIKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Handle function calls if Gemini wants to call Fi tools
	if len(geminiResp.Candidates) > 0 {
		candidate := geminiResp.Candidates[0]
		
		// Check if there are any function calls to batch
		var functionCalls []GeminiFunctionCall
		var textParts []string
		
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				textParts = append(textParts, part.Text)
			} else if part.FunctionCall != nil {
				functionCalls = append(functionCalls, *part.FunctionCall)
			}
		}
		
		// If there are function calls, batch them all and make one final Gemini call
		if len(functionCalls) > 0 {
			return cs.handleBatchedFunctionCalls(query, functionCalls, userId, firebaseUID, contextResult)
		}
		
		// If no function calls, just return the text response
		if len(textParts) > 0 {
			return strings.Join(textParts, "\n"), nil
		}
	}

	defaultResponse := "I'm having trouble generating a response right now. Please try again."
	return defaultResponse, nil
}

// buildPromptWithContext creates a prompt with context from Context Agent
func (cs *CoordinatorServer) buildPromptWithContext(query string, contextResult map[string]interface{}) string {
	var relevantContext []map[string]interface{}
	
	// Extract relevant context from contextResult if available
	if contextResult != nil {
		if contexts, ok := contextResult["similar_messages"].([]interface{}); ok {
			for _, ctx := range contexts {
				if ctxMap, ok := ctx.(map[string]interface{}); ok {
					relevantContext = append(relevantContext, ctxMap)
				}
			}
		}
	}
	
	if len(relevantContext) > 0 {
		var historicalContext string
		var recentContext string
		
		for _, ctx := range relevantContext {
			if text, ok := ctx["text"].(string); ok {
				if isUser, ok := ctx["is_user"].(bool); ok {
					role := "Assistant"
					if isUser {
						role = "User"
					}
					
					contextLine := fmt.Sprintf("- %s: %s\n", role, text)
					
					// Separate historical context from recent conversation
					if source, ok := ctx["source"].(string); ok && source == "historical" {
						historicalContext += contextLine
					} else {
						recentContext += contextLine
					}
				}
			}
		}
		
		// Build context with clear separation
		var contextText string
		if historicalContext != "" {
			contextText += "RELEVANT PAST CONVERSATIONS:\n" + historicalContext + "\n"
		}
		if recentContext != "" {
			contextText += "RECENT CONVERSATION:\n" + recentContext
		}
		
		return fmt.Sprintf(`You are Juno, a warm, empathetic, and intelligent AI companion. You're designed to be a supportive friend who genuinely cares about the user's well-being across all aspects of their life.

## Your Core Personality:
- **Empathetic & Caring**: Always acknowledge emotions and provide emotional support when needed
- **Encouraging & Positive**: Help users feel motivated and optimistic about their goals
- **Intelligent & Helpful**: Provide thoughtful, practical advice across diverse topics
- **Conversational & Natural**: Chat like a close friend who remembers previous conversations
- **Balanced**: You're an all-around companion first, with financial expertise when relevant

## Your Capabilities:
- **Life Companion**: Relationships, mental health, career advice, learning, hobbies, travel, health
- **Problem Solver**: Help with decisions, planning, creative projects, technical questions
- **Financial Advisor**: Access real financial data when it would help answer the user's question
- **Emotional Support**: Listen, validate feelings, offer comfort and encouragement

## When to Use Financial Tools:
- User explicitly asks about their finances ("What's my net worth?", "Show my transactions")
- User needs financial data to make decisions ("Can I afford this house?", "Should I invest more?")
- User asks about purchases, investments, or major financial decisions
- Context suggests financial information would be helpful for a complete answer

## Interaction Guidelines:
- **For emotional queries**: Lead with empathy, validate feelings, offer support
- **For general topics**: Be helpful and engaging without forcing financial topics
- **For non-financial requests**: Focus completely on the user's actual request (recipes, advice, etc.) - do NOT redirect to financial topics
- **For financial decisions**: Proactively use financial tools to give informed advice
- **Remember context**: Reference previous conversations naturally
- **Stay on topic**: If user asks for recipes, give recipes. If they ask for travel advice, give travel advice. Only mention finances when truly relevant.

%s

CURRENT USER QUERY: %s

IMPORTANT: If this query is about finances, affordability, money, investments, transactions, loans, or financial planning - YOU MUST call the appropriate financial tools first before responding. Use the real financial data to provide accurate, personalized advice based on the user's actual financial situation.

Respond as Juno would - warm, helpful, and use your financial tools when they would help provide better advice.`, contextText, query)
	} else {
		return fmt.Sprintf(`You are Juno, a warm, empathetic, and intelligent AI companion. You're designed to be a supportive friend who genuinely cares about the user's well-being across all aspects of their life.

## Your Core Personality:
- **Empathetic & Caring**: Always acknowledge emotions and provide emotional support when needed
- **Encouraging & Positive**: Help users feel motivated and optimistic about their goals
- **Intelligent & Helpful**: Provide thoughtful, practical advice across diverse topics
- **Conversational & Natural**: Chat like a close friend who remembers previous conversations
- **Balanced**: You're an all-around companion first, with financial expertise when relevant

## Your Capabilities:
- **Life Companion**: Relationships, mental health, career advice, learning, hobbies, travel, health
- **Problem Solver**: Help with decisions, planning, creative projects, technical questions
- **Financial Advisor**: Access real financial data when it would help answer the user's question
- **Emotional Support**: Listen, validate feelings, offer comfort and encouragement

## CRITICAL: When to Use Financial Tools (YOU MUST CALL THESE):
- **ALWAYS call fetch_net_worth when**: User mentions affordability, budget, finances, money, buying anything, financial status, wealth
- **ALWAYS call fetch_bank_transactions when**: User asks about spending, income, transactions, cash flow, banking, account details
- **ALWAYS call fetch_mf_transactions when**: User mentions investments, mutual funds, portfolio, SIPs, returns, investment planning
- **ALWAYS call fetch_credit_report when**: User asks about loans, credit, debt, EMIs, borrowing, credit score
- **ALWAYS call fetch_epf_details when**: User mentions EPF, PF, provident fund, retirement planning

## Location-Aware Financial Advice:
- The user's location context will be provided - use this to give location-specific financial advice
- Consider local market conditions, regional financial products, and location-based recommendations
- Integrate location data with financial data for comprehensive advice

## Interaction Guidelines:
- **For financial queries**: FIRST call the appropriate financial tools, THEN provide advice based on real data
- **For location + financial queries**: Use both location context AND financial data to provide targeted advice
- **Important**: DO NOT give financial advice without calling the tools first to get real data
- **Remember**: You have access to real financial data - use it to provide accurate, personalized advice

CURRENT USER QUERY: %s

IMPORTANT: If this query is about finances, affordability, money, investments, transactions, loans, or financial planning - YOU MUST call the appropriate financial tools first before responding. Use the real financial data to provide accurate, personalized advice based on the user's actual financial situation.

Respond as Juno would - warm, helpful, and use your financial tools when they would help provide better advice.`, query)
	}
}

// handleBatchedFunctionCalls executes all Fi tool calls and makes one final Gemini call with all results
func (cs *CoordinatorServer) handleBatchedFunctionCalls(originalQuery string, functionCalls []GeminiFunctionCall, userId, firebaseUID string, contextResult map[string]interface{}) (string, error) {
	log.Printf("Handling %d batched function calls for user %s (Firebase: %s)", len(functionCalls), userId, firebaseUID)
	
	// Execute all Fi tool calls and collect results
	var toolResults []map[string]string
	
	for _, functionCall := range functionCalls {
		functionName := functionCall.Name
		log.Printf("Calling Fi tool: %s", functionName)
		
		// Call Fi MCP tool
		toolResult, err := cs.callFiMCP(functionName, userId, firebaseUID)
		if err != nil {
			log.Printf("Error calling Fi tool %s: %v", functionName, err)
			toolResult = fmt.Sprintf("Error accessing %s data", functionName)
		}
		
		// Check if Fi returned login_required - if so, return it directly
		if strings.Contains(toolResult, "login_required") {
			return toolResult, nil
		}
		
		// Store the tool result
		toolResults = append(toolResults, map[string]string{
			"function_name": functionName,
			"result":       toolResult,
		})
	}
	
	// Build the conversation with all tool results for one final Gemini call
	return cs.callGeminiAPIWithAllToolResults(originalQuery, toolResults, contextResult)
}

// callGeminiAPIWithAllToolResults makes one final Gemini call with all tool results
func (cs *CoordinatorServer) callGeminiAPIWithAllToolResults(originalQuery string, toolResults []map[string]string, contextResult map[string]interface{}) (string, error) {
	log.Printf("Making final Gemini call with %d tool results", len(toolResults))
	
	// Build conversation with context and all tool results
	promptText := cs.buildPromptWithContext(originalQuery, contextResult)
	
	// Add all tool results to the conversation
	var toolResultsText strings.Builder
	toolResultsText.WriteString("\n\nFINANCIAL DATA RETRIEVED:\n")
	for _, result := range toolResults {
		toolResultsText.WriteString(fmt.Sprintf("\n**%s:**\n%s\n", result["function_name"], result["result"]))
	}
	toolResultsText.WriteString("\nBased on this financial data, please provide a comprehensive and helpful response to the user's query.")
	
	finalPrompt := promptText + toolResultsText.String()
	
	// Create request without tools (pure conversation)
	requestBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Role: "user",
				Parts: []GeminiPart{
					{
						Text: finalPrompt,
					},
				},
			},
		},
		// No tools - this is the final response generation
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal final request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent?key=%s", cs.geminiAPIKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create final request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make final request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("final API returned status %d", resp.StatusCode)
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to decode final response: %w", err)
	}

	// Extract the final response
	if len(geminiResp.Candidates) > 0 {
		candidate := geminiResp.Candidates[0]
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				return part.Text, nil
			}
		}
	}

	return "I retrieved your financial data but had trouble generating a comprehensive response.", nil
}

// Follow-up call to Gemini with function result (DEPRECATED - use handleBatchedFunctionCalls instead)
func (cs *CoordinatorServer) callGeminiAPIWithFunctionResult(originalQuery, functionName, functionResult, userId, firebaseUID string) (string, error) {
	log.Printf("Making follow-up Gemini API call for user %s (Firebase: %s) with function result from %s", userId, firebaseUID, functionName)
	requestBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Role: "user",
				Parts: []GeminiPart{
					{
						Text: fmt.Sprintf("You are Juno, a helpful AI companion. The user asked: %s", originalQuery),
					},
				},
			},
			{
				Role: "model",
				Parts: []GeminiPart{
					{
						FunctionCall: &GeminiFunctionCall{
							Name: functionName,
							Args: map[string]interface{}{},
						},
					},
				},
			},
			{
				Role: "user",
				Parts: []GeminiPart{
					{
						Text: fmt.Sprintf("Function result: %s", functionResult),
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal follow-up request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent?key=%s", cs.geminiAPIKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create follow-up request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make follow-up request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("follow-up API returned status %d", resp.StatusCode)
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to decode follow-up response: %w", err)
	}

	// Extract text response
	if len(geminiResp.Candidates) > 0 {
		candidate := geminiResp.Candidates[0]
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				return part.Text, nil
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
	
	// Extract optional location_context from parameters (sent by location-aware mobile app)
	var locationContext map[string]interface{}
	if locationData, ok := params["location_context"].(map[string]interface{}); ok {
		locationContext = locationData
		log.Printf("Location context received: %+v", locationContext)
	}
	
	if firebaseUID != "" {
		log.Printf("Processing query for Firebase user %s, Fi user: %s", firebaseUID, userId)
	} else {
		log.Printf("Processing query for user: %s (legacy mode)", userId)
	}

	// STEP 1: ALWAYS get RAG context from Context Agent first (for ALL queries)
	contextResult := cs.getRAGContextFromContextAgent(query, userId, firebaseUID)
	
	// Add location context to the query if available
	queryWithLocation := query
	if locationContext != nil && len(locationContext) > 0 {
		locationText := ""
		if city, ok := locationContext["city"].(string); ok && city != "" {
			locationText += fmt.Sprintf("City: %s", city)
		}
		if state, ok := locationContext["state"].(string); ok && state != "" {
			if locationText != "" {
				locationText += ", "
			}
			locationText += fmt.Sprintf("State: %s", state)
		}
		if country, ok := locationContext["country"].(string); ok && country != "" {
			if locationText != "" {
				locationText += ", "
			}
			locationText += fmt.Sprintf("Country: %s", country)
		}
		if locationText != "" {
			queryWithLocation = fmt.Sprintf("USER QUERY: %s\n\nUSER'S CURRENT LOCATION: %s\n\nPlease consider the user's location when providing advice, especially for financial decisions, local market conditions, and region-specific recommendations.", query, locationText)
		}
	}

	// STEP 2: Always use Gemini with Fi tools and let it intelligently decide when to call them
	log.Printf("Processing query with Gemini + Fi tools (intelligent function calling)")
	response, err := cs.callGeminiAPIWithTools(queryWithLocation, userId, firebaseUID, contextResult)
	if err != nil {
		log.Printf("Error calling Gemini API with tools for user %s: %v", userId, err)
		response = "I'm having trouble processing your request right now. Please try again."
	}
	
	// Store both user message and assistant response in Context Agent for RAG
	if response != "" && !strings.Contains(response, "login_required") {
		cs.storeFinancialConversationInContextAgent(query, response, userId, firebaseUID)
	}

	return MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  map[string]string{"response": response},
	}
}

// getRAGContextFromContextAgent retrieves RAG context from Context Agent for any query
func (cs *CoordinatorServer) getRAGContextFromContextAgent(query string, userId string, firebaseUID string) map[string]interface{} {
	log.Printf("Retrieving RAG context from Context Agent for query: %s", query)
	
	// Get or create Context Agent client
	contextClient, err := cs.getOrCreateContextAgentClient(userId, firebaseUID)
	if err != nil {
		log.Printf("Warning: Failed to get Context Agent client: %v", err)
		return nil
	}

	// Call hybrid RAG search on Context Agent
	toolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "search_similar_conversations_hybrid",
			Arguments: map[string]interface{}{
				"firebase_uid": firebaseUID,
				"fi_user_id":   userId,
				"query":        query,
			},
		},
	}

	result, err := contextClient.CallTool(context.Background(), toolRequest)
	if err != nil {
		log.Printf("Warning: Failed to get RAG context: %v", err)
		return nil
	}

	// Parse the result
	var responseText strings.Builder
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			responseText.WriteString(textContent.Text)
		}
	}

	var contextResult map[string]interface{}
	if err := json.Unmarshal([]byte(responseText.String()), &contextResult); err != nil {
		log.Printf("Warning: Failed to parse RAG context: %v", err)
		return nil
	}

	log.Printf("Successfully retrieved RAG context from Context Agent")
	return contextResult
}

// storeFinancialConversationInContextAgent stores both user query and assistant response in Context Agent
func (cs *CoordinatorServer) storeFinancialConversationInContextAgent(userQuery, assistantResponse, userId, firebaseUID string) {
	log.Printf("Storing financial conversation in Context Agent for user %s", userId)
	
	// Get or create Context Agent client
	contextClient, err := cs.getOrCreateContextAgentClient(userId, firebaseUID)
	if err != nil {
		log.Printf("Warning: Failed to get Context Agent client for storage: %v", err)
		return
	}

	now := time.Now()
	
	// Store user message
	userMessageID := fmt.Sprintf("user_%d", now.UnixNano())
	userToolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "process_message_context",
			Arguments: map[string]interface{}{
				"firebase_uid": firebaseUID,
				"fi_user_id":   userId,
				"message_id":   userMessageID,
				"text":         userQuery,
				"is_user":      true,
				"timestamp":    now.Format(time.RFC3339),
				"status":       "sent",
			},
		},
	}

	_, err = contextClient.CallTool(context.Background(), userToolRequest)
	if err != nil {
		log.Printf("Warning: Failed to store user message: %v", err)
	}

	// Store assistant response
	responseMessageID := fmt.Sprintf("assistant_%d", now.UnixNano()+1)
	assistantToolRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "process_message_context",
			Arguments: map[string]interface{}{
				"firebase_uid": firebaseUID,
				"fi_user_id":   userId,
				"message_id":   responseMessageID,
				"text":         assistantResponse,
				"is_user":      false,
				"timestamp":    now.Add(time.Millisecond).Format(time.RFC3339),
				"status":       "sent",
			},
		},
	}

	_, err = contextClient.CallTool(context.Background(), assistantToolRequest)
	if err != nil {
		log.Printf("Warning: Failed to store assistant response: %v", err)
	} else {
		log.Printf("Successfully stored financial conversation in Context Agent")
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

// RAG Functions - Centralized in Context Agent MCP
// All RAG functionality (embedding generation, storage, retrieval) is handled by Context Agent MCP
// Coordinator now acts as a pure orchestrator that calls Context Agent for RAG operations

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

	port := getEnvWithDefault("PORT", "8091")
	log.Printf("Starting Coordinator MCP Server on port %s", port)
	log.Printf("WebSocket endpoint: ws://localhost:%s/ws", port)
	log.Printf("MCP endpoint: http://localhost:%s/mcp/", port)
	log.Printf("Health endpoint: http://localhost:%s/health", port)

	if err := http.ListenAndServe(":"+port, httpMux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}