# Location-Aware Web Search Tool - Implementation Plan

## 🎯 **Concept: Geographic Intelligence for Financial Decisions**

Transform Juno into a **location-aware financial assistant** that understands geographic context and provides location-specific financial advice through real-time web search capabilities.

---

## 🏗️ **Architecture: Gemini Native Web Search Integration**

### **Current Multi-User Architecture** ✅
```
Firebase User A
├── Fi User 1010101010 (isolated chat + location context)
├── Fi User 1111111111 (isolated chat + location context)
├── ...
└── Fi User 9999999999 (isolated chat + location context)

Firebase User B
├── Fi User 1010101010 (completely separate from User A)
├── ...
```

### **Enhanced Tool Flow with Location Context** 🚀
```
User Query → Location Detection → Coordinator → RAG Context → Gemini with Tools:
                     ↓                                      ├── Fi Tools (existing)
                Location Context                            ├── Life Events Tool (existing)
                     ↓                                      └── Web Search Tool (NEW)
                Web Search Query
```

**Gemini Intelligence**: Let Gemini decide when location-based web search is needed based on query understanding and detected location context.

---

## 🛠️ **Tool Implementation**

### **1. New Coordinator MCP Tool**

**Tool Name**: `web_search_with_location`

**Description**: 
```
Perform real-time web searches with geographic context for location-specific 
financial advice. Uses Gemini's native "Grounding with Google Search" to find 
current information about real estate markets, local regulations, investment 
opportunities, and financial services based on user's location.
```

**Gemini Function Declaration**:
```json
{
  "name": "web_search_with_location",
  "description": "Search the web for location-specific financial information. Use when user asks about real estate, local markets, regulations, or location-specific financial advice.",
  "parameters": {
    "type": "object",
    "properties": {
      "search_query": {
        "type": "string",
        "description": "The specific search query to execute"
      },
      "location_context": {
        "type": "object",
        "description": "User's location context for search refinement",
        "properties": {
          "city": {"type": "string"},
          "state": {"type": "string"}, 
          "country": {"type": "string"},
          "coordinates": {
            "type": "object",
            "properties": {
              "latitude": {"type": "number"},
              "longitude": {"type": "number"}
            }
          }
        }
      },
      "search_intent": {
        "type": "string",
        "enum": ["real_estate", "local_markets", "regulations", "services", "investment_opportunities", "general"],
        "description": "The type of location-specific information needed"
      }
    },
    "required": ["search_query"]
  }
}
```

### **2. Gemini Native Web Search Integration**

**Implementation Method**: Use Gemini 2.5 Flash Lite's built-in "Grounding with Google Search" capability.

**Technical Approach**:
```go
// Add Google Search tool to Gemini request
type GeminiTool struct {
    FunctionDeclarations []GeminiFunction `json:"functionDeclarations,omitempty"`
    GoogleSearch        *GoogleSearchTool `json:"googleSearch,omitempty"`
}

type GoogleSearchTool struct {
    // Empty object enables Google Search grounding
}

// Enhanced request structure
requestBody := GeminiRequest{
    Contents: []GeminiContent{...},
    Tools: []GeminiTool{
        {
            // Existing Fi function declarations
            FunctionDeclarations: [...],
        },
        {
            // Enable native Google Search
            GoogleSearch: &GoogleSearchTool{},
        },
    },
}
```

### **3. When Gemini Should Use Web Search**

**Gemini's Intelligence** will determine when to search based on:
- Real estate queries ("best areas to buy apartment in Bangalore")
- Local market conditions ("current property prices in Mumbai")
- Location-specific regulations ("home loan rates in Delhi")
- Investment opportunities ("mutual fund agents near me")
- Local financial services ("best banks in Chennai")
- Market trends and comparisons ("rent vs buy in Pune")

**No hardcoded triggers** - let Gemini understand location context naturally!

---

## 📱 **Flutter Location Detection Implementation**

### **Package Dependencies**
```yaml
dependencies:
  geolocator: ^10.1.0  # Location services
  permission_handler: ^11.0.1  # Permission management
```

### **Location Service Class**
```dart
// lib/services/location_service.dart
class LocationService {
  static LocationService? _instance;
  static LocationService get instance => _instance ??= LocationService._();
  LocationService._();

  Position? _lastKnownPosition;
  String? _lastKnownCity;
  String? _lastKnownState;
  String? _lastKnownCountry;

  Future<Map<String, dynamic>?> getCurrentLocation() async {
    try {
      // Check permissions
      LocationPermission permission = await Geolocator.checkPermission();
      if (permission == LocationPermission.denied) {
        permission = await Geolocator.requestPermission();
        if (permission == LocationPermission.denied) {
          return null;
        }
      }

      // Get current position
      Position position = await Geolocator.getCurrentPosition(
        desiredAccuracy: LocationAccuracy.medium,
        timeLimit: Duration(seconds: 10),
      );

      _lastKnownPosition = position;

      // Reverse geocoding for city/state/country
      List<Placemark> placemarks = await placemarkFromCoordinates(
        position.latitude,
        position.longitude,
      );

      if (placemarks.isNotEmpty) {
        Placemark place = placemarks.first;
        _lastKnownCity = place.locality;
        _lastKnownState = place.administrativeArea;
        _lastKnownCountry = place.country;

        return {
          'coordinates': {
            'latitude': position.latitude,
            'longitude': position.longitude,
          },
          'city': _lastKnownCity,
          'state': _lastKnownState,
          'country': _lastKnownCountry,
          'accuracy': position.accuracy,
          'timestamp': position.timestamp?.toIso8601String(),
        };
      }
    } catch (e) {
      debugPrint('Location error: $e');
    }
    return null;
  }

  Map<String, dynamic>? getLastKnownLocation() {
    if (_lastKnownPosition != null) {
      return {
        'coordinates': {
          'latitude': _lastKnownPosition!.latitude,
          'longitude': _lastKnownPosition!.longitude,
        },
        'city': _lastKnownCity,
        'state': _lastKnownState,
        'country': _lastKnownCountry,
        'cached': true,
      };
    }
    return null;
  }
}
```

### **Enhanced WebSocket Service**
```dart
// Update sendMessage to include location context
Future<String> sendMessage(String message, String userId, {String? firebaseUID}) async {
  // Get location context
  final locationService = LocationService.instance;
  Map<String, dynamic>? location = await locationService.getCurrentLocation();
  
  // Fallback to last known location if current detection fails
  location ??= locationService.getLastKnownLocation();

  final Map<String, dynamic> params = {
    'query': message,
    'userId': userId,
  };
  
  if (firebaseUID != null && firebaseUID.isNotEmpty) {
    params['firebaseUID'] = firebaseUID;
  }
  
  // Add location context if available
  if (location != null) {
    params['location_context'] = location;
  }

  // Rest of sendMessage implementation...
}
```

---

## 🌐 **Coordinator MCP Implementation**

### **Enhanced Request Processing**
```go
func (cs *CoordinatorServer) handleProcessQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    arguments := request.GetArguments()
    query, _ := arguments["query"].(string)
    userId, _ := arguments["userId"].(string)
    firebaseUID, _ := arguments["firebaseUID"].(string)
    
    // Extract location context from mobile app
    locationContext, _ := arguments["location_context"].(map[string]interface{})
    
    // Get RAG context from Context Agent first
    contextResult := cs.getRAGContextFromContextAgent(query, userId, firebaseUID)
    
    // Call Gemini API with tools, RAG context, and location context
    response, err := cs.callGeminiAPIWithToolsAndLocation(query, userId, firebaseUID, contextResult, locationContext)
    
    // Rest of implementation...
}
```

### **Enhanced Gemini API Call with Web Search**
```go
func (cs *CoordinatorServer) callGeminiAPIWithToolsAndLocation(query string, userId string, firebaseUID string, contextResult map[string]interface{}, locationContext map[string]interface{}) (string, error) {
    // Define tools including Google Search
    tools := []GeminiTool{
        {
            // Existing Fi function declarations
            FunctionDeclarations: []GeminiFunction{
                // All existing Fi tools...
            },
        },
        {
            // Enable native Google Search grounding
            GoogleSearch: &GoogleSearchTool{},
        },
    }

    // Build enhanced prompt with location context
    promptText := cs.buildPromptWithLocationContext(query, contextResult, locationContext)

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

    // Rest of API call implementation...
}

func (cs *CoordinatorServer) buildPromptWithLocationContext(query string, contextResult map[string]interface{}, locationContext map[string]interface{}) string {
    var promptBuilder strings.Builder
    
    // Add location context if available
    if locationContext != nil {
        promptBuilder.WriteString("USER LOCATION CONTEXT:\n")
        if city, ok := locationContext["city"].(string); ok && city != "" {
            promptBuilder.WriteString(fmt.Sprintf("City: %s\n", city))
        }
        if state, ok := locationContext["state"].(string); ok && state != "" {
            promptBuilder.WriteString(fmt.Sprintf("State: %s\n", state))
        }
        if country, ok := locationContext["country"].(string); ok && country != "" {
            promptBuilder.WriteString(fmt.Sprintf("Country: %s\n", country))
        }
        promptBuilder.WriteString("\n")
    }

    // Add RAG context
    if contextResult != nil {
        if ragContext, ok := contextResult["context"].(string); ok && ragContext != "" {
            promptBuilder.WriteString("RELEVANT CONVERSATION CONTEXT:\n")
            promptBuilder.WriteString(ragContext)
            promptBuilder.WriteString("\n\n")
        }
    }

    // Add enhanced instructions for location-aware responses
    promptBuilder.WriteString(`You are Juno, an AI Financial Assistant with access to:
1. User's financial data through Fi tools
2. Previous conversation context through RAG
3. Real-time web search through Google Search grounding
4. User's current location context

When responding to location-specific queries (real estate, local markets, regulations), 
use the Google Search grounding to find current, location-relevant information.

USER QUERY: `)
    promptBuilder.WriteString(query)

    return promptBuilder.String()
}
```

---

## 🎨 **User Experience Design**

### **Location-Aware Response Examples**

**Without Location Context** (current):
```
"I'd recommend looking at property prices in your area, but I don't have access to current market data."
```

**With Location Context + Web Search** (enhanced):
```
"Based on your location in Bangalore and current market data:

🏠 **Current Market Analysis** (from recent search):
- Average apartment prices in your area: ₹8,000-12,000 per sq ft
- Best localities for ₹30L budget: Whitefield, Electronic City, Sarjapur
- Recent price trends: 5-8% growth in the last 12 months

🎯 **Personalized Recommendation**:
Given your ₹17L net worth and Bangalore location, I suggest:
1. **Focus Areas**: Whitefield (emerging IT hub, good connectivity)
2. **Property Size**: 2BHK (1000-1200 sq ft) fits your budget
3. **Timing**: Current market is stable, good time to buy
4. **Financing**: You'll need ₹13L additional funding - check SBI/HDFC rates

Would you like me to search for specific properties or current home loan rates in Bangalore?"
```

### **Smart Location Detection Patterns**

**Proactive Location Context**:
```
When user asks: "What are the best areas to buy an apartment?"
Juno responds: "I can give you much better recommendations if I know your location. May I access your current location to find the best areas in your city?"
```

**Location-Specific Follow-ups**:
```
After location-based advice: "Since you're in Mumbai, would you also like me to check:
- Current stamp duty rates in Maharashtra
- Best home loan providers in your area  
- Upcoming infrastructure projects near potential properties"
```

---

## 🔄 **Implementation Strategy**

### **Phase 1: Flutter Location Integration** 
1. Add `geolocator` and `permission_handler` dependencies
2. Create `LocationService` class for location detection
3. Update `WebSocketService` to include location context
4. Add location permission requests in UI

### **Phase 2: Coordinator Web Search Enhancement**
1. Add Google Search tool to Gemini function declarations
2. Update request processing to handle location context
3. Enhance prompt building with location information
4. Test Gemini's automatic web search triggering

### **Phase 3: Smart Integration**
1. Location-based query detection and enhancement
2. Automatic web search for location-specific financial queries
3. Cached location for performance optimization
4. Location privacy controls and permissions

### **Phase 4: Advanced Features**
1. Location-based financial service recommendations
2. Local market trend analysis and alerts
3. Regulatory updates specific to user's location
4. Community-based financial insights

---

## 🎯 **Key Benefits**

### **For Users**:
- **Location-Relevant Advice**: Financial guidance specific to their city/state
- **Real-Time Market Data**: Current prices, trends, and opportunities
- **Local Regulatory Awareness**: State-specific rules, taxes, and benefits
- **Contextual Recommendations**: Services and opportunities in their area

### **For Juno**:
- **Differentiation**: Location-aware AI vs generic financial assistants
- **Accuracy**: Real-time data vs outdated information
- **Relevance**: Locally applicable advice increases user trust
- **Engagement**: Users rely on Juno for location-specific decisions

---

## 💰 **Cost Considerations**

### **Gemini Web Search Pricing**:
- **Cost**: $35 per 1,000 grounded queries
- **Billing**: Per API request that includes google_search tool
- **Optimization**: Multiple searches in single request = single charge

### **Location Services**:
- **Flutter Geolocator**: Free, uses device GPS
- **Reverse Geocoding**: Uses device services, no additional API costs
- **Caching**: Store last location to minimize repeated API calls

---

## 🚀 **Success Metrics**

- **Location Detection Rate**: % of queries that successfully detect location
- **Web Search Utilization**: % of location-specific queries that trigger web search
- **Response Relevance**: User feedback on location-aware vs generic advice
- **Market Data Accuracy**: Comparison of Juno's data vs actual market conditions
- **User Engagement**: Increased conversation depth for location-specific queries

---

## 🔒 **Privacy & Security**

### **Location Privacy**:
- **User Consent**: Explicit permission for location access
- **Data Minimization**: Store only city/state, not precise coordinates
- **Opt-out**: Users can disable location features anytime
- **Transparency**: Clear indication when location is used

### **Search Privacy**:
- **No Personal Data**: Never include personal financial details in search queries
- **Generalized Queries**: "Bangalore property prices" not "John's apartment search"
- **Anonymized**: Web searches don't contain user identifiers

---

## 🎯 **Immediate Next Steps**

1. **Add** location dependencies to Flutter `pubspec.yaml`
2. **Create** `LocationService` class for location detection
3. **Update** `WebSocketService` to include location context
4. **Implement** Google Search tool in Coordinator MCP
5. **Add** location-aware prompt building
6. **Test** Gemini's intelligence in location-based search decisions

---

## 💡 **Advanced Future Enhancements**

### **Predictive Location Intelligence**
- Learn user's frequent locations (home, work) for better recommendations
- Predict location-based financial needs (commute costs, local services)
- Seasonal location-aware advice (festival expenses, travel planning)

### **Multi-Location Support**
- Track multiple properties/locations for investment analysis
- Compare opportunities across different cities
- Migration planning with financial implications

### **Community Integration**
- Anonymous local financial insights from other users
- Crowdsourced market data and recommendations
- Local expert connections and referrals

This transforms Juno from a location-agnostic financial calculator into a **geo-intelligent financial companion** that understands the critical role of location in financial decisions! 🚀🌍