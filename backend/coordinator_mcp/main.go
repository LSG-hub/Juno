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
	} `json:"result"`
	ID string `json:"id"`
}

// Fi MCP Server configuration
var fiMCPBaseURL string

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Initialize Fi MCP configuration
func initializeFiMCP() {
	fiMCPURL := os.Getenv("FI_MCP_URL")
	if fiMCPURL == "" {
		fiMCPURL = "https://fi-mcp-server-amhclo2grq-uc.a.run.app"
	}
	fiMCPBaseURL = fiMCPURL
	log.Printf("✅ Fi MCP Server configured at: %s", fiMCPBaseURL)
}

// Call Fi MCP Server tool via HTTP
func callFiMCPTool(toolName string, userID string) (string, error) {
	// Create MCP tool call request
	mcpRequest := MCPToolCallRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}{
			Name: toolName,
			Arguments: map[string]interface{}{
				"phone_number": userID, // Fi MCP uses phone_number parameter
			},
		},
		ID: fmt.Sprintf("req_%d", time.Now().UnixNano()),
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(mcpRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP request: %v", err)
	}

	// Make HTTP request to Fi MCP server
	url := fmt.Sprintf("%s/mcp/", fiMCPBaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	log.Printf("📞 Calling Fi MCP tool '%s' for user '%s' at %s", toolName, userID, url)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Fi MCP: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Fi MCP response: %v", err)
	}

	if resp.StatusCode != 200 {
		log.Printf("❌ Fi MCP error: %d - %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("Fi MCP returned status %d", resp.StatusCode)
	}

	// Parse MCP response
	var mcpResponse MCPToolCallResponse
	if err := json.Unmarshal(body, &mcpResponse); err != nil {
		// If it's not a proper MCP response, return raw body
		log.Printf("⚠️ Non-MCP response from Fi server, returning raw content")
		return string(body), nil
	}

	// Extract content from MCP response
	if len(mcpResponse.Result.Content) > 0 {
		content := mcpResponse.Result.Content[0].Text
		log.Printf("✅ Fi MCP response received for user '%s': %d characters", userID, len(content))
		return content, nil
	}

	return "", fmt.Errorf("empty response from Fi MCP")
}

// Enhanced query processing with REAL financial data fetching
func processQuery(query, userID, firebaseUID string) string {
	lowerQuery := strings.ToLower(query)
	
	var enhancedQuery string
	var fiData string
	
	// 🚀 KEY CHANGE: Actually fetch real financial data for money/balance queries
	if strings.Contains(lowerQuery, "money") || strings.Contains(lowerQuery, "balance") || 
	   strings.Contains(lowerQuery, "account") || strings.Contains(lowerQuery, "net worth") ||
	   strings.Contains(lowerQuery, "financial status") || strings.Contains(lowerQuery, "how much") ||
	   strings.Contains(lowerQuery, "wealth") || strings.Contains(lowerQuery, "assets") {
		
		log.Printf("💰 Detected financial data query - fetching real data from Fi MCP")
		
		// Call Fi MCP to get actual financial data
		netWorthData, err := callFiMCPTool("fetch_net_worth", userID)
		if err != nil {
			log.Printf("⚠️ Failed to fetch net worth: %v", err)
			fiData = "I'm having trouble accessing your Fi account data right now. Please make sure you're logged in to Fi. "
		} else {
			fiData = fmt.Sprintf("📊 REAL FINANCIAL DATA FROM FI:\n%s\n\n", netWorthData)
		}
		
		enhancedQuery = fmt.Sprintf("💰 REAL FINANCIAL DATA QUERY: %s\n\n%sThe user asked: '%s'\n\nPlease analyze this real financial data from Fi and provide personalized insights and advice based on their actual account balances and net worth.", 
			query, fiData, query)
	
	// 🚀 KEY CHANGE: Fetch real transaction data for spending queries  
	} else if strings.Contains(lowerQuery, "spend") || strings.Contains(lowerQuery, "transaction") ||
	         strings.Contains(lowerQuery, "expense") || strings.Contains(lowerQuery, "budget") ||
	         strings.Contains(lowerQuery, "mutual fund") || strings.Contains(lowerQuery, "investment") {
		
		log.Printf("💳 Detected spending/investment query - fetching transaction data from Fi MCP")
		
		// Try to get mutual fund transactions
		transactionData, err := callFiMCPTool("fetch_mf_transactions", userID)
		if err != nil {
			log.Printf("⚠️ Failed to fetch transactions: %v", err)
			// Try bank transactions as fallback
			bankTxnData, bankErr := callFiMCPTool("fetch_bank_transactions", userID)
			if bankErr != nil {
				fiData = "I'm having trouble accessing your transaction data right now. Please make sure you're logged in to Fi. "
			} else {
				fiData = fmt.Sprintf("💳 REAL BANK TRANSACTION DATA FROM FI:\n%s\n\n", bankTxnData)
			}
		} else {
			fiData = fmt.Sprintf("📈 REAL MUTUAL FUND TRANSACTION DATA FROM FI:\n%s\n\n", transactionData)
		}
		
		enhancedQuery = fmt.Sprintf("💳 REAL SPENDING/INVESTMENT DATA QUERY: %s\n\n%sThe user asked: '%s'\n\nPlease analyze this real transaction data from Fi and provide insights about their spending patterns and investment behavior.", 
			query, fiData, query)
	
	// 🚀 KEY CHANGE: Fetch real credit data for credit-related queries
	} else if strings.Contains(lowerQuery, "credit") || strings.Contains(lowerQuery, "score") ||
	         strings.Contains(lowerQuery, "loan") || strings.Contains(lowerQuery, "debt") ||
	         strings.Contains(lowerQuery, "emi") {
		
		log.Printf("🏦 Detected credit query - fetching credit data from Fi MCP")
		
		creditData, err := callFiMCPTool("fetch_credit_report", userID)
		if err != nil {
			log.Printf("⚠️ Failed to fetch credit data: %v", err)
			fiData = "I'm having trouble accessing your credit data right now. Please make sure your credit report is linked to Fi. "
		} else {
			fiData = fmt.Sprintf("🏦 REAL CREDIT REPORT DATA FROM FI:\n%s\n\n", creditData)
		}
		
		enhancedQuery = fmt.Sprintf("🏦 REAL CREDIT DATA QUERY: %s\n\n%sThe user asked: '%s'\n\nPlease analyze this real credit data from Fi and provide advice on credit management and loan optimization.", 
			query, fiData, query)
	
	// 🚀 KEY CHANGE: Fetch EPF data for retirement-related queries
	} else if strings.Contains(lowerQuery, "epf") || strings.Contains(lowerQuery, "provident fund") ||
	         strings.Contains(lowerQuery, "retirement") || strings.Contains(lowerQuery, "pf") {
		
		log.Printf("🏛️ Detected EPF query - fetching EPF data from Fi MCP")
		
		epfData, err := callFiMCPTool("fetch_epf_details", userID)
		if err != nil {
			log.Printf("⚠️ Failed to fetch EPF data: %v", err)
			fiData = "I'm having trouble accessing your EPF data right now. Please make sure your EPF account is linked to Fi. "
		} else {
			fiData = fmt.Sprintf("🏛️ REAL EPF DATA FROM FI:\n%s\n\n", epfData)
		}
		
		enhancedQuery = fmt.Sprintf("🏛️ REAL EPF DATA QUERY: %s\n\n%sThe user asked: '%s'\n\nPlease analyze this real EPF data from Fi and provide retirement planning advice.", 
			query, fiData, query)
	
	// Greeting detection - no data needed
	} else if strings.Contains(lowerQuery, "hi") || strings.Contains(lowerQuery, "hello") || 
	   strings.Contains(lowerQuery, "hey") || strings.Contains(lowerQuery, "good morning") ||
	   strings.Contains(lowerQuery, "good evening") {
		enhancedQuery = fmt.Sprintf("👋 USER GREETING: %s\n\nPlease respond with a warm, friendly greeting and briefly introduce yourself as Juno, their AI financial assistant powered by Google Gemini. Mention that you can help with budgeting, investments, savings, credit management, and that you can access their real-time financial data from Fi to provide personalized advice.", query)
	
	// General financial query - try to get basic financial overview for context
	} else {
		log.Printf("🤔 General query - attempting to fetch basic financial context")
		
		// Try to get net worth for context (but don't fail if it doesn't work)
		netWorthData, err := callFiMCPTool("fetch_net_worth", userID)
		if err == nil {
			fiData = fmt.Sprintf("📊 USER'S FINANCIAL CONTEXT FROM FI:\n%s\n\n", netWorthData)
		}
		
		enhancedQuery = fmt.Sprintf("🤔 GENERAL FINANCIAL QUERY: %s\n\n%sUser %s is asking for financial guidance. Provide helpful, personalized advice based on their context (if available) and ask clarifying questions to better understand their specific needs.", 
			query, fiData, userID)
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

🏦 CORE FINANCIAL SERVICES:
- Personal budgeting and expense tracking using real transaction data
- Investment portfolio analysis using actual holdings and transaction history
- Retirement planning using real EPF balances and contribution history
- Debt management using actual credit reports and loan details
- Tax planning and optimization based on real investment data
- Insurance and risk assessment using comprehensive financial profiles
- Emergency fund planning using actual account balances
- Financial goal setting using real net worth and income data

💡 YOUR PERSONALITY:
- Friendly, warm, and conversational (like talking to a trusted friend)
- Professional yet accessible and empathetic
- Data-driven but focuses on practical, actionable advice
- Proactive in offering specific, personalized suggestions
- Clear and concise, avoids jargon, explains complex concepts simply

👤 CURRENT USER CONTEXT:
- User ID: %s
- Indian market focus (₹ currency, Indian financial products)
- Connected to Fi Money platform with real-time financial data access
- Age group: Likely 25-40 years (typical Fi user demographic)

🔗 REAL DATA INTEGRATION CAPABILITIES:
- Access to user's ACTUAL bank account balances from connected accounts
- Real mutual fund investment holdings and transaction history
- Actual EPF account balances and contribution records
- Live credit reports with real credit scores and loan details
- Comprehensive net worth calculations from real account data
- Spending pattern analysis from actual transaction histories

⭐ CRITICAL INSTRUCTIONS FOR REAL DATA:
1. When you receive real financial data, analyze it thoroughly and provide SPECIFIC advice based on actual numbers
2. Reference exact amounts, dates, and percentages from their real data
3. Don't give generic advice - use their actual financial situation to give tailored recommendations
4. Point out specific opportunities or risks you see in their real data
5. If data shows concerning patterns (low savings, high debt, etc.), address them constructively
6. Use their real account names, investment schemes, and actual transaction patterns in your advice

💰 RESPONSE STYLE:
- Start with their actual situation summary when you have real data
- Use specific numbers from their accounts (₹ amounts, percentages, dates)
- Give actionable next steps they can take immediately
- Explain WHY your recommendations make sense for their specific situation
- Be encouraging and supportive while being honest about financial realities

Remember: You have access to their REAL financial data, so make your advice as personalized and specific as possible!`, userID)

	// Prepare the request
	requestBody := GeminiRequest{
		SystemInstruction: &GeminiContent{
			Parts: []GeminiPart{{Text: systemPrompt}},
		},
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
			MaxOutputTokens: 1000,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	// Create request to Gemini API
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		log.Printf("❌ Gemini API error: %d - %s", resp.StatusCode, string(body))
		return "I'm experiencing some technical difficulties. Please try again in a moment.", nil
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

// Health check handler
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
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    status,
		"service":   "juno-coordinator-mcp",
		"version":   "2.1.0-fi-integration",
		"ai":        "gemini-2.5-flash",
		"provider":  "Google",
		"fi_mcp":    fiStatus,
		"fi_url":    fiMCPBaseURL,
		"data_mode": "real_financial_data_from_fi",
		"test_users": "1111111111, 2121212121, 1313131313",
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
		"endpoints":   "/chat (POST), /ws (WebSocket), /health (GET)",
		"description": "AI-powered financial assistant with REAL Fi MCP data integration",
		"hackathon":   "Google Agentic AI Day 2025",
		"update":      "Now fetches real financial data from Fi MCP Server via HTTP",
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
	
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}