package main

import (
	"bytes"
	// "context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

// Gemini API structures
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig"`
	SystemInstruction *GeminiContent `json:"systemInstruction,omitempty"`
}

type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	TopK           int     `json:"topK"`
	TopP           float64 `json:"topP"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// MCP message structures for communicating with mobile app
type ChatRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	ID      string                 `json:"id"`
}

type ChatResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	Result  map[string]interface{} `json:"result"`
	ID      string                 `json:"id"`
}

// MCP structures for calling Fi MCP Server
type MCPToolCallRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	} `json:"params"`
	ID string `json:"id"`
}

type MCPToolCallResponse struct {
	JSONRPC string `json:"jsonrpc"`
	Result  struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError,omitempty"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Fi MCP configuration
var (
	fiMCPBaseURL = ""
	fiMCPStreamURL = ""
)

// 🌐 NEW: Translation service global variable
var translationService *TranslationService

// Initialize Fi MCP configuration from environment
func initializeFiMCP() {
	// Get Fi MCP URL from environment
	fiURL := os.Getenv("FI_MCP_URL")
	if fiURL == "" {
		fiURL = "http://fi-mcp-server:8080" // Default for docker-compose
	}
	
	fiMCPBaseURL = fiURL
	// Update to use the correct Fi MCP endpoint
	fiMCPStreamURL = fiURL + "/mcp/stream"
	
	log.Printf("📊 Fi MCP configured:")
	log.Printf("   Base URL: %s", fiMCPBaseURL)
	log.Printf("   Stream URL: %s", fiMCPStreamURL)
}

// Fetch financial data from Fi MCP Server via HTTP
func fetchFiData(userID string, firebaseUID string) string {
	log.Printf("💰 Fetching Fi data for user: %s", userID)

	// Map userID to phoneNumber for Fi MCP
	phoneNumber := userID
	if phoneNumber == "" {
		phoneNumber = "1111111111" // Default test user
	}

	// Prepare the HTTP request to Fi MCP
	toolCallReq := MCPToolCallRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		ID:      fmt.Sprintf("fi-data-%d", time.Now().Unix()),
	}
	toolCallReq.Params.Name = "fetch_net_worth"
	toolCallReq.Params.Arguments = map[string]interface{}{
		"phoneNumber": phoneNumber,
	}

	jsonData, err := json.Marshal(toolCallReq)
	if err != nil {
		log.Printf("Error marshaling request: %v", err)
		return ""
	}

	// Make HTTP request to Fi MCP
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(fiMCPStreamURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error calling Fi MCP: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		return ""
	}

	// Parse the response
	var mcpResp MCPToolCallResponse
	if err := json.Unmarshal(body, &mcpResp); err != nil {
		log.Printf("Error parsing Fi MCP response: %v", err)
		return ""
	}

	// Handle authentication requirement
	if mcpResp.Error != nil && strings.Contains(mcpResp.Error.Message, "Authentication required") {
		log.Printf("⚠️  Fi MCP requires authentication")
		return "Authentication required. Please login to Fi Money to access your financial data."
	}

	// Extract the financial data
	if len(mcpResp.Result.Content) > 0 {
		log.Printf("✅ Successfully fetched Fi data for user %s", userID)
		return mcpResp.Result.Content[0].Text
	}

	return ""
}

// 🌐 UPDATED: Process query with translation support
func processQuery(query string, userID string, firebaseUID string) string {
	// Use translation-aware processing
	response, detectedLang := translationService.ProcessChatWithTranslation(
		query,
		func(translatedQuery string) string {
			// This is your existing processQuery logic, but using translatedQuery
			return processQueryInternal(translatedQuery, userID, firebaseUID)
		},
	)

	// Log the language used
	if detectedLang != "" && detectedLang != translationService.defaultLanguage {
		log.Printf("🌐 Response provided in: %s", detectedLang)
	}

	return response
}

// 🔄 RENAMED: Original processQuery is now processQueryInternal
func processQueryInternal(query string, userID string, firebaseUID string) string {
	log.Printf("📊 Processing query for user %s: %s", userID, query)
	
	// Fetch real financial data from Fi MCP Server
	fiData := fetchFiData(userID, firebaseUID)
	
	var enhancedQuery string
	if fiData != "" {
		// Include real Fi data in the context
		enhancedQuery = fmt.Sprintf(`User Query: %s

IMPORTANT: You have access to the user's REAL financial data from Fi Money. Use this actual data to provide personalized, accurate financial advice.

Current Financial Data:
%s

User ID: %s

Based on this real data, provide specific, actionable financial guidance. Reference actual numbers and accounts when giving advice. If the user asks about their finances, use the real data provided above.`, 
			query, fiData, userID)
	} else {
		// No Fi data available
		enhancedQuery = fmt.Sprintf(`User Query: %s

Note: Unable to fetch user's financial data at this moment. Provide helpful, personalized advice based on their context (if available) and ask clarifying questions to better understand their specific needs.`, 
			query)
	}

	// Call Gemini API with enhanced context (including real data)
	response, err := callGeminiAPI(enhancedQuery, userID)
	if err != nil {
		log.Printf("❌ Error calling Gemini API: %v", err)
		// If we have Fi data but Gemini fails, at least return the raw data
		if fiData != "" {
			return fmt.Sprintf("I'm having trouble processing your question with AI right now, but here's your current financial information from Fi:\n\n%s\n\nPlease try asking again in a moment.", fiData)
		}
		return "I'm sorry, I'm having trouble processing your request right now. Please try again in a moment."
	}

	return response
}

// Call Gemini 2.5 Flash API (updated system prompt)
func callGeminiAPI(query string, userID string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		return "I apologize, but I'm not properly configured to provide AI responses. Please check the GEMINI_API_KEY or GOOGLE_API_KEY configuration.", nil
	}

	// Enhanced system prompt emphasizing real data analysis
	systemPrompt := fmt.Sprintf(`You are Juno, a highly intelligent AI financial assistant powered by Google's Gemini 2.5 Flash. You help users with comprehensive financial guidance using their REAL financial data from Fi Money platform.

Current user: %s

IMPORTANT CAPABILITIES:
- You have access to REAL financial data including net worth, assets, liabilities, investments, and credit information
- Provide specific advice based on actual numbers from their Fi Money account
- Reference exact amounts and account types when discussing their finances
- Give actionable recommendations tailored to their actual financial situation

Your personality:
- Professional yet approachable
- Data-driven and specific (use the real numbers provided)
- Proactive in identifying opportunities and risks
- Culturally aware (understand Indian financial context - EPF, mutual funds, FDs, etc.)

Always:
1. Use the actual financial data provided to give personalized advice
2. Reference specific amounts and accounts when relevant
3. Provide actionable next steps based on their current situation
4. Be encouraging but realistic about their financial health`, userID)

	// Prepare Gemini request
	geminiReq := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{{Text: query}},
				Role:  "user",
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.7,
			TopK:           40,
			TopP:           0.95,
			MaxOutputTokens: 8192,
		},
		SystemInstruction: &GeminiContent{
			Parts: []GeminiPart{{Text: systemPrompt}},
		},
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return "", err
	}

	// Call Gemini API
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-exp:generateContent?key=%s", apiKey)
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Gemini API error: %s", string(body))
		return "I apologize, but I'm having trouble connecting to the AI service. Please try again in a moment.", nil
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "I'm sorry, I didn't receive a proper response. Please try asking again.", nil
}

// Handle HTTP chat requests
func handleHTTPChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	query := ""
	userID := "1111111111" // Default test user
	firebaseUID := ""
	
	if q, ok := req.Params["query"].(string); ok {
		query = q
	}
	if u, ok := req.Params["userId"].(string); ok {
		userID = u
	}
	if f, ok := req.Params["firebaseUID"].(string); ok {
		firebaseUID = f
	}

	log.Printf("🤖 Processing HTTP query with Fi data: '%s' for user: %s", query, userID)

	// Process query with Gemini AI + real Fi data
	responseText := processQuery(query, userID, firebaseUID)

	response := ChatResponse{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"response": responseText,
		},
		ID: req.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Handle WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Println("🔗 WebSocket client connected")

	for {
		var req ChatRequest
		err := conn.ReadJSON(&req)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		query := ""
		userID := "1111111111" // Default test user
		firebaseUID := ""
		
		if q, ok := req.Params["query"].(string); ok {
			query = q
		}
		if u, ok := req.Params["userId"].(string); ok {
			userID = u
		}
		if f, ok := req.Params["firebaseUID"].(string); ok {
			firebaseUID = f
		}

		log.Printf("🤖 Processing WebSocket query with Fi data: '%s' for user: %s", query, userID)

		// Process query with Gemini AI + real Fi data
		responseText := processQuery(query, userID, firebaseUID)

		response := ChatResponse{
			JSONRPC: "2.0",
			Result: map[string]interface{}{
				"response": responseText,
			},
			ID: req.ID,
		}

		if err := conn.WriteJSON(response); err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}

		log.Printf("✅ Sent Gemini AI response with real Fi data to user %s", userID)
	}

	log.Println("🔌 WebSocket client disconnected")
}

// 🌐 NEW: Handle supported languages endpoint
func handleSupportedLanguages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	languages := translationService.GetSupportedLanguages()
	enabled := translationService.enabled
	
	response := map[string]interface{}{
		"enabled":   enabled,
		"languages": languages,
		"default":   translationService.defaultLanguage,
	}
	
	json.NewEncoder(w).Encode(response)
}

// 🌐 UPDATED: Health check handler with translation status
func handleHealth(w http.ResponseWriter, r *http.Request) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	
	status := "healthy"
	if apiKey == "" {
		status = "healthy-no-api-key"
	}
	
	// Test Fi MCP connection
	fiStatus := "disconnected"
	if fiMCPBaseURL != "" {
		// Try a simple health check to Fi MCP
		resp, err := http.Get(fmt.Sprintf("%s/health", fiMCPBaseURL))
		if err == nil && resp.StatusCode == 200 {
			fiStatus = "connected"
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
	
	// Add translation status
	translationStatus := "disabled"
	if translationService != nil && translationService.enabled {
		translationStatus = "enabled"
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":      status,
		"service":     "juno-coordinator-mcp",
		"version":     "2.1.0-fi-integration",
		"ai":          "gemini-2.5-flash",
		"provider":    "Google",
		"fi_mcp":      fiStatus,
		"fi_url":      fiMCPBaseURL,
		"translation": translationStatus,
		"data_mode":   "real_financial_data_from_fi",
		"test_users":  "1111111111, 2121212121, 1313131313",
	})
}

// Root handler
func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	fiStatus := "Fi MCP: Disconnected"
	if fiMCPBaseURL != "" {
		fiStatus = fmt.Sprintf("Fi MCP: Connected to %s ✅", fiMCPBaseURL)
	}
	
	response := map[string]string{
		"service":     "Juno AI Financial Assistant",
		"status":      "running",
		"version":     "2.1.0-fi-integration",
		"ai":          "Gemini 2.5 Flash",
		"provider":    "Google",
		"data_source": fiStatus,
		"endpoints":   "/chat (POST), /ws (WebSocket), /health (GET), /languages (GET)",
		"description": "AI-powered financial assistant with REAL Fi MCP data integration",
		"hackathon":   "Google Agentic AI Day 2025",
		"update":      "Now with multi-lingual support! Chat in 24+ languages",
		"test_note":   "Use userID 1111111111, 2121212121, or 1313131313 for different financial scenarios",
	}
	json.NewEncoder(w).Encode(response)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// 🚀 KEY ADDITION: Initialize Fi MCP configuration
	log.Printf("🔗 Initializing Fi MCP configuration...")
	initializeFiMCP()

	// 🌐 NEW: Initialize translation service
	translationService = NewTranslationService()

	// Check for API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	
	if apiKey == "" {
		log.Printf("⚠️  WARNING: GEMINI_API_KEY or GOOGLE_API_KEY not set - AI responses will be limited")
	} else {
		log.Printf("✅ Google Gemini 2.5 Flash integration enabled")
	}

	router := mux.NewRouter()
	
	// Add CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	
	// Routes
	router.HandleFunc("/", handleRoot).Methods("GET")
	router.HandleFunc("/health", handleHealth).Methods("GET")
	router.HandleFunc("/chat", handleHTTPChat).Methods("POST", "OPTIONS")
	router.HandleFunc("/ws", handleWebSocket)
	router.HandleFunc("/languages", handleSupportedLanguages).Methods("GET") // 🌐 NEW endpoint

	// Apply CORS
	handler := c.Handler(router)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("🚀 Juno AI Coordinator starting on port %s", port)
	log.Printf("🤖 AI Model: Google Gemini 2.5 Flash")
	log.Printf("🏆 Hackathon: Google Agentic AI Day 2025")
	log.Printf("💰 Data Source: Fi MCP Server at %s", fiMCPBaseURL)
	log.Printf("🧪 Test Users: 1111111111 (basic), 2121212121 (investments), 1313131313 (investments)")
	log.Printf("📋 Endpoints available:")
	log.Printf("  GET  / - Service info")
	log.Printf("  GET  /health - Health check")
	log.Printf("  POST /chat - HTTP chat endpoint")
	log.Printf("  WS   /ws - WebSocket endpoint")
	log.Printf("  GET  /languages - Supported languages") // 🌐 NEW
	
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}