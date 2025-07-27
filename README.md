# Juno - AI Financial Assistant 🤖💰

A multi-agent AI financial assistant built with Flutter, Go, and Gemini AI that provides intelligent financial guidance with speech-to-text capabilities.

## 🚀 Quick Start

### Prerequisites
- **Flutter**: 3.32.7+
- **Go**: 1.21+
- **Chrome/Edge Browser**: Required for voice features
- **Firebase Project**: Set up with authentication enabled

### Clone and Setup
```bash
# Clone the repository
git clone https://github.com/LSG-hub/Juno.git
cd Juno

# Set up environment variables
cp .env.example .env
# Edit .env with your API keys and Firebase config

# Start all services
./start_juno.sh

# Stop all services when done
./stop_juno.sh
```

### Environment Configuration
Create a `.env` file from the example:
```bash
# Copy the example file
cp .env.example .env

# Edit with your actual values
GEMINI_API_KEY=your_gemini_api_key_here
GOOGLE_API_KEY=your_google_api_key_here
FI_MCP_PORT=8090
COORDINATOR_MCP_PORT=8091
CONTEXT_AGENT_PORT=8092
SECURITY_AGENT_PORT=8093
ENABLE_TRANSLATION=true
DEFAULT_LANGUAGE=en
```

**Required APIs:**
- **Gemini API Key**: Get from [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
- **Google API Key**: For translation services (can use same as Gemini)

## 📱 Access Points

Once started, access Juno through:

- **🌐 Mobile App**: http://localhost:3000
- **🔧 Fi MCP Server**: http://localhost:8090  
- **🎛️ Coordinator**: http://localhost:8091
- **🧠 Context Agent**: http://localhost:8092
- **🔒 Security Agent**: http://localhost:8093

## ✨ Features

- **🎤 Voice Input**: Speech-to-text functionality in supported browsers
- **🔐 Firebase Authentication**: Secure user authentication system
- **💬 Real-time Chat**: WebSocket-based communication
- **🧠 AI Memory**: Context-aware conversations with RAG
- **💰 Financial Data**: Integration with Fi.money services
- **🎨 Modern UI**: Fi.money green theme with gradient design
- **📱 Responsive**: Works on desktop and mobile browsers
- **🔄 Multi-User**: Support for multiple Fi users per Firebase account

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Flutter Web  │    │  Coordinator    │    │   Gemini API    │
│   (Port 3000)  │◄──►│   (Port 8091)   │◄──►│   (AI Engine)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                    ┌───────────┼───────────┐
                    │           │           │
         ┌─────────────────┐    │    ┌─────────────────┐
         │ Context Agent   │    │    │ Security Agent  │
         │  (Port 8092)    │◄───┼───►│  (Port 8093)    │
         └─────────────────┘    │    └─────────────────┘
                                │
                    ┌─────────────────┐
                    │  Fi MCP Server  │
                    │  (Port 8090)    │
                    └─────────────────┘
```

### Core Components

- **Frontend**: Flutter Web with enhanced UI and voice capabilities
- **Coordinator MCP**: Main orchestrator handling user queries and AI responses
- **Context Agent MCP**: RAG system for conversation memory and context
- **Security Agent MCP**: Security validation and threat detection
- **Fi MCP Server**: Financial data integration and transaction management
- **AI Engine**: Gemini 2.5 Flash Lite with function calling

## 📁 Project Structure

```
Juno/
├── mobile_app/              # Flutter web application
│   ├── lib/                # Dart source code
│   │   ├── services/       # Voice, auth, chat services
│   │   ├── widgets/        # UI components including voice button
│   │   └── screens/        # App screens
│   └── pubspec.yaml        # Flutter dependencies
├── backend/                # Go MCP servers
│   ├── coordinator_mcp/    # Main orchestration service
│   ├── context_agent_mcp/  # RAG and memory management
│   └── security_agent_mcp/ # Security validation
├── fi-mcp-server/          # Financial data MCP server
├── plan/                   # Documentation and future features
├── start_juno.sh          # Startup script
├── stop_juno.sh           # Shutdown script
└── .env                   # Environment configuration
```

## 🛠️ Development

### Testing the Application
1. **Start Services**: Run `./start_juno.sh`
2. **Open Browser**: Navigate to http://localhost:3000
3. **Sign In**: Use Firebase authentication
4. **Test Voice**: Click the microphone button (Chrome/Edge required)
5. **Chat**: Ask financial questions and test Fi tool integration

### Voice Features
- **Browser Support**: Chrome, Edge (HTTPS/localhost required)
- **Permissions**: Microphone access needed
- **Languages**: Supports multiple locales via Web Speech API
- **Visual Feedback**: Animated pulse during listening, real-time transcription

### Adding New Features
1. **Backend**: Add new tools to relevant MCP servers
2. **Frontend**: Update UI components in Flutter
3. **Integration**: Modify coordinator for new tool declarations
4. **Testing**: Verify with `./start_juno.sh`

## 🎯 Planned Features

### 🌟 Life Events Intelligence Tool
Transform Juno into a **life-aware financial companion** that understands major life events:

- **Smart Context Collection**: Detect marriages, job changes, home purchases, education plans
- **Life-Aware Advice**: Financial guidance that fits actual life circumstances
- **Timeline Integration**: Plan finances around major life transitions
- **Proactive Planning**: AI that remembers and plans ahead for life events

**Example**: Instead of generic advice, Juno considers your upcoming marriage when recommending apartment purchases, suggesting joint financing options and timeline adjustments.

### 🌍 Location-Aware Web Search Tool
Transform Juno into a **geo-intelligent financial companion** with real-time location context:

- **Location Detection**: Automatic GPS-based location detection in Flutter
- **Real-Time Market Data**: Current property prices, local regulations, investment opportunities
- **Gemini Web Search**: Native "Grounding with Google Search" for current information
- **Location-Specific Advice**: Financial guidance tailored to your city/state

**Example**: "Based on your location in Bangalore and current market data: Average apartment prices ₹8,000-12,000/sq ft, best areas for ₹30L budget: Whitefield, Electronic City..."

### Implementation Status
- ✅ **Core Platform**: Multi-agent architecture with voice capabilities
- 📋 **Life Events Tool**: Detailed implementation plan ready
- 📋 **Location Tool**: Technical specifications complete
- 🔄 **Next Phase**: Life events intelligence integration

## 🔧 Troubleshooting

### Common Issues
- **Voice not working**: Ensure Chrome/Edge browser with microphone permissions
- **Services not starting**: Check port availability (8090-8093, 3000)
- **Build failures**: Verify Go and Flutter versions
- **Authentication issues**: Check Firebase configuration in `.env`

### Logs and Debugging
- **Service logs**: Check individual MCP server outputs
- **Flutter logs**: Browser developer console
- **Stop/Start**: Use `./stop_juno.sh` then `./start_juno.sh` for fresh restart

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make changes and test with `./start_juno.sh`
4. Commit changes: `git commit -m "Add feature"`
5. Push and create Pull Request

## 📄 License

This project is licensed under the MIT License.

## 🔗 Links

- **Repository**: https://github.com/LSG-hub/Juno.git
- **Issues**: https://github.com/LSG-hub/Juno/issues
- **Documentation**: See `/plan` directory for detailed technical specs

---

Built with ❤️ using Flutter, Go, and Gemini AI