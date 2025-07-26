package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ContextAgent struct {
	mcpServer      *server.MCPServer
	fiMCPURL       string
	geminiAPIKey   string
	firebaseAPIKey string
	projectID      string
}

type UserContext struct {
	UserID           string                 `json:"user_id"`
	LastActivity     time.Time              `json:"last_activity"`
	Location         string                 `json:"location,omitempty"`
	RecentEvents     []string               `json:"recent_events,omitempty"`
	SpendingPatterns map[string]interface{} `json:"spending_patterns,omitempty"`
	Preferences      map[string]interface{} `json:"preferences,omitempty"`
}

// Gemini Embedding API structures
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

// Enhanced message structure with embeddings (extends existing ChatMessage)
type EnhancedMessage struct {
	// Existing ChatMessage fields
	ID        string    `firestore:"id" json:"id"`
	Text      string    `firestore:"text" json:"text"`
	IsUser    bool      `firestore:"isUser" json:"isUser"`
	Timestamp time.Time `firestore:"timestamp" json:"timestamp"`
	Status    string    `firestore:"status" json:"status"`
	Metadata  map[string]interface{} `firestore:"metadata,omitempty" json:"metadata,omitempty"`
	
	// New embedding fields
	Embedding            []float64 `firestore:"embedding,omitempty" json:"embedding,omitempty"`
	EmbeddingGeneratedAt *time.Time `firestore:"embedding_generated_at,omitempty" json:"embedding_generated_at,omitempty"`
	EmbeddingModel       string    `firestore:"embedding_model,omitempty" json:"embedding_model,omitempty"`
}

func NewContextAgent() *ContextAgent {
	ca := &ContextAgent{
		fiMCPURL:       getEnvWithDefault("FI_MCP_URL", "http://localhost:8090"),
		geminiAPIKey:   os.Getenv("GEMINI_API_KEY"),
		firebaseAPIKey: getEnvWithDefault("FIREBASE_API_KEY", "AIzaSyCbFfZYvqbkeZlcK_Padg9hKnO7Xqbl1NI"),
		projectID:      getEnvWithDefault("GOOGLE_CLOUD_PROJECT", "juno-financial-assistant"),
	}
	
	log.Printf("Context Agent initialized with Firebase project: %s", ca.projectID)
	return ca
}

// Firebase REST API helper functions

// updateFirebaseDocument updates a document using Firebase REST API
func (ca *ContextAgent) updateFirebaseDocument(docPath string, data map[string]interface{}) error {
	url := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents/%s?key=%s", ca.projectID, docPath, ca.firebaseAPIKey)
	
	// Firebase REST API requires field values to be structured
	fields := make(map[string]interface{})
	for key, value := range data {
		fields[key] = ca.formatFirebaseValue(value)
	}
	
	updateDoc := map[string]interface{}{
		"fields": fields,
	}
	
	jsonData, err := json.Marshal(updateDoc)
	if err != nil {
		return fmt.Errorf("failed to marshal update data: %w", err)
	}
	
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Firebase update returned status %d", resp.StatusCode)
	}
	
	return nil
}

// queryFirebaseCollection queries a collection using Firebase REST API
func (ca *ContextAgent) queryFirebaseCollection(collectionPath string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents/%s?key=%s", ca.projectID, collectionPath, ca.firebaseAPIKey)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create query request: %w", err)
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Firebase query returned status %d", resp.StatusCode)
	}
	
	var result struct {
		Documents []struct {
			Fields map[string]interface{} `json:"fields"`
		} `json:"documents"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}
	
	var messages []map[string]interface{}
	for _, doc := range result.Documents {
		messageData := make(map[string]interface{})
		for key, field := range doc.Fields {
			messageData[key] = ca.parseFirebaseValue(field)
		}
		messages = append(messages, messageData)
	}
	
	return messages, nil
}

// formatFirebaseValue converts Go values to Firebase REST API format
func (ca *ContextAgent) formatFirebaseValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return map[string]interface{}{"stringValue": v}
	case int, int64:
		return map[string]interface{}{"integerValue": fmt.Sprintf("%d", v)}
	case float64:
		return map[string]interface{}{"doubleValue": v}
	case bool:
		return map[string]interface{}{"booleanValue": v}
	case []float64:
		arrayValues := make([]interface{}, len(v))
		for i, val := range v {
			arrayValues[i] = map[string]interface{}{"doubleValue": val}
		}
		return map[string]interface{}{
			"arrayValue": map[string]interface{}{
				"values": arrayValues,
			},
		}
	default:
		return map[string]interface{}{"stringValue": fmt.Sprintf("%v", v)}
	}
}

// parseFirebaseValue converts Firebase REST API values to Go values
func (ca *ContextAgent) parseFirebaseValue(field interface{}) interface{} {
	fieldMap, ok := field.(map[string]interface{})
	if !ok {
		return field
	}
	
	if stringVal, ok := fieldMap["stringValue"]; ok {
		return stringVal
	}
	if intVal, ok := fieldMap["integerValue"]; ok {
		if intStr, ok := intVal.(string); ok {
			if val, err := strconv.ParseInt(intStr, 10, 64); err == nil {
				return val
			}
		}
		return intVal
	}
	if doubleVal, ok := fieldMap["doubleValue"]; ok {
		return doubleVal
	}
	if boolVal, ok := fieldMap["booleanValue"]; ok {
		return boolVal
	}
	if arrayVal, ok := fieldMap["arrayValue"]; ok {
		if arrayMap, ok := arrayVal.(map[string]interface{}); ok {
			if values, ok := arrayMap["values"].([]interface{}); ok {
				result := make([]float64, len(values))
				for i, val := range values {
					if valMap, ok := val.(map[string]interface{}); ok {
						if doubleVal, ok := valMap["doubleValue"].(float64); ok {
							result[i] = doubleVal
						}
					}
				}
				return result
			}
		}
	}
	
	return field
}

// parseMessageData converts Firebase message data to EnhancedMessage struct
func (ca *ContextAgent) parseMessageData(data map[string]interface{}, message *EnhancedMessage) error {
	if id, ok := data["id"].(string); ok {
		message.ID = id
	}
	
	if text, ok := data["text"].(string); ok {
		message.Text = text
	}
	
	if isUser, ok := data["isUser"].(bool); ok {
		message.IsUser = isUser
	}
	
	if status, ok := data["status"].(string); ok {
		message.Status = status
	}
	
	if timestamp, ok := data["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			message.Timestamp = t
		}
	}
	
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		message.Metadata = metadata
	}
	
	if embedding, ok := data["embedding"].([]float64); ok {
		message.Embedding = embedding
	}
	
	if embModel, ok := data["embedding_model"].(string); ok {
		message.EmbeddingModel = embModel
	}
	
	if embGenAt, ok := data["embedding_generated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, embGenAt); err == nil {
			message.EmbeddingGeneratedAt = &t
		}
	}
	
	return nil
}

func (ca *ContextAgent) setupMCPServer() {
	ca.mcpServer = server.NewMCPServer(
		"context-agent-mcp",
		"0.1.0",
		server.WithInstructions(`Juno Context Agent MCP Server - The intelligent memory and context hub that powers personalized conversations and decisions.

CORE RESPONSIBILITIES:
• **Conversation Memory**: Store and retrieve all user interactions with intelligent embedding-based search
• **Context Intelligence**: Analyze user behavior patterns, life events, and temporal context
• **RAG System**: Automatically enhance conversations with relevant historical context
• **Environmental Awareness**: Provide situational context for better decision-making

KEY CAPABILITIES:
1. **Automatic RAG Processing**:
   - Generate embeddings for all messages (user + assistant)
   - Store conversations with metadata in Firestore
   - Search similar conversations using cosine similarity (0.7 threshold)
   - Return top 5 most relevant contexts automatically

2. **Behavioral Analysis**:
   - Spending pattern recognition and insights
   - Life event detection from data patterns
   - Temporal context analysis (time-sensitive decisions)
   - User preference learning and adaptation

3. **Data Architecture**:
   - Per-user isolation: users/{firebaseUID}/chats/{fiUserID}/messages/
   - Multi-user support with complete data separation
   - Production-ready Firestore vector storage
   - Research-optimized parameters for maximum relevance

INTEGRATION GUIDELINES:
• Called automatically by Coordinator for every user interaction
• Provides enriched context without manual user intervention
• Maintains conversation continuity across user sessions
• Supports seamless switching between different Fi test users

This agent operates transparently in the background, making Juno contextually aware and conversationally intelligent.`),
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	// Add context analysis tools
	ca.mcpServer.AddTool(
		mcp.NewTool("analyze_user_context",
			mcp.WithDescription("Analyze user's current context for financial decision making"),
			mcp.WithString("user_id",
				mcp.Description("User ID for context analysis"),
			),
		),
		ca.handleAnalyzeContext,
	)

	ca.mcpServer.AddTool(
		mcp.NewTool("get_spending_patterns",
			mcp.WithDescription("Get user's spending patterns and behavioral insights"),
			mcp.WithString("user_id",
				mcp.Description("User ID for spending analysis"),
			),
		),
		ca.handleGetSpendingPatterns,
	)

	ca.mcpServer.AddTool(
		mcp.NewTool("detect_life_events",
			mcp.WithDescription("Detect significant life events from user data patterns"),
			mcp.WithString("user_id",
				mcp.Description("User ID for life event detection"),
			),
		),
		ca.handleDetectLifeEvents,
	)

	ca.mcpServer.AddTool(
		mcp.NewTool("get_temporal_context",
			mcp.WithDescription("Get time-based context for financial decisions"),
		),
		ca.handleGetTemporalContext,
	)

	// RAG-enabled tools for context search and embeddings
	ca.mcpServer.AddTool(
		mcp.NewTool("generate_text_embedding",
			mcp.WithDescription("Generate Gemini embeddings for text content"),
			mcp.WithString("text",
				mcp.Description("Text content to generate embeddings for"),
				mcp.Required(),
			),
			mcp.WithString("task_type",
				mcp.Description("Embedding task type: RETRIEVAL_DOCUMENT or RETRIEVAL_QUERY"),
				mcp.Required(),
			),
		),
		ca.handleGenerateTextEmbedding,
	)

	// Unified context processing: automatically called by Coordinator for every message
	ca.mcpServer.AddTool(
		mcp.NewTool("process_message_context",
			mcp.WithDescription("Process message: store with embeddings + retrieve relevant context automatically"),
			mcp.WithString("firebase_uid",
				mcp.Description("Firebase user ID for data isolation"),
				mcp.Required(),
			),
			mcp.WithString("fi_user_id", 
				mcp.Description("Fi user ID (1010101010-9999999999)"),
				mcp.Required(),
			),
			mcp.WithString("message_id",
				mcp.Description("Unique message identifier"),
				mcp.Required(),
			),
			mcp.WithString("text",
				mcp.Description("Message text content"),
				mcp.Required(),
			),
			mcp.WithBoolean("is_user",
				mcp.Description("Whether message is from user (true) or assistant (false)"),
				mcp.Required(),
			),
			mcp.WithString("timestamp",
				mcp.Description("Message timestamp in RFC3339 format"),
				mcp.Required(),
			),
			mcp.WithString("status",
				mcp.Description("Message status (optional)"),
			),
		),
		ca.handleProcessMessageContext,
	)

	ca.mcpServer.AddTool(
		mcp.NewTool("search_similar_conversations",
			mcp.WithDescription("Search for similar conversations using embedding similarity"),
			mcp.WithString("user_id",
				mcp.Description("User ID for context search"),
				mcp.Required(),
			),
			mcp.WithString("query",
				mcp.Description("Query text to search for similar conversations"),
				mcp.Required(),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of similar conversations to return (default: 5)"),
			),
			mcp.WithNumber("similarity_threshold",
				mcp.Description("Minimum similarity threshold (default: 0.7)"),
			),
		),
		ca.handleSearchSimilarConversations,
	)

	// Load chat history for mobile app UI (RAG works automatically regardless)
	ca.mcpServer.AddTool(
		mcp.NewTool("load_chat_history",
			mcp.WithDescription("Load complete chat history for a specific Fi user from Context Agent storage"),
			mcp.WithString("firebase_uid",
				mcp.Description("Firebase user ID for data isolation"),
				mcp.Required(),
			),
			mcp.WithString("fi_user_id",
				mcp.Description("Fi user ID (1010101010-9999999999)"),
				mcp.Required(),
			),
		),
		ca.handleLoadChatHistory,
	)

	// General conversation tool for non-financial queries
	ca.mcpServer.AddTool(
		mcp.NewTool("general_conversation",
			mcp.WithDescription("Handle general conversation using Gemini with context from RAG"),
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
		ca.handleGeneralConversation,
	)

	// RAG with hybrid search: Recent messages + Similarity search
	ca.mcpServer.AddTool(
		mcp.NewTool("search_similar_conversations_hybrid",
			mcp.WithDescription("Hybrid RAG search: recent conversation window + similarity search"),
			mcp.WithString("firebase_uid",
				mcp.Description("Firebase user ID for data isolation"),
				mcp.Required(),
			),
			mcp.WithString("fi_user_id",
				mcp.Description("Fi user ID (1010101010-9999999999)"),
				mcp.Required(),
			),
			mcp.WithString("query",
				mcp.Description("Query text to search for similar conversations"),
				mcp.Required(),
			),
		),
		ca.handleSearchSimilarConversationsHybrid,
	)
}

func (ca *ContextAgent) handleAnalyzeContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	userID, ok := arguments["user_id"].(string)
	if !ok {
		userID = "default_user"
	}

	// For MVP, return mock context data
	userContext := UserContext{
		UserID:       userID,
		LastActivity: time.Now(),
		Location:     "Mumbai, India",
		RecentEvents: []string{
			"salary_credited_yesterday",
			"monthly_rent_payment",
			"grocery_shopping_increased",
		},
		SpendingPatterns: map[string]interface{}{
			"average_monthly_spend": 25000,
			"top_categories": []string{
				"groceries",
				"transportation",
				"entertainment",
			},
			"spending_trend": "stable",
		},
		Preferences: map[string]interface{}{
			"preferred_language": "english",
			"risk_tolerance":     "moderate",
			"savings_goal":       "emergency_fund",
		},
	}

	jsonData, err := json.Marshal(userContext)
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

func (ca *ContextAgent) handleGetSpendingPatterns(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Mock spending patterns for MVP
	patterns := map[string]interface{}{
		"monthly_analysis": map[string]interface{}{
			"current_month_spend": 18500,
			"average_spend":       22000,
			"trend":              "decreasing",
			"variance":           15.9,
		},
		"category_breakdown": map[string]interface{}{
			"groceries":      3200,
			"entertainment":  2100,
			"transportation": 1500,
			"utilities":      1200,
			"dining":         1800,
			"shopping":       2700,
			"other":          6000,
		},
		"behavioral_insights": []string{
			"Spending less than usual this month",
			"Entertainment spending 20% above average",
			"Grocery spending optimized well",
			"Good control over discretionary expenses",
		},
		"recommendations": []string{
			"Current spending pace is sustainable",
			"Consider investing surplus this month",
			"Entertainment budget could be reviewed",
		},
	}

	jsonData, err := json.Marshal(patterns)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Failed to marshal spending patterns",
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

func (ca *ContextAgent) handleDetectLifeEvents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Mock life event detection for MVP
	lifeEvents := map[string]interface{}{
		"detected_events": []map[string]interface{}{
			{
				"event_type":   "salary_increment",
				"confidence":   0.85,
				"detected_on":  time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
				"description":  "Increased monthly salary deposits detected",
				"impact":       "positive",
				"suggestions": []string{
					"Consider increasing SIP amounts",
					"Review and update financial goals",
					"Build larger emergency fund",
				},
			},
			{
				"event_type":  "new_subscription",
				"confidence":  0.72,
				"detected_on": time.Now().AddDate(0, 0, -15).Format("2006-01-02"),
				"description": "New recurring payment pattern detected",
				"impact":      "neutral",
				"suggestions": []string{
					"Review subscription value",
					"Track if it fits monthly budget",
				},
			},
		},
		"monitoring": []string{
			"Watching for major purchase patterns",
			"Monitoring income stability",
			"Tracking spending behavior changes",
		},
	}

	jsonData, err := json.Marshal(lifeEvents)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Failed to marshal life events",
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

func (ca *ContextAgent) handleGetTemporalContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	now := time.Now()
	
	temporalContext := map[string]interface{}{
		"current_time": now.Format(time.RFC3339),
		"time_of_day":  getTimeOfDay(now),
		"day_of_week":  now.Weekday().String(),
		"month":        now.Month().String(),
		"financial_calendar": map[string]interface{}{
			"days_since_salary": getDaysSinceSalary(now),
			"days_to_month_end": getDaysToMonthEnd(now),
			"is_festival_season": isFestivalSeason(now),
			"is_tax_season":     isTaxSeason(now),
		},
		"spending_context": map[string]interface{}{
			"typical_spending_time": getTypicalSpendingTime(now),
			"budget_availability":   "high", // Based on recent salary credit
			"recommended_caution":   "low",  // Early in month, post-salary
		},
		"market_context": map[string]interface{}{
			"market_hours": isMarketHours(now),
			"trading_day":  isTradingDay(now),
		},
	}

	jsonData, err := json.Marshal(temporalContext)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Failed to marshal temporal context",
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

// RAG Tool Handlers

// generateTextEmbedding calls Gemini Embedding API to generate embeddings for text
func (ca *ContextAgent) generateTextEmbedding(text string, taskType string) ([]float64, error) {
	if ca.geminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not configured")
	}

	// Prepare request
	request := GeminiEmbeddingRequest{
		Model: "models/text-embedding-004",
		Content: GeminiEmbeddingContent{
			Parts: []GeminiEmbeddingPart{
				{Text: text},
			},
		},
		TaskType: taskType,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	// Make API call
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/text-embedding-004:embedContent?key=%s", ca.geminiAPIKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini embedding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini embedding API returned status %d", resp.StatusCode)
	}

	var embeddingResp GeminiEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	return embeddingResp.Embedding.Values, nil
}

// calculateCosineSimilarity computes cosine similarity between two vectors
func calculateCosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (ca *ContextAgent) handleGenerateTextEmbedding(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	
	text, ok := arguments["text"].(string)
	if !ok || text == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: text parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	taskType, ok := arguments["task_type"].(string)
	if !ok || taskType == "" {
		taskType = "RETRIEVAL_DOCUMENT"
	}

	// Validate task type
	if taskType != "RETRIEVAL_DOCUMENT" && taskType != "RETRIEVAL_QUERY" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: task_type must be RETRIEVAL_DOCUMENT or RETRIEVAL_QUERY",
				},
			},
			IsError: true,
		}, nil
	}

	embedding, err := ca.generateTextEmbedding(text, taskType)
	if err != nil {
		log.Printf("Error generating embedding: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error generating embedding: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	result := map[string]interface{}{
		"embedding": embedding,
		"dimensions": len(embedding),
		"text": text,
		"task_type": taskType,
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error marshalling embedding result",
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonResult),
			},
		},
	}, nil
}

// Unified message processing: store with embeddings + retrieve relevant context
func (ca *ContextAgent) handleProcessMessageContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	text, ok := arguments["text"].(string)
	if !ok || text == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: text parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	isUser, ok := arguments["is_user"].(bool)
	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: is_user parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	timestamp, ok := arguments["timestamp"].(string)
	if !ok || timestamp == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: timestamp parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	status := "sent" // default
	if statusVal, ok := arguments["status"].(string); ok && statusVal != "" {
		status = statusVal
	}

	// STEP 1: Generate embedding for the message
	embedding, err := ca.generateTextEmbedding(text, "RETRIEVAL_DOCUMENT")
	if err != nil {
		log.Printf("Error generating embedding: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error generating embedding: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// STEP 2: Store complete message with embedding in Firestore
	if ca.firebaseAPIKey == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Firebase API key not configured",
				},
			},
			IsError: true,
		}, nil
	}

	docPath := fmt.Sprintf("users/%s/chats/%s/messages/%s", firebaseUID, fiUserID, messageID)
	now := time.Now()
	
	// Store complete message data with embedding
	messageData := map[string]interface{}{
		"id":        messageID,
		"text":      text,
		"isUser":    isUser,
		"timestamp": timestamp,
		"status":    status,
		"embedding": embedding,
		"embedding_generated_at": now.Format(time.RFC3339),
		"embedding_model": "text-embedding-004",
	}
	
	err = ca.updateFirebaseDocument(docPath, messageData)
	if err != nil {
		log.Printf("Error storing message: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error storing message: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// STEP 3: Retrieve relevant context for user messages only (to provide context for Gemini's response)
	var relevantContext []map[string]interface{}
	if isUser {
		// Generate query embedding
		queryEmbedding, err := ca.generateTextEmbedding(text, "RETRIEVAL_QUERY")
		if err != nil {
			log.Printf("Warning: Could not generate query embedding for context retrieval: %v", err)
		} else {
			// Search for similar conversations
			collectionPath := fmt.Sprintf("users/%s/chats/%s/messages", firebaseUID, fiUserID)
			messages, err := ca.queryFirebaseCollection(collectionPath)
			if err != nil {
				log.Printf("Warning: Could not query for similar conversations: %v", err)
			} else {
				// Find similar messages (excluding the current one)
				similarityThreshold := 0.7
				limit := 5
				
				type SimilarMessage struct {
					Text       string  `json:"text"`
					IsUser     bool    `json:"is_user"`
					Timestamp  string  `json:"timestamp"`
					Similarity float64 `json:"similarity"`
				}
				
				var similarMessages []SimilarMessage
				for _, messageData := range messages {
					var msg EnhancedMessage
					if err := ca.parseMessageData(messageData, &msg); err != nil {
						continue
					}
					
					// Skip current message and messages without embeddings
					if msg.ID == messageID || len(msg.Embedding) == 0 {
						continue
					}
					
					similarity := calculateCosineSimilarity(queryEmbedding, msg.Embedding)
					if similarity >= similarityThreshold {
						similarMessages = append(similarMessages, SimilarMessage{
							Text:       msg.Text,
							IsUser:     msg.IsUser,
							Timestamp:  msg.Timestamp.Format(time.RFC3339),
							Similarity: similarity,
						})
					}
				}
				
				// Sort by similarity (highest first) and limit
				for i := 0; i < len(similarMessages)-1; i++ {
					for j := i + 1; j < len(similarMessages); j++ {
						if similarMessages[j].Similarity > similarMessages[i].Similarity {
							similarMessages[i], similarMessages[j] = similarMessages[j], similarMessages[i]
						}
					}
				}
				
				if len(similarMessages) > limit {
					similarMessages = similarMessages[:limit]
				}
				
				// Format context for Coordinator
				for _, sim := range similarMessages {
					relevantContext = append(relevantContext, map[string]interface{}{
						"text":       sim.Text,
						"is_user":    sim.IsUser,
						"timestamp":  sim.Timestamp,
						"similarity": sim.Similarity,
					})
				}
			}
		}
	}

	// Return result with storage confirmation and relevant context
	result := map[string]interface{}{
		"status": "processed",
		"stored": map[string]interface{}{
			"firebase_uid":         firebaseUID,
			"fi_user_id":          fiUserID,
			"message_id":          messageID,
			"embedding_dimensions": len(embedding),
			"stored_at":           now.Format(time.RFC3339),
			"firestore_path":      docPath,
		},
		"relevant_context": relevantContext,
		"context_count":   len(relevantContext),
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error marshalling processing result",
				},
			},
			IsError: true,
		}, nil
	}

	log.Printf("Processed message for user %s/%s: stored + found %d relevant contexts", firebaseUID, fiUserID, len(relevantContext))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonResult),
			},
		},
	}, nil
}


func (ca *ContextAgent) handleLoadChatHistory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	if ca.firebaseAPIKey == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Firebase API key not configured",
				},
			},
			IsError: true,
		}, nil
	}

	// Load all messages for this specific Fi user
	collectionPath := fmt.Sprintf("users/%s/chats/%s/messages", firebaseUID, fiUserID)
	messages, err := ca.queryFirebaseCollection(collectionPath)
	if err != nil {
		log.Printf("Error loading chat history for %s/%s: %v", firebaseUID, fiUserID, err)
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

	// Parse and sort messages by timestamp
	var parsedMessages []EnhancedMessage
	for _, messageData := range messages {
		var msg EnhancedMessage
		if err := ca.parseMessageData(messageData, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}
		parsedMessages = append(parsedMessages, msg)
	}

	// Sort messages by timestamp (oldest first)
	for i := 0; i < len(parsedMessages)-1; i++ {
		for j := i + 1; j < len(parsedMessages); j++ {
			if parsedMessages[i].Timestamp.After(parsedMessages[j].Timestamp) {
				parsedMessages[i], parsedMessages[j] = parsedMessages[j], parsedMessages[i]
			}
		}
	}

	// Format response for mobile app
	result := map[string]interface{}{
		"firebase_uid": firebaseUID,
		"fi_user_id":   fiUserID,
		"messages":     parsedMessages,
		"message_count": len(parsedMessages),
		"loaded_at":    time.Now().Format(time.RFC3339),
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error marshalling chat history",
				},
			},
			IsError: true,
		}, nil
	}

	log.Printf("Loaded %d messages for user %s/%s", len(parsedMessages), firebaseUID, fiUserID)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonResult),
			},
		},
	}, nil
}

func (ca *ContextAgent) handleSearchSimilarConversations(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	
	userID, ok := arguments["user_id"].(string)
	if !ok || userID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: user_id parameter is required",
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

	limit := 5
	if limitVal, ok := arguments["limit"].(float64); ok {
		limit = int(limitVal)
	}

	similarityThreshold := 0.7
	if thresholdVal, ok := arguments["similarity_threshold"].(float64); ok {
		similarityThreshold = thresholdVal
	}

	// Generate embedding for the query
	queryEmbedding, err := ca.generateTextEmbedding(query, "RETRIEVAL_QUERY")
	if err != nil {
		log.Printf("Error generating query embedding: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error generating query embedding: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Find similar conversations for this user from Firestore
	type SimilarityResult struct {
		Message    EnhancedMessage `json:"message"`
		Similarity float64         `json:"similarity"`
	}

	if ca.firebaseAPIKey == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Firebase API key not configured",
				},
			},
			IsError: true,
		}, nil
	}

	// For now, use userID as firebaseUID (should be properly passed in PHASE 3)
	firebaseUID := userID
	
	// Query all messages for this user that have embeddings using Firebase REST API
	collectionPath := fmt.Sprintf("users/%s/chats/%s/messages", firebaseUID, userID)
	messages, err := ca.queryFirebaseCollection(collectionPath)
	if err != nil {
		log.Printf("Error querying messages: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error querying messages: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	var results []SimilarityResult
	for _, messageData := range messages {
		// Parse message data
		var message EnhancedMessage
		if err := ca.parseMessageData(messageData, &message); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		// Skip if no embedding available
		if len(message.Embedding) == 0 {
			continue
		}

		similarity := calculateCosineSimilarity(queryEmbedding, message.Embedding)
		if similarity >= similarityThreshold {
			results = append(results, SimilarityResult{
				Message:    message,
				Similarity: similarity,
			})
		}
	}

	// Sort by similarity (highest first)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	searchResult := map[string]interface{}{
		"query": query,
		"firebase_uid": firebaseUID,
		"user_id": userID,
		"similar_messages": results,
		"total_found": len(results),
		"similarity_threshold": similarityThreshold,
		"search_timestamp": time.Now(),
	}

	jsonResult, err := json.Marshal(searchResult)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error marshalling search results",
				},
			},
			IsError: true,
		}, nil
	}

	log.Printf("Found %d similar conversations for user %s query: %s", len(results), userID, query)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonResult),
			},
		},
	}, nil
}

// Migration functions removed - user is clearing database for fresh start

func (ca *ContextAgent) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "context-agent-mcp",
		"version": "0.1.0",
	})
}

// Helper functions for temporal context
func getTimeOfDay(t time.Time) string {
	hour := t.Hour()
	switch {
	case hour < 6:
		return "early_morning"
	case hour < 12:
		return "morning"
	case hour < 17:
		return "afternoon"
	case hour < 21:
		return "evening"
	default:
		return "night"
	}
}

func getDaysSinceSalary(t time.Time) int {
	// Assume salary is credited on 1st of every month
	if t.Day() == 1 {
		return 0
	}
	return t.Day() - 1
}

func getDaysToMonthEnd(t time.Time) int {
	nextMonth := t.AddDate(0, 1, 0)
	lastDay := nextMonth.AddDate(0, 0, -nextMonth.Day())
	return lastDay.Day() - t.Day()
}

func isFestivalSeason(t time.Time) bool {
	month := t.Month()
	// October to December is festival season in India
	return month >= 10 || month <= 12
}

func isTaxSeason(t time.Time) bool {
	month := t.Month()
	// March to July is tax season in India
	return month >= 3 && month <= 7
}

func getTypicalSpendingTime(t time.Time) string {
	hour := t.Hour()
	day := t.Weekday()
	
	if day == time.Saturday || day == time.Sunday {
		return "weekend_spending"
	}
	
	switch {
	case hour >= 9 && hour <= 17:
		return "work_hours_spending"
	case hour >= 18 && hour <= 22:
		return "evening_spending"
	default:
		return "off_hours"
	}
}

func isMarketHours(t time.Time) bool {
	hour := t.Hour()
	day := t.Weekday()
	
	// Indian market hours: 9:15 AM to 3:30 PM, Monday to Friday
	return day >= time.Monday && day <= time.Friday && hour >= 9 && hour < 16
}

func isTradingDay(t time.Time) bool {
	day := t.Weekday()
	return day >= time.Monday && day <= time.Friday
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Hybrid RAG search: Recent conversation window + similarity search
func (ca *ContextAgent) handleSearchSimilarConversationsHybrid(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	if ca.firebaseAPIKey == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Firebase API key not configured",
				},
			},
			IsError: true,
		}, nil
	}

	// Fetch all messages from Firestore
	collectionPath := fmt.Sprintf("users/%s/chats/%s/messages", firebaseUID, fiUserID)
	messages, err := ca.queryFirebaseCollection(collectionPath)
	if err != nil {
		log.Printf("Warning: Failed to fetch messages: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "[]", // Return empty array if no messages
				},
			},
		}, nil
	}

	// Parse all messages
	var allMessages []EnhancedMessage
	for _, messageData := range messages {
		var msg EnhancedMessage
		if err := ca.parseMessageData(messageData, &msg); err != nil {
			continue
		}
		allMessages = append(allMessages, msg)
	}

	if len(allMessages) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "[]", // Return empty array
				},
			},
		}, nil
	}

	// Sort messages by timestamp (newest first)
	for i := 0; i < len(allMessages)-1; i++ {
		for j := 0; j < len(allMessages)-i-1; j++ {
			if allMessages[j].Timestamp.Before(allMessages[j+1].Timestamp) {
				allMessages[j], allMessages[j+1] = allMessages[j+1], allMessages[j]
			}
		}
	}

	var contextMessages []map[string]interface{}

	// PART 1: Always include the last 4 messages (recent conversation window)
	recentCount := 4
	if len(allMessages) < recentCount {
		recentCount = len(allMessages)
	}

	// Get recent messages and reverse them to be in chronological order (oldest first)
	var recentMessages []map[string]interface{}
	for i := 0; i < recentCount; i++ {
		msg := allMessages[i]
		recentMessages = append(recentMessages, map[string]interface{}{
			"text":       msg.Text,
			"is_user":    msg.IsUser,
			"timestamp":  msg.Timestamp.Format(time.RFC3339),
			"similarity": 1.0, // Mark recent messages as high relevance
			"source":     "recent",
		})
	}
	
	// Reverse to chronological order (oldest first) for natural conversation flow
	for i := len(recentMessages) - 1; i >= 0; i-- {
		contextMessages = append(contextMessages, recentMessages[i])
	}

	// PART 2: Generate query embedding for similarity search
	queryEmbedding, err := ca.generateTextEmbeddingV2(query, "RETRIEVAL_QUERY")
	if err != nil {
		log.Printf("Warning: Failed to generate query embedding, using recent messages only: %v", err)
		// Return just recent messages
		result := map[string]interface{}{
			"similar_messages": contextMessages,
			"total_found":     len(contextMessages),
			"search_type":     "recent_only",
		}

		jsonResult, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(jsonResult),
				},
			},
		}, nil
	}

	// PART 3: Find similar messages from older conversations (beyond recent window)
	var similarMessages []map[string]interface{}
	for i := recentCount; i < len(allMessages); i++ {
		msg := allMessages[i]
		if len(msg.Embedding) > 0 {
			similarity := calculateCosineSimilarity(queryEmbedding, msg.Embedding)
			// Use 0.4 threshold (40%) for better conversational continuity
			if similarity > 0.4 {
				similarMessages = append(similarMessages, map[string]interface{}{
					"text":       msg.Text,
					"is_user":    msg.IsUser,
					"timestamp":  msg.Timestamp.Format(time.RFC3339),
					"similarity": similarity,
					"source":     "similar",
				})
			}
		}
	}

	// Sort similar messages by similarity (highest first)
	if len(similarMessages) > 1 {
		for i := 0; i < len(similarMessages)-1; i++ {
			for j := 0; j < len(similarMessages)-i-1; j++ {
				sim1 := similarMessages[j]["similarity"].(float64)
				sim2 := similarMessages[j+1]["similarity"].(float64)
				if sim1 < sim2 {
					similarMessages[j], similarMessages[j+1] = similarMessages[j+1], similarMessages[j]
				}
			}
		}
	}

	// PART 4: Add relevant similar messages (only if they're highly relevant)
	// Priority: Recent conversation is most important, similar messages are secondary
	maxSimilar := 1  // Reduced to avoid confusion - prioritize recent conversation
	minSimilarityForInclusion := 0.6  // Higher threshold to only include very relevant similar messages
	
	var relevantSimilarMessages []map[string]interface{}
	for i := 0; i < len(similarMessages) && len(relevantSimilarMessages) < maxSimilar; i++ {
		if similarity, ok := similarMessages[i]["similarity"].(float64); ok && similarity >= minSimilarityForInclusion {
			// Mark as historical context to distinguish from recent conversation
			similarMessages[i]["source"] = "historical"
			relevantSimilarMessages = append(relevantSimilarMessages, similarMessages[i])
		}
	}
	
	// Add similar messages at the beginning (before recent conversation) if any
	if len(relevantSimilarMessages) > 0 {
		// Insert at the beginning
		var finalContextMessages []map[string]interface{}
		finalContextMessages = append(finalContextMessages, relevantSimilarMessages...)
		finalContextMessages = append(finalContextMessages, contextMessages...)
		contextMessages = finalContextMessages
	}

	// Return the hybrid result
	result := map[string]interface{}{
		"similar_messages": contextMessages,
		"total_found":     len(contextMessages),
		"recent_count":    recentCount,
		"similar_count":   len(relevantSimilarMessages),
		"search_type":     "hybrid",
		"threshold":       0.6, // Updated to match new threshold
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error marshalling hybrid search results",
				},
			},
			IsError: true,
		}, nil
	}

	log.Printf("Hybrid RAG: Found %d recent + %d similar messages for user %s/%s", recentCount, len(relevantSimilarMessages), firebaseUID, fiUserID)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonResult),
			},
		},
	}, nil
}

// General conversation handler with RAG context
func (ca *ContextAgent) handleGeneralConversation(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	if ca.geminiAPIKey == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Hello! I'm Juno, your helpful AI companion. I'm currently running in demo mode. How can I help you today?",
				},
			},
		}, nil
	}

	// STEP 1: Get relevant context using hybrid RAG search
	hybridResult, err := ca.handleSearchSimilarConversationsHybrid(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"firebase_uid": firebaseUID,
				"fi_user_id":   fiUserID,
				"query":        query,
			},
		},
	})
	if err != nil {
		log.Printf("Warning: Failed to get context: %v", err)
	}

	// Parse context from hybrid search
	var relevantContext []map[string]interface{}
	if hybridResult != nil && len(hybridResult.Content) > 0 {
		if textContent, ok := hybridResult.Content[0].(mcp.TextContent); ok {
			var contextData map[string]interface{}
			if err := json.Unmarshal([]byte(textContent.Text), &contextData); err == nil {
				if similarMsgs, ok := contextData["similar_messages"].([]interface{}); ok {
					for _, msg := range similarMsgs {
						if msgMap, ok := msg.(map[string]interface{}); ok {
							relevantContext = append(relevantContext, msgMap)
						}
					}
				}
			}
		}
	}

	// STEP 2: Build prompt with context
	var promptText string
	if len(relevantContext) > 0 {
		var contextText string
		for _, ctx := range relevantContext {
			if text, ok := ctx["text"].(string); ok {
				if isUser, ok := ctx["is_user"].(bool); ok {
					role := "Assistant"
					if isUser {
						role = "User"
					}
					contextText += fmt.Sprintf("- %s: %s\n", role, text)
				}
			}
		}

		promptText = fmt.Sprintf(`You are Juno, a warm, empathetic, and intelligent AI companion. You're designed to be a supportive friend who genuinely cares about the user's well-being across all aspects of their life.

## Your Core Personality:
- **Empathetic & Caring**: Always acknowledge emotions and provide emotional support when needed
- **Encouraging & Positive**: Help users feel motivated and optimistic about their goals
- **Intelligent & Helpful**: Provide thoughtful, practical advice across diverse topics
- **Conversational & Natural**: Chat like a close friend who remembers previous conversations
- **Balanced**: You're an all-around companion first, with financial expertise when relevant

## Your Capabilities:
- **Life Companion**: Relationships, mental health, career advice, learning, hobbies, travel, health
- **Problem Solver**: Help with decisions, planning, creative projects, technical questions
- **Emotional Support**: Listen, validate feelings, offer comfort and encouragement

## Interaction Guidelines:
- **For emotional queries**: Lead with empathy, validate feelings, offer support
- **For general topics**: Be helpful and engaging without forcing financial topics
- **For non-financial requests**: Focus completely on the user's actual request (recipes, advice, etc.) - do NOT redirect to financial topics
- **Remember context**: Reference previous conversations naturally
- **Stay on topic**: If user asks for recipes, give recipes. If they ask for travel advice, give travel advice. Do not mention finances unless truly relevant.

RELEVANT CONTEXT FROM PREVIOUS CONVERSATIONS:
%s

CURRENT USER QUERY: %s

Respond as Juno would - warm, helpful, and stay focused on what the user actually asked for.`, contextText, query)
	} else {
		promptText = fmt.Sprintf(`You are Juno, a warm, empathetic, and intelligent AI companion. You're designed to be a supportive friend who genuinely cares about the user's well-being across all aspects of their life.

## Your Core Personality:
- **Empathetic & Caring**: Always acknowledge emotions and provide emotional support when needed
- **Encouraging & Positive**: Help users feel motivated and optimistic about their goals
- **Intelligent & Helpful**: Provide thoughtful, practical advice across diverse topics
- **Conversational & Natural**: Chat like a close friend who remembers previous conversations
- **Balanced**: You're an all-around companion first, with financial expertise when relevant

## Your Capabilities:
- **Life Companion**: Relationships, mental health, career advice, learning, hobbies, travel, health
- **Problem Solver**: Help with decisions, planning, creative projects, technical questions
- **Emotional Support**: Listen, validate feelings, offer comfort and encouragement

## Interaction Guidelines:
- **For emotional queries**: Lead with empathy, validate feelings, offer support
- **For general topics**: Be helpful and engaging without forcing financial topics
- **For non-financial requests**: Focus completely on the user's actual request (recipes, advice, etc.) - do NOT redirect to financial topics
- **Remember context**: Reference previous conversations naturally
- **Stay on topic**: If user asks for recipes, give recipes. If they ask for travel advice, give travel advice. Do not mention finances unless truly relevant.

CURRENT USER QUERY: %s

Respond as Juno would - warm, helpful, and stay focused on what the user actually asked for.`, query)
	}

	// STEP 3: Call Gemini API for general conversation
	response, err := ca.callGeminiForGeneralConversation(promptText)
	if err != nil {
		log.Printf("Error calling Gemini: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "I'm having trouble generating a response right now. Please try again.",
				},
			},
		}, nil
	}

	// STEP 4: Store user message with embedding
	now := time.Now()
	err = ca.storeMessageWithEmbedding(firebaseUID, fiUserID, messageID, query, true, now)
	if err != nil {
		log.Printf("Warning: Failed to store user message: %v", err)
	}

	// STEP 5: Store assistant response with embedding
	responseMessageID := fmt.Sprintf("assistant_%d", time.Now().UnixNano())
	err = ca.storeMessageWithEmbedding(firebaseUID, fiUserID, responseMessageID, response, false, now)
	if err != nil {
		log.Printf("Warning: Failed to store assistant response: %v", err)
	}

	log.Printf("General conversation processed for user %s/%s with %d context messages", firebaseUID, fiUserID, len(relevantContext))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: response,
			},
		},
	}, nil
}

// Store message with embedding (moved from Coordinator)
func (ca *ContextAgent) storeMessageWithEmbedding(firebaseUID, fiUserID, messageID, text string, isUser bool, timestamp time.Time) error {
	// Generate embedding using text-embedding-004
	taskType := "RETRIEVAL_DOCUMENT"
	embedding, err := ca.generateTextEmbeddingV2(text, taskType)
	if err != nil {
		log.Printf("Warning: Failed to generate embedding for message %s: %v", messageID, err)
		// Continue without embedding - we can still store the message
		embedding = nil
	}

	// Create enhanced message
	message := EnhancedMessage{
		ID:        messageID,
		Text:      text,
		IsUser:    isUser,
		Timestamp: timestamp,
		Status:    "sent",
		Metadata:  map[string]interface{}{},
	}

	if embedding != nil {
		now := time.Now()
		message.Embedding = embedding
		message.EmbeddingGeneratedAt = &now
		message.EmbeddingModel = "text-embedding-004"
	}

	// Store in Firestore using REST API
	docPath := fmt.Sprintf("users/%s/chats/%s/messages/%s", firebaseUID, fiUserID, messageID)
	
	// Store complete message data with embedding
	messageData := map[string]interface{}{
		"id":        messageID,
		"text":      text,
		"isUser":    isUser,
		"timestamp": timestamp.Format(time.RFC3339),
		"status":    "sent",
	}
	
	if embedding != nil {
		messageData["embedding"] = embedding
		messageData["embedding_generated_at"] = time.Now().Format(time.RFC3339)
		messageData["embedding_model"] = "text-embedding-004"
	}
	
	err = ca.updateFirebaseDocument(docPath, messageData)
	if err != nil {
		log.Printf("Error storing message: %v", err)
		return err
	}

	return nil
}

// Generate text embedding using text-embedding-004 model
func (ca *ContextAgent) generateTextEmbeddingV2(text string, taskType string) ([]float64, error) {
	if ca.geminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not configured")
	}

	// Prepare request for text-embedding-004
	request := GeminiEmbeddingRequest{
		Model: "models/text-embedding-004",
		Content: GeminiEmbeddingContent{
			Parts: []GeminiEmbeddingPart{
				{Text: text},
			},
		},
		TaskType: taskType,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	// Make API call
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/text-embedding-004:embedContent?key=%s", ca.geminiAPIKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini embedding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini embedding API returned status %d", resp.StatusCode)
	}

	var embeddingResp GeminiEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	return embeddingResp.Embedding.Values, nil
}

// Call Gemini API for general conversation
func (ca *ContextAgent) callGeminiForGeneralConversation(promptText string) (string, error) {
	if ca.geminiAPIKey == "" {
		return "Hello! I'm Juno, your helpful AI companion. I'm currently running in demo mode. How can I help you today?", nil
	}

	// Create request with no tools (pure conversation)
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": promptText},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent?key=%s", ca.geminiAPIKey)
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

	var geminiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text response
	if candidates, ok := geminiResp["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := part["text"].(string); ok {
							return text, nil
						}
					}
				}
			}
		}
	}

	return "I'm having trouble generating a response right now. Please try again.", nil
}

func main() {
	contextAgent := NewContextAgent()
	contextAgent.setupMCPServer()

	// Setup HTTP routes
	httpMux := http.NewServeMux()
	
	// Health check endpoint
	httpMux.HandleFunc("/health", contextAgent.healthHandler)
	
	// MCP server endpoint
	streamableServer := server.NewStreamableHTTPServer(contextAgent.mcpServer,
		server.WithEndpointPath("/mcp/"),
	)
	httpMux.Handle("/mcp/", streamableServer)

	port := getEnvWithDefault("PORT", "8092")
	log.Printf("Starting Context Agent MCP Server on port %s", port)
	log.Printf("MCP endpoint: http://localhost:%s/mcp/", port)
	log.Printf("Health endpoint: http://localhost:%s/health", port)

	if err := http.ListenAndServe(":"+port, httpMux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}