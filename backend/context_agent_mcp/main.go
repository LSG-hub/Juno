package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ContextAgent struct {
	mcpServer *server.Server
	fiMCPURL  string
}

type UserContext struct {
	UserID           string                 `json:"user_id"`
	LastActivity     time.Time              `json:"last_activity"`
	Location         string                 `json:"location,omitempty"`
	RecentEvents     []string               `json:"recent_events,omitempty"`
	SpendingPatterns map[string]interface{} `json:"spending_patterns,omitempty"`
	Preferences      map[string]interface{} `json:"preferences,omitempty"`
}

func NewContextAgent() *ContextAgent {
	return &ContextAgent{
		fiMCPURL: getEnvWithDefault("FI_MCP_URL", "http://fi-mcp-server:8080"),
	}
}

func (ca *ContextAgent) setupMCPServer() {
	ca.mcpServer = server.NewMCPServer(
		"Context Agent MCP",
		"0.1.0",
		server.WithInstructions("Juno Context Agent MCP Server - Provides user context and environmental awareness for financial decisions"),
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	// Add context analysis tools
	ca.mcpServer.AddTool(
		mcp.NewTool("analyze_user_context", mcp.WithDescription("Analyze user's current context for financial decision making")),
		ca.handleAnalyzeContext,
	)

	ca.mcpServer.AddTool(
		mcp.NewTool("get_spending_patterns", mcp.WithDescription("Get user's spending patterns and behavioral insights")),
		ca.handleGetSpendingPatterns,
	)

	ca.mcpServer.AddTool(
		mcp.NewTool("detect_life_events", mcp.WithDescription("Detect significant life events from user data patterns")),
		ca.handleDetectLifeEvents,
	)

	ca.mcpServer.AddTool(
		mcp.NewTool("get_temporal_context", mcp.WithDescription("Get time-based context for financial decisions")),
		ca.handleGetTemporalContext,
	)
}

func (ca *ContextAgent) handleAnalyzeContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params, ok := request.Params.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid parameters"), nil
	}

	userID, ok := params["user_id"].(string)
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
		return mcp.NewToolResultError("Failed to marshal context data"), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
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
		return mcp.NewToolResultError("Failed to marshal spending patterns"), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
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
		return mcp.NewToolResultError("Failed to marshal life events"), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
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
		return mcp.NewToolResultError("Failed to marshal temporal context"), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

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

	port := getEnvWithDefault("PORT", "8082")
	log.Printf("Starting Context Agent MCP Server on port %s", port)
	log.Printf("MCP endpoint: http://localhost:%s/mcp/", port)
	log.Printf("Health endpoint: http://localhost:%s/health", port)

	if err := http.ListenAndServe(":"+port, httpMux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}