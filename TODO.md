# Juno Financial Assistant - Development Status

## COMPLETED TASKS ✅

### 1. Docker Build Fix

- **Fixed**: Mobile app Dockerfile - removed pubspec.lock from COPY command
- **File**: `/mobile_app/Dockerfile`
- **Issue**: pubspec.lock wasn't generated yet during Docker build

### 2. Architecture Simplification

- **Removed**: User selection screen - direct navigation to ChatScreen
- **Files**: `/mobile_app/lib/main.dart`, `/mobile_app/lib/screens/chat_screen.dart`
- **Deleted**: `/mobile_app/lib/screens/user_selection_screen.dart`

### 3. MCP Protocol Implementation

- **Fixed**: Proper MCP tool calling instead of keyword detection
- **File**: `/backend/coordinator_mcp/main.go`
- **Change**: Coordinator now calls Claude API with Fi tools exposed, Claude decides when to call Fi

### 4. Authentication Flow

- **Implemented**: Browser-based Fi authentication
- **Files**: `/mobile_app/lib/services/websocket_service.dart`, `/mobile_app/lib/widgets/message_widget.dart`
- **Flow**: Fi returns login_required → Mobile app shows login button → Opens browser

### 5. Code Cleanup (Recently Completed)

- **Removed**: Unused `callClaudeAPI()` function
- **Removed**: Unused phone number parameters from all Fi tool definitions
- **Fixed**: MCP client API usage - `NewStreamableHttpClient` and proper `CallToolRequest` struct
- **Fixed**: Function signatures to remove phone number parameters throughout

### 6. Login Required Response Fix

- **Fixed**: login_required responses now bypass Claude processing
- **File**: `/backend/coordinator_mcp/main.go:431-433`
- **Change**: When Fi returns login_required JSON, pass it directly to mobile app instead of letting Claude convert to text

## RECENTLY COMPLETED ✅

### 7. Session Persistence Fix (MAJOR SUCCESS!)

- **FIXED**: User can now login once and stay logged in for subsequent requests
- **Solution**: Implemented persistent Fi MCP client connection
- **Files Modified**: `/backend/coordinator_mcp/main.go`
- **Changes Made**:
  1. ✅ Added `fiMCPClient *client.Client` to CoordinatorServer struct
  2. ✅ Added `initializeFiClient()` method called once at startup
  3. ✅ Replaced `callFiMCPTool()` to use persistent client (no more `defer Close()`)
  4. ✅ Session now maintained across all requests - LOGIN WORKS!

### 8. Multi-User Support Implementation (PHASE 2 COMPLETE!)

- **SOLVED**: Multiple users can now have separate Fi sessions and data isolation
- **Solution**: Per-User Fi Client Pool with Thread Safety
- **Files Modified**: `/backend/coordinator_mcp/main.go`
- **Changes Made**:
  1. ✅ Added `fiClients map[string]*client.Client` client pool
  2. ✅ Added `sync.Mutex` for thread-safe concurrent access
  3. ✅ Implemented `getOrCreateFiClient(userId)` method
  4. ✅ Updated all function signatures to accept and pass userId
  5. ✅ Enhanced WebSocket processing to extract userId from mobile app
  6. ✅ Added fallback compatibility (defaults to "1111111111")
  7. ✅ Fixed all compiler warnings and modernized code (`interface{}` → `any`)
  8. ✅ Added proper error handling and logging per user
  9. ✅ **TESTED & WORKING**: Each user dropdown selection creates separate Fi session

## PENDING TASKS 📋

### ✅ COMPLETED: Multi-User App Authentication (WAS CRITICAL FOR HACKATHON) ✅

- **Challenge**: Multiple hackathon participants will interfere with each other's Fi sessions
- **Problem**: Person A logs into Fi user "1111111111", Person B sees Person A's data
- **Solution**: Firebase Auth to isolate each app user's access to the 16 Fi test datasets

#### Firebase Auth Implementation Plan:

**Goal**: Give each app user their own isolated set of 16 Fi test users

**Phase 1: Firebase Setup & Research** ✅ **COMPLETED**

- ✅ Research Firebase free tier limits (10K monthly active users confirmed)
- ✅ Set up Firebase project: `juno-financial-assistant`
- ✅ Configure auth providers: Email, Google, Anonymous (perfect for judge testing)
- ✅ Firebase config obtained for Flutter integration
- ✅ Project ID: `juno-financial-assistant`
- ✅ App ID: `1:929051225142:web:1d59d1710c38785ea0bc97`

**Phase 2: Backward-Compatible Coordinator Changes** ✅ **COMPLETED**

- ✅ Extended WebSocket protocol to accept optional `firebaseUID` parameter
- ✅ Updated client pool key generation: `${firebaseUID}_${userId}` vs legacy `${userId}`
- ✅ Ensured existing functionality works without Firebase (fallback mode)
- ✅ Added Firebase user cleanup endpoint for logout
- ✅ Updated all function signatures to support Firebase isolation
- ✅ Added comprehensive logging for Firebase vs legacy mode
- ✅ Implemented `cleanupFirebaseUserClients()` method for proper resource cleanup
- ✅ **BACKWARD COMPATIBLE**: All existing functionality preserved

**Phase 3: Flutter Firebase Integration** ✅ **COMPLETED**

- ✅ Added Firebase SDK and FirebaseUI Auth to pubspec.yaml
- ✅ Created beautiful auth gate/landing page with login/signup/anonymous options
- ✅ Updated WebSocket service to include Firebase UID in messages
- ✅ Preserved existing dropdown and chat functionality
- ✅ Added Firebase configuration files and options
- ✅ Created AuthService for Firebase authentication management
- ✅ Updated ChatProvider to support Firebase UID parameter
- ✅ Added user indicator in ChatScreen AppBar
- ✅ Implemented logout functionality with cleanup
- ✅ **FULLY FUNCTIONAL**: Firebase auth + Fi isolation working together

**Phase 4: User Experience & Cleanup** ✅ **COMPLETED**

- ✅ **Logout button implemented**: PopupMenuButton with "Sign Out" option in AppBar with proper Fi client cleanup
- ✅ **User indicator in UI**: AppBar shows authenticated user display name (email/anonymous)
- ✅ **Complete flow tested**: Firebase login → Fi user selection → Fi auth → Logout working end-to-end
- ✅ **Anonymous auth flow**: Working perfectly for judges/mentors with "Quick Demo Access"
- ✅ **Firebase web compilation issue RESOLVED** by upgrading firebase_auth_web from 5.8.13 to 5.15.3

## ✅ COMPLETED: **Phase 4.5: Per-User Chat History with Pure Test Mode** ✅ (CRITICAL UX FIX)

**Goal**: Implement separate chat histories for each Fi test user to improve demo experience with perfect anonymous session isolation

### **Problem SOLVED**: 
When switching between Fi test users (1010101010, 1111111111, etc.) in dropdown, chat history persists, making it confusing to track which responses came from which user's data.

### **Solution IMPLEMENTED**: 
Per-user chat persistence with Firestore storage, two-level clear options, and **Pure Test Mode** for anonymous users

### **Implementation Tasks COMPLETED**:
- ✅ **Firestore Integration**: Added `cloud_firestore` dependency to pubspec.yaml
- ✅ **ChatProvider Enhancement**: Replaced in-memory storage with Firestore per-user chat storage
- ✅ **User Switching Logic**: Save current user's chat to Firestore, load selected user's chat history
- ✅ **Two-Level Clear Options**:
  - "Clear Chat" → Clears only current Fi user's chat from Firestore (`clearCurrentUserChat()`)
  - "Clear All Chats" → Clears chat history for ALL 16 Fi users from Firestore (`clearAllUsersChats()`)
- ✅ **Welcome Message Logic**: Add welcome message only for first-time user selection
- ✅ **UI Integration**: Updated `_onUserChanged()` method to switch chat contexts with Firestore
- ✅ **PopupMenu Enhancement**: Added "Clear All Chats" option with `Icons.delete_sweep`
- ✅ **Code Quality**: Fixed all Flutter analyzer issues and debug print statements
- ✅ **Auth Session Isolation**: Fixed auto-login issue - always shows auth screen on container rebuild
- ✅ **Firebase UID Change Detection**: Prevents chat history bleeding between different auth methods
- ✅ **Pure Test Mode**: Anonymous users get completely ephemeral sessions with automatic cleanup

### **Technical Implementation Details**:
- ✅ **Firestore Structure**: `/users/{firebaseUID}/chats/{userId}/messages/{messageId}`
- ✅ **Per-User Isolation**: Each Fi user (1010101010-9999999999) has separate chat collection
- ✅ **Firebase User Isolation**: Each Firebase authenticated user gets their own data space
- ✅ **Automatic Persistence**: Messages saved to Firestore immediately on send/receive
- ✅ **Batch Operations**: Efficient Firestore batch operations for clearing chats
- ✅ **Error Handling**: Graceful fallback to local storage if Firestore fails
- ✅ **Background Saving**: Current chat automatically saved when switching users
- ✅ **Auth Method Isolation**: Anonymous and email users have completely separate data spaces
- ✅ **Anonymous Data Cleanup**: Complete Firestore deletion on anonymous sign out

### **Pure Test Mode Features**:
- ✅ **Ephemeral Anonymous Sessions**: Each anonymous login gets unique Firebase UID
- ✅ **Complete Data Isolation**: Anonymous sessions never interfere with each other
- ✅ **Automatic Cleanup**: All anonymous user data deleted from Firestore on sign out
- ✅ **Perfect Judge Experience**: Each judge gets completely fresh database state
- ✅ **Scalable Testing**: Unlimited anonymous sessions without data accumulation

### **Benefits ACHIEVED**:
- ✅ Each Fi user maintains separate conversation context across sessions
- ✅ Judges can switch between users and continue previous conversations within session
- ✅ Anonymous judges get completely fresh experience every time
- ✅ No data pollution between different judges/sessions  
- ✅ Email users have persistent data, anonymous users have ephemeral data
- ✅ "Clear All Chats" gives fresh start for current user
- ✅ Perfect hackathon demo experience with clean database hygiene
- ✅ **RAG-ready**: Persistent storage for future context analysis

## ✅ COMPLETED: **Phase 5.1: Google Ecosystem Migration** ✅ (HACKATHON STRATEGY)

**Goal**: Switch to full Google AI stack for maximum hackathon scoring with Google judges

### **Migration Tasks COMPLETED**:
- ✅ **Gemini 2.5 Flash Lite Integration**: Replaced Claude API with Gemini in coordinator
- ✅ **Environment Variables**: Switched from `ANTHROPIC_API_KEY` to `GEMINI_API_KEY`
- ✅ **Request/Response Format**: Converted Claude format to Gemini API format
- ✅ **Function Calling**: Migrated Claude tools to Gemini function declarations
- ✅ **API Endpoints**: Updated to Google Generative Language API
- ✅ **Critical Bug Fixes**: 
  - Fixed Fi login URL port issue (internal 8080 → external 8090)
  - Fixed duplicate message display bug (multiple stream subscriptions)
  - Fixed Fi MCP connection issues

### **Technical Implementation**:
- ✅ **API URL**: `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent`
- ✅ **Model**: `gemini-2.5-flash-lite` (optimized for speed and cost-effectiveness)
- ✅ **Function Calling**: All Fi tools converted to Gemini function declarations
- ✅ **Request Format**: Claude messages → Gemini contents with parts structure
- ✅ **Response Parsing**: Claude content → Gemini candidates with parts handling
- ✅ **Environment Configuration**: Docker and .env files updated for Gemini integration

### **Benefits ACHIEVED**:
- ✅ **Google Ecosystem Bonus**: Maximum hackathon points with Google judges
- ✅ **Performance**: Faster responses with Flash Lite optimization
- ✅ **Cost Efficiency**: Better price/performance ratio
- ✅ **Unified Stack**: Complete Google AI integration (Firebase + Firestore + Gemini)

## NEXT PRIORITY: **Phase 5.2: Microservices Deployment Architecture** 🏗️ (SCALABILITY)

**Goal**: Migrate from Docker monolith to separate Firebase-hosted microservices for better development workflow and scalability

### **Strategic Advantage**:
- 🏆 **Google ecosystem bonus points** for Google-hosted hackathon
- 🚀 **Latest Google AI technology** showcase
- 🔥 **Unified tech stack** - All Google services
- 💡 **Innovation demonstration** - Cutting-edge Google AI integration

### **Implementation Tasks**:

#### **5.1: Switch from Claude to Gemini 2.5 Flash Lite** ✅ **COMPLETED**
- ✅ **Update coordinator MCP**: Replace Anthropic API calls with Gemini API
- ✅ **Change API endpoint**: `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent`
- ✅ **Update request format**: Convert Claude API format to Gemini API format
- ✅ **Tool calling migration**: Adapt MCP tool calling to Gemini function calling
- ✅ **Environment variable**: Switch from `ANTHROPIC_API_KEY` to `GEMINI_API_KEY`
- ✅ **Critical Bug Fixes**: Fixed Fi login URL port issue and duplicate message display bug

#### **5.2: Implement Gemini Embeddings for RAG**
- **Add embedding service**: Create `GeminiEmbeddingService` in Flutter app
- **Use latest model**: `text-embedding-005` (verify latest available)
- **Embedding types**: 
  - `RETRIEVAL_DOCUMENT` for storing chat messages
  - `RETRIEVAL_QUERY` for RAG search queries
- **Firestore integration**: Store embeddings with chat messages for future RAG

#### **5.3: Complete Google Stack Integration**
- ✅ **Firebase Auth** - Already implemented
- ✅ **Firestore** - Chat storage with embeddings
- 🆕 **Gemini 2.5 Flash Lite** - Main conversational AI
- 🆕 **Gemini Embeddings** - RAG and semantic search
- 🆕 **Google Cloud Functions** - Backend processing (if needed)

### **Technical Specifications**:
```go
// Gemini API integration
GEMINI_API_KEY=your-gemini-key
Model: gemini-2.5-flash-lite
Embedding: text-embedding-005
Dimensions: 768 (verify latest)
```

### **Benefits**:
- ✅ **Fast responses** - Gemini 2.5 Flash Lite optimized for speed
- ✅ **Cost effective** - Great price/performance ratio  
- ✅ **Future-ready** - RAG capabilities with embeddings
- ✅ **Hackathon advantage** - Full Google AI showcase

## LATER: **Phase 6: Voice Assistant Integration** 🎙️ (PREMIUM FEATURE)

**Goal**: Add multi-modal voice capabilities to create a truly conversational AI financial assistant

### **Strategic Advantage**:
- 🎯 **Hackathon differentiator** - Most teams won't have voice integration
- 🌍 **Multi-language support** - Global accessibility 
- 🚀 **Premium UX** - Voice-first financial AI experience
- 🏆 **Google stack showcase** - Full GCP AI services demonstration

### **Required GCP Services** (for teammate to enable):
```bash
# Core Voice APIs
- Speech-to-Text API (speech.googleapis.com)
- Text-to-Speech API (texttospeech.googleapis.com)  
- Cloud Translation API (translate.googleapis.com)
- Generative Language API (generativelanguage.googleapis.com) # Already planned

# Supporting APIs  
- Cloud Functions API (cloudfunctions.googleapis.com)
- Cloud Storage API (storage.googleapis.com)
- Vertex AI API (aiplatform.googleapis.com)
```

### **Implementation Tasks**:

#### **6.1: Flutter Voice Input**
- **Add speech recognition**: Integrate Speech-to-Text API in Flutter
- **Voice button UI**: Add microphone button to chat input area
- **Audio recording**: Implement audio capture and streaming
- **Real-time transcription**: Show live speech-to-text conversion
- **Voice activity detection**: Auto-start/stop recording

#### **6.2: Voice Output Integration**  
- **Text-to-Speech service**: Convert Gemini responses to speech
- **Voice selection**: Multiple voice options (male/female, different accents)
- **Audio streaming**: Real-time audio playback
- **Speech controls**: Play/pause/speed controls
- **Background playback**: Continue audio while using other features

#### **6.3: Multi-Language Support**
- **Language detection**: Auto-detect user's language
- **Translation integration**: Cloud Translation API for multi-language queries
- **Localized responses**: Gemini responses in user's preferred language
- **Voice localization**: Native TTS voices for different languages
- **Supported languages**: English, Spanish, Hindi, Mandarin, French (expandable)

#### **6.4: Voice-Optimized UX**
- **Conversational flow**: Voice-first interaction patterns
- **Audio feedback**: Sound effects for voice actions
- **Visual voice indicators**: Waveform visualization during recording
- **Voice shortcuts**: "Hey Juno" wake word using Picovoice Porcupine (FREE)
- **Hands-free mode**: Complete voice-only operation with wake word activation
- **Wake word integration**: 
  - Create custom "Hey Juno" wake word via Picovoice Console (free)
  - Integrate `porcupine_flutter` plugin for offline detection
  - Auto-start voice input when wake word detected
  - Background listening with minimal battery impact

### **Technical Architecture**:
```
Wake Word Detection ("Hey Juno") → Voice Input → Speech-to-Text → 
Translation (if needed) → Gemini → Translation (if needed) → 
Text-to-Speech → Voice Output
```

### **Free Services Stack**:
- **Wake Word**: Picovoice Porcupine (FREE tier)
- **Speech-to-Text**: GCP Speech-to-Text API ($300 credits)
- **AI Processing**: Gemini 2.5 Flash Lite ($300 credits)
- **Text-to-Speech**: GCP Text-to-Speech API ($300 credits)
- **Translation**: GCP Translation API ($300 credits)

### **Benefits**:
- ✅ **Accessibility** - Voice-first financial assistance
- ✅ **Multi-language** - Global user support
- ✅ **Premium UX** - Conversational AI experience  
- ✅ **Hackathon edge** - Advanced multi-modal AI demonstration
- ✅ **Google showcase** - Full GCP AI stack utilization

## LATER: **Phase 7: Demo Polish** ✨

**Goal**: Perfect the hackathon demonstration experience

### Remaining Tasks:
- **Landing page explaining demo and Fi datasets**
  - Welcome screen for judges explaining the 16 Fi test users (1010101010-9999999999)
  - Quick overview of available financial data per user
  - Instructions for judges to get started quickly

- **Smooth onboarding for hackathon judges**
  - Streamlined anonymous login flow
  - Clear UI guidance for first-time users
  - Demo scenarios suggestions ("Try asking: 'What's my net worth?'")

- **Enhanced error handling and loading states**
  - Better loading indicators during Fi authentication
  - Graceful error messages for network issues
  - Connection retry mechanisms and fallback states

- **Documentation for mentors/judges**
  - README with demo instructions
  - Quick reference for hackathon features
  - Troubleshooting guide for common issues

### PRIORITY 2: Production Optimizations (LOW PRIORITY)

- **Enhancement**: Add connection health monitoring
- **Enhancement**: Performance optimizations

#### Multi-User Architecture Plan:

**Implementation Approach: Per-User Fi Client Pool**

```
User A ──┐
User B ──┤── Coordinator ──┤── Fi Connection A (SessionId A, Phone: 1111111111)
User C ──┘                 ├── Fi Connection B (SessionId B, Phone: 2222222222)  
                           └── Fi Connection C (SessionId C, Phone: 3333333333)
```

#### Changes Required:

1. **Mobile App User Selection**

   - Add dropdown with 16 test phone numbers (1010101010 to 9999999999)
   - **IMPORTANT**: Use same UI theme, design, and Material 3 components as existing app
   - Follow existing design patterns from ChatScreen (colors, spacing, typography)
   - Integrate seamlessly with current purple gradient theme and card designs
   - **Design References**:
     * Color scheme: `ColorScheme.fromSeed(seedColor: Color(0xFF6750A4))` (Material Purple)
     * App bar style with gradient avatar and "Juno" branding
     * Input field styling from chat input area with rounded corners
     * Elevation and shadow patterns from existing containers
   - **Dropdown Design**: Liquid Glass/Glassmorphism Effect
     * Semi-transparent background with backdrop blur (`BackdropFilter`)
     * Subtle gradient overlay matching purple theme
     * Smooth rounded corners and soft shadows
     * Animated transitions for open/close states
     * Glass-like border with opacity
     * Placement: AppBar next to Juno title
   - Update WebSocket protocol to include `userId` in messages
   - UI element to select "user" for testing different Fi datasets
2. **Coordinator Client Pool**

   ```go
   type CoordinatorServer struct {
       // ... existing fields
       fiClients map[string]*client.Client // Pool of Fi clients per user
       clientsMu sync.Mutex                // Thread safety for concurrent users
   }
   ```
3. **Dynamic Client Management**

   - `getOrCreateFiClient(userId string)` method
   - Check fiClients map for existing client per user
   - Create new persistent client if not exists
   - Automatic login_required handling per user session
4. **Session Isolation Benefits**

   - Each user gets own persistent Fi connection
   - Complete data isolation (no cross-user data leakage)
   - Scalable to hundreds/thousands of concurrent users
   - Maintains performance benefits of persistent connections
5. **Testing Infrastructure**

   - Test all 16 Fi phone number datasets independently
   - Verify session isolation between concurrent users
   - Load testing with multiple simultaneous users

#### Implementation Priority: ✅ **ALL PHASES COMPLETE**

- **Phase 1**: Mobile app user selection UI (foundation) ✅ **COMPLETED & TESTED**
- **Phase 2**: Coordinator client pool implementation ✅ **COMPLETED & TESTED**
- **Phase 3**: Dynamic client management with thread safety ✅ **COMPLETED & TESTED**
- **Phase 4**: Testing with all 16 datasets simultaneously ✅ **COMPLETED & TESTED**

#### Multi-User System Status: ✅ **FULLY OPERATIONAL**

- ✅ **Phase 1 - Liquid Glass Dropdown**: Beautiful glassmorphism UI with 16 test users
- ✅ **Phase 1 - AppBar Integration**: Seamlessly integrated next to Juno branding
- ✅ **Phase 1 - Overlay Positioning**: Fixed visibility issues, renders above AppBar
- ✅ **Phase 1 - Smooth Animations**: Scale, rotation, and opacity transitions working
- ✅ **Phase 1 - Click Outside to Close**: Proper UX with gesture detection
- ✅ **Phase 1 - User Selection**: All 16 phone numbers selectable (1010101010 to 9999999999)
- ✅ **Phase 1 - WebSocket Protocol**: Updated to include userId in messages
- ✅ **Phase 1 - Clean Code**: All Flutter warnings fixed, production ready
- ✅ **Phase 2 - Client Pool**: Thread-safe map of Fi clients per user
- ✅ **Phase 2 - Dynamic Management**: `getOrCreateFiClient(userId)` working
- ✅ **Phase 2 - Session Isolation**: Complete data separation between users
- ✅ **Phase 2 - Persistent Sessions**: Login once per user, stay logged in
- ✅ **Phase 3 - Thread Safety**: Concurrent user support with sync.Mutex
- ✅ **Phase 3 - Error Handling**: Proper logging and fallback mechanisms
- ✅ **Phase 4 - Testing Complete**: All 16 Fi datasets tested independently
- ✅ **Phase 4 - User Flow Verified**: Dropdown → Login → Switch → No Re-login

## CURRENT ARCHITECTURE ✅ **UPDATED WITH MULTI-USER SUPPORT**

### Mobile App Flow

1. App starts → ChatScreen with user dropdown selector
2. User selects from 16 Fi test users (1010101010, 1111111111, etc.)
3. ChatProvider initializes → WebSocket connects to coordinator with userId
4. User message → WebSocket JSON-RPC → Coordinator (includes userId)

### Coordinator Flow (Per-User)

1. Receives process_query with userId → Gets/Creates Fi client for that user
2. Calls Gemini 2.5 Flash Lite API with Fi tools available for specific user
3. Gemini detects financial query → Calls fetch_net_worth tool
4. Coordinator calls Fi MCP using user's dedicated client
5. Fi returns login_required (first time) OR user's financial data
6. Response flows back to mobile app

### Authentication Flow (Per-User Session)

1. User selects phone number (e.g., 1111111111) from dropdown
2. Fi returns login_required JSON with sessionId for that user
3. Mobile app shows "Login to Fi Money" button
4. Button opens Fi login page → User logs in with selected phone number
5. **FIXED**: Session persists in user's dedicated Fi client
6. Switch to different user → New login required for that user
7. Switch back to original user → **NO re-login needed** ✅

## FILES MODIFIED

### Mobile App

- `/mobile_app/Dockerfile` - Fixed pubspec.lock issue
- `/mobile_app/lib/main.dart` - Direct ChatScreen navigation
- `/mobile_app/lib/screens/chat_screen.dart` - Removed user parameters
- `/mobile_app/lib/services/websocket_service.dart` - Added login_required handling
- `/mobile_app/lib/widgets/message_widget.dart` - Added login button
- **DELETED**: `/mobile_app/lib/screens/user_selection_screen.dart`

### Backend Coordinator

- `/backend/coordinator_mcp/main.go` - **MAJOR MULTI-USER IMPLEMENTATION**:
  - ✅ Added per-user Fi client pool with thread safety
  - ✅ Implemented `getOrCreateFiClient(userId)` method
  - ✅ Updated all function signatures to pass userId
  - ✅ Enhanced WebSocket processing for userId extraction
  - ✅ Fixed all compiler warnings and modernized code
  - ✅ Added comprehensive error handling and logging
  - ✅ Maintained backward compatibility with fallback defaults

## ENVIRONMENT

- **Working Directory**: `/Users/sreenivasg/Desktop/Projects/Juno/backend/coordinator_mcp`
- **Docker**: Use `docker-compose build --no-cache && docker-compose up`
- **Test Data**: Fi has test phone numbers like 1111111111, any OTP works

## NEXT SESSION INSTRUCTIONS

1. Read this TODO.txt file first
2. Read `/Users/sreenivasg/Desktop/Projects/Juno/backend/coordinator_mcp/main.go`
3. Focus on `callFiMCPTool()` function - this creates new clients each time
4. Implement persistent Fi MCP client to maintain login sessions
5. Test the complete login flow end-to-end

## TESTING COMMANDS

```bash
# Rebuild and test
cd /Users/sreenivasg/Desktop/Projects/Juno
docker-compose build --no-cache && docker-compose up

# Test login
# 1. Open mobile app at http://localhost:3000
# 2. Ask "What's my net worth?"  
# 3. Click login button, use phone: 1111111111, OTP: 123456
# 4. Return to chat, ask again - should NOT ask for login again
```
