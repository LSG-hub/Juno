package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type SecurityAgent struct {
	mcpServer *server.MCPServer
	fiMCPURL  string
}

type SecurityAssessment struct {
	UserID           string                 `json:"user_id"`
	OverallRisk      string                 `json:"overall_risk"`
	RiskScore        float64                `json:"risk_score"`
	EmergencyFund    EmergencyFundStatus    `json:"emergency_fund"`
	DebtAnalysis     DebtAnalysis           `json:"debt_analysis"`
	InsuranceCoverage InsuranceCoverage     `json:"insurance_coverage"`
	Recommendations  []string               `json:"recommendations"`
	Alerts           []SecurityAlert        `json:"alerts,omitempty"`
}

type EmergencyFundStatus struct {
	CurrentAmount    float64 `json:"current_amount"`
	RecommendedAmount float64 `json:"recommended_amount"`
	MonthsCovered    float64 `json:"months_covered"`
	Status           string  `json:"status"`
}

type DebtAnalysis struct {
	TotalDebt        float64            `json:"total_debt"`
	DebtToIncomeRatio float64           `json:"debt_to_income_ratio"`
	CreditUtilization float64           `json:"credit_utilization"`
	DebtBreakdown    map[string]float64 `json:"debt_breakdown"`
	Status           string             `json:"status"`
}

type InsuranceCoverage struct {
	LifeInsurance   InsuranceStatus `json:"life_insurance"`
	HealthInsurance InsuranceStatus `json:"health_insurance"`
	Status          string          `json:"status"`
}

type InsuranceStatus struct {
	HasCoverage bool    `json:"has_coverage"`
	Coverage    float64 `json:"coverage_amount"`
	Premium     float64 `json:"premium_amount"`
	Adequate    bool    `json:"adequate"`
}

type SecurityAlert struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	ActionItems []string  `json:"action_items"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewSecurityAgent() *SecurityAgent {
	return &SecurityAgent{
		fiMCPURL: getEnvWithDefault("FI_MCP_URL", "http://localhost:8090"),
	}
}

func (sa *SecurityAgent) setupMCPServer() {
	sa.mcpServer = server.NewMCPServer(
		"security-agent-mcp",
		"0.1.0",
		server.WithInstructions(`Juno Security Agent MCP Server - The conservative guardian that prioritizes financial safety and risk management.

CORE MISSION:
• **Risk-First Approach**: Evaluate all financial decisions through a security lens
• **Conservative Bias**: Prioritize financial stability and loss prevention
• **Protective Strategy**: Recommend defensive financial measures and safeguards
• **Long-term Security**: Focus on sustainable financial health over quick gains

SPECIALIZED CAPABILITIES:
1. **Comprehensive Security Assessment**:
   - Overall financial risk scoring (1-10 scale)
   - Emergency fund adequacy analysis (6+ months coverage)
   - Debt-to-income ratio evaluation
   - Credit utilization monitoring
   - Insurance coverage gap analysis

2. **Risk Analytics**:
   - Portfolio volatility assessment
   - Credit risk evaluation
   - Liquidity risk analysis
   - Scenario-based stress testing
   - Multi-factor risk modeling

3. **Protective Recommendations**:
   - Emergency fund optimization strategies
   - Insurance coverage recommendations
   - Debt consolidation and payoff plans
   - Risk mitigation techniques
   - Financial safety net establishment

4. **Proactive Monitoring**:
   - Generate security alerts and warnings
   - Track risk factor changes over time
   - Identify potential financial vulnerabilities
   - Recommend preventive measures

DECISION-MAKING PERSONALITY:
• **Loss Aversion Priority**: Preventing financial loss takes precedence
• **Stability Focus**: Favor stable, predictable financial strategies
• **Conservative Recommendations**: Err on the side of caution
• **Defensive Mindset**: Build financial fortresses, not castles in the air

This agent serves as Juno's financial conscience, ensuring users maintain strong financial foundations before pursuing growth opportunities.`),
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	// Add security analysis tools
	sa.mcpServer.AddTool(
		mcp.NewTool("assess_financial_security",
			mcp.WithDescription("Comprehensive financial security assessment"),
			mcp.WithString("user_id",
				mcp.Description("User ID for security assessment"),
			),
		),
		sa.handleAssessFinancialSecurity,
	)

	sa.mcpServer.AddTool(
		mcp.NewTool("analyze_emergency_fund",
			mcp.WithDescription("Analyze emergency fund adequacy"),
			mcp.WithString("user_id",
				mcp.Description("User ID for emergency fund analysis"),
			),
		),
		sa.handleAnalyzeEmergencyFund,
	)

	sa.mcpServer.AddTool(
		mcp.NewTool("evaluate_debt_risk",
			mcp.WithDescription("Evaluate debt and credit risk"),
			mcp.WithString("user_id",
				mcp.Description("User ID for debt risk evaluation"),
			),
		),
		sa.handleEvaluateDebtRisk,
	)

	sa.mcpServer.AddTool(
		mcp.NewTool("check_insurance_gaps",
			mcp.WithDescription("Identify insurance coverage gaps"),
			mcp.WithString("user_id",
				mcp.Description("User ID for insurance gap analysis"),
			),
		),
		sa.handleCheckInsuranceGaps,
	)

	sa.mcpServer.AddTool(
		mcp.NewTool("generate_security_alerts",
			mcp.WithDescription("Generate security-related alerts and warnings"),
			mcp.WithString("user_id",
				mcp.Description("User ID for security alerts"),
			),
		),
		sa.handleGenerateSecurityAlerts,
	)
}

func (sa *SecurityAgent) handleAssessFinancialSecurity(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	userID, ok := arguments["user_id"].(string)
	if !ok {
		userID = "default_user"
	}

	// For MVP, return mock security assessment for userID
	assessment := SecurityAssessment{
		UserID:      userID,
		OverallRisk: "low",
		RiskScore:   2.3, // Scale of 1-10
		EmergencyFund: EmergencyFundStatus{
			CurrentAmount:     170000,
			RecommendedAmount: 150000,
			MonthsCovered:     6.8,
			Status:           "adequate",
		},
		DebtAnalysis: DebtAnalysis{
			TotalDebt:         85000,
			DebtToIncomeRatio: 0.12,
			CreditUtilization: 0.25,
			DebtBreakdown: map[string]float64{
				"credit_cards":    25000,
				"personal_loans":  60000,
				"other":          0,
			},
			Status: "manageable",
		},
		InsuranceCoverage: InsuranceCoverage{
			LifeInsurance: InsuranceStatus{
				HasCoverage: true,
				Coverage:    5000000,
				Premium:     15000,
				Adequate:    true,
			},
			HealthInsurance: InsuranceStatus{
				HasCoverage: true,
				Coverage:    500000,
				Premium:     8000,
				Adequate:    true,
			},
			Status: "well_covered",
		},
		Recommendations: []string{
			"Emergency fund is well-maintained",
			"Debt levels are under control",
			"Insurance coverage is adequate",
			"Consider increasing investment allocation",
			"Monitor credit utilization to keep below 30%",
		},
		Alerts: []SecurityAlert{
			{
				Type:     "maintenance",
				Severity: "low",
				Message:  "Credit card utilization slightly elevated",
				ActionItems: []string{
					"Consider paying down credit card balance",
					"Monitor monthly spending patterns",
				},
				CreatedAt: time.Now(),
			},
		},
	}

	jsonData, err := json.Marshal(assessment)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Failed to marshal security assessment",
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

func (sa *SecurityAgent) handleAnalyzeEmergencyFund(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Mock emergency fund analysis for MVP
	analysis := map[string]interface{}{
		"current_status": map[string]interface{}{
			"amount":         170000,
			"months_covered": 6.8,
			"target_months":  6,
			"status":         "adequate",
		},
		"monthly_expenses": 25000,
		"income_stability": map[string]interface{}{
			"stability_score": 0.85,
			"income_sources":  1,
			"variability":     "low",
		},
		"recommendations": []string{
			"Emergency fund exceeds target by 0.8 months",
			"Current fund can cover 6.8 months of expenses",
			"Consider investing surplus beyond 6 months",
			"Maintain automatic monthly contributions",
		},
		"risk_factors": []string{
			"Single income source",
			"Industry-specific risks to monitor",
		},
		"optimization_tips": []string{
			"Keep emergency fund in high-yield savings",
			"Consider laddering some funds in FDs",
			"Review and adjust quarterly",
		},
	}

	jsonData, err := json.Marshal(analysis)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Failed to marshal emergency fund analysis",
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

func (sa *SecurityAgent) handleEvaluateDebtRisk(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Mock debt risk evaluation for MVP
	evaluation := map[string]interface{}{
		"debt_summary": map[string]interface{}{
			"total_debt":           85000,
			"monthly_payments":     7500,
			"debt_to_income":      0.12,
			"credit_utilization":  0.25,
			"payment_history":     "excellent",
		},
		"risk_assessment": map[string]interface{}{
			"overall_risk":    "low",
			"risk_score":      2.8,
			"risk_factors": []string{
				"Credit utilization at 25% (slightly elevated)",
				"Good payment history",
				"Manageable debt-to-income ratio",
			},
		},
		"debt_breakdown": map[string]interface{}{
			"credit_cards": map[string]interface{}{
				"balance":      25000,
				"limit":        100000,
				"utilization":  0.25,
				"min_payment":  2500,
				"interest_rate": 0.18,
			},
			"personal_loans": map[string]interface{}{
				"balance":        60000,
				"monthly_emi":    5000,
				"remaining_term": 12,
				"interest_rate":  0.11,
			},
		},
		"recommendations": []string{
			"Consider paying down credit card balance to reduce utilization",
			"Continue current loan payments on schedule",
			"Avoid taking additional debt in near term",
			"Build credit score by maintaining low utilization",
		},
		"alerts": []string{
			"Credit utilization above ideal 20% threshold",
		},
	}

	jsonData, err := json.Marshal(evaluation)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Failed to marshal debt risk evaluation",
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

func (sa *SecurityAgent) handleCheckInsuranceGaps(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Mock insurance gap analysis for MVP
	analysis := map[string]interface{}{
		"coverage_summary": map[string]interface{}{
			"life_insurance": map[string]interface{}{
				"current_coverage": 5000000,
				"recommended":      5000000,
				"gap":             0,
				"adequacy":        "adequate",
				"annual_premium":  15000,
			},
			"health_insurance": map[string]interface{}{
				"current_coverage": 500000,
				"recommended":      500000,
				"gap":             0,
				"adequacy":        "adequate",
				"annual_premium":  8000,
			},
			"disability_insurance": map[string]interface{}{
				"current_coverage": 0,
				"recommended":      300000,
				"gap":             300000,
				"adequacy":        "gap_identified",
			},
		},
		"risk_exposure": map[string]interface{}{
			"uninsured_risks": []string{
				"Disability income protection",
				"Critical illness coverage",
			},
			"coverage_gaps": []string{
				"No disability insurance",
				"Consider top-up health insurance",
			},
		},
		"recommendations": []string{
			"Life insurance coverage is adequate",
			"Health insurance meets current needs",
			"Consider disability insurance for income protection",
			"Evaluate critical illness coverage",
			"Review beneficiary nominations",
		},
		"priority_actions": []string{
			"Research disability insurance options",
			"Compare critical illness policies",
			"Annual review of coverage amounts",
		},
	}

	jsonData, err := json.Marshal(analysis)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Failed to marshal insurance gap analysis",
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

func (sa *SecurityAgent) handleGenerateSecurityAlerts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Mock security alerts for MVP
	alerts := map[string]interface{}{
		"active_alerts": []SecurityAlert{
			{
				Type:     "credit_utilization",
				Severity: "medium",
				Message:  "Credit card utilization is at 25% - consider reducing to below 20%",
				ActionItems: []string{
					"Pay down credit card balance by ₹5,000",
					"Monitor monthly spending on credit cards",
					"Consider increasing credit limit if needed",
				},
				CreatedAt: time.Now().AddDate(0, 0, -2),
			},
			{
				Type:     "insurance_gap",
				Severity: "low",
				Message:  "No disability insurance coverage detected",
				ActionItems: []string{
					"Research disability insurance options",
					"Get quotes from multiple providers",
					"Consider term vs. whole life options",
				},
				CreatedAt: time.Now().AddDate(0, 0, -7),
			},
		},
		"monitoring": []string{
			"Credit score changes",
			"Debt-to-income ratio trends",
			"Emergency fund balance",
			"Insurance coverage adequacy",
			"Unusual spending patterns",
		},
		"upcoming_reviews": []map[string]interface{}{
			{
				"type":        "insurance_review",
				"due_date":    time.Now().AddDate(0, 3, 0).Format("2006-01-02"),
				"description": "Annual insurance coverage review",
			},
			{
				"type":        "emergency_fund_review",
				"due_date":    time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
				"description": "Monthly emergency fund adequacy check",
			},
		},
	}

	jsonData, err := json.Marshal(alerts)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Failed to marshal security alerts",
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

func (sa *SecurityAgent) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "security-agent-mcp",
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
	securityAgent := NewSecurityAgent()
	securityAgent.setupMCPServer()

	// Setup HTTP routes
	httpMux := http.NewServeMux()
	
	// Health check endpoint
	httpMux.HandleFunc("/health", securityAgent.healthHandler)
	
	// MCP server endpoint
	streamableServer := server.NewStreamableHTTPServer(securityAgent.mcpServer,
		server.WithEndpointPath("/mcp/"),
	)
	httpMux.Handle("/mcp/", streamableServer)

	port := getEnvWithDefault("PORT", "8093")
	log.Printf("Starting Security Agent MCP Server on port %s", port)
	log.Printf("MCP endpoint: http://localhost:%s/mcp/", port)
	log.Printf("Health endpoint: http://localhost:%s/health", port)

	if err := http.ListenAndServe(":"+port, httpMux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}