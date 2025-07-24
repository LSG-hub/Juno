# 🚀 Juno Microservices Deployment Guide

**Transform from Docker Monolith to Firebase-Hosted Microservices**

## 🎯 **Architecture Overview**

### **Current State (Docker Monolith)**
```
Docker Container
├── fi-mcp-server (Port 8090)
├── coordinator-mcp (Port 8091) 
├── context-agent-mcp (Port 8092)
├── security-agent-mcp (Port 8093)
└── mobile-app (Port 3000)
```

### **Target State (Firebase Microservices)**
```
Firebase Ecosystem
├── 📱 mobile-app → Firebase Hosting
├── 🤖 coordinator-mcp → Firebase Functions
├── 💰 fi-mcp-server → Firebase Functions  
├── 🧠 context-agent-mcp → Firebase Functions
└── 🔒 security-agent-mcp → Firebase Functions
```

## 📋 **Step-by-Step Migration Plan**

---

## **Phase 1: Repository Structure Setup** 🏗️

### **1.1 Create Separate Repositories**

```bash
# Create parent directory
mkdir ~/Juno-Microservices
cd ~/Juno-Microservices

# Create individual repositories
git clone <current-repo> juno-mobile-app
git clone <current-repo> juno-coordinator-mcp
git clone <current-repo> juno-fi-mcp
git clone <current-repo> juno-context-agent
git clone <current-repo> juno-security-agent
```

### **1.2 Repository Structure**

```
~/Juno-Microservices/
├── juno-mobile-app/           # Flutter Web App
│   ├── lib/
│   ├── web/
│   ├── pubspec.yaml
│   ├── firebase.json
│   └── .firebaserc
│
├── juno-coordinator-mcp/      # Main Orchestration Service
│   ├── functions/
│   │   ├── main.go
│   │   ├── go.mod
│   │   └── go.sum
│   ├── firebase.json
│   └── .firebaserc
│
├── juno-fi-mcp/              # Fi Money Integration
│   ├── functions/
│   │   ├── main.go
│   │   ├── go.mod  
│   │   └── go.sum
│   ├── firebase.json
│   └── .firebaserc
│
├── juno-context-agent/       # Context & RAG Service
│   ├── functions/
│   │   ├── main.go
│   │   ├── go.mod
│   │   └── go.sum
│   ├── firebase.json
│   └── .firebaserc
│
└── juno-security-agent/      # Security & Risk Assessment
    ├── functions/
    │   ├── main.go
    │   ├── go.mod
    │   └── go.sum
    ├── firebase.json
    └── .firebaserc
```

---

## **Phase 2: Mobile App Migration (Firebase Hosting)** 📱

### **2.1 Repository: `juno-mobile-app`**

```bash
cd ~/Juno-Microservices/juno-mobile-app

# Copy Flutter app
cp -r ~/Desktop/Projects/Juno/mobile_app/* .

# Initialize Firebase
firebase init hosting
```

### **2.2 Firebase Configuration**

**firebase.json**
```json
{
  "hosting": {
    "public": "build/web",
    "ignore": [
      "firebase.json",
      "**/.*",
      "**/node_modules/**"
    ],
    "rewrites": [
      {
        "source": "**",
        "destination": "/index.html"
      }
    ]
  }
}
```

**Environment Configuration**
```dart
// lib/config/environment.dart
class Environment {
  static const String coordinatorUrl = String.fromEnvironment(
    'COORDINATOR_URL',
    defaultValue: 'https://us-central1-juno-financial-assistant.cloudfunctions.net/coordinator',
  );
  
  static const String wsUrl = String.fromEnvironment(
    'WS_URL', 
    defaultValue: 'wss://us-central1-juno-financial-assistant.cloudfunctions.net/coordinator-ws',
  );
}
```

### **2.3 Build and Deploy**

```bash
# Build Flutter web
flutter build web --dart-define=COORDINATOR_URL=https://us-central1-juno-financial-assistant.cloudfunctions.net/coordinator

# Deploy to Firebase Hosting
firebase deploy --only hosting
```

**Live URL**: `https://juno-financial-assistant.web.app`

---

## **Phase 3: Coordinator MCP Service (Firebase Functions)** 🤖

### **3.1 Repository: `juno-coordinator-mcp`**

```bash
mkdir -p ~/Juno-Microservices/juno-coordinator-mcp/functions
cd ~/Juno-Microservices/juno-coordinator-mcp

# Copy coordinator code
cp ~/Desktop/Projects/Juno/backend/coordinator_mcp/main.go functions/
cp ~/Desktop/Projects/Juno/backend/coordinator_mcp/go.mod functions/
cp ~/Desktop/Projects/Juno/backend/coordinator_mcp/go.sum functions/

# Initialize Firebase Functions
firebase init functions
```

### **3.2 Firebase Functions Configuration**

**firebase.json**
```json
{
  "functions": [
    {
      "source": "functions",
      "codebase": "coordinator",
      "runtime": "go121"
    }
  ]
}
```

**functions/go.mod** (Updated)
```go
module juno-coordinator

go 1.21

require (
    github.com/GoogleCloudPlatform/functions-framework-go v1.8.0
    github.com/gorilla/websocket v1.5.0
    github.com/mark3labs/mcp-go v0.1.0
)
```

### **3.3 Adapt for Firebase Functions**

**functions/main.go** (Key Changes)
```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    
    "github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
    functions.HTTP("coordinator", coordinatorHandler)
    functions.HTTP("coordinator-ws", coordinatorWSHandler)
}

func coordinatorHandler(w http.ResponseWriter, r *http.Request) {
    // HTTP endpoint for REST API calls
    cs := NewCoordinatorServer()
    cs.handleHTTP(w, r)
}

func coordinatorWSHandler(w http.ResponseWriter, r *http.Request) {
    // WebSocket endpoint for real-time communication
    cs := NewCoordinatorServer()
    cs.handleWebSocket(w, r)
}

func main() {
    // Use PORT environment variable or default to 8080
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

### **3.4 Environment Variables**

```bash
# Set Firebase Functions environment variables
firebase functions:config:set \
  gemini.api_key="AIzaSyBvIzIMpPcqUduNF6rSUL2o-ClYWO4GtTA" \
  fi.mcp_url="https://us-central1-juno-financial-assistant.cloudfunctions.net/fi-mcp" \
  context.agent_url="https://us-central1-juno-financial-assistant.cloudfunctions.net/context-agent" \
  security.agent_url="https://us-central1-juno-financial-assistant.cloudfunctions.net/security-agent"
```

### **3.5 Deploy**

```bash
firebase deploy --only functions:coordinator
```

**Live URL**: `https://us-central1-juno-financial-assistant.cloudfunctions.net/coordinator`

---

## **Phase 4: Fi MCP Service (Firebase Functions)** 💰

### **4.1 Repository: `juno-fi-mcp`**

```bash
mkdir -p ~/Juno-Microservices/juno-fi-mcp/functions
cd ~/Juno-Microservices/juno-fi-mcp

# Copy Fi MCP server code
cp -r ~/Desktop/Projects/Juno/fi-mcp-server/* functions/

# Initialize Firebase
firebase init functions
```

### **4.2 Adapt Fi Server for Functions**

**functions/main.go**
```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    
    "github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
    functions.HTTP("fi-mcp", fiMCPHandler)
}

func fiMCPHandler(w http.ResponseWriter, r *http.Request) {
    // Handle Fi MCP requests
    server := NewFiMCPServer()
    server.handleRequest(w, r)
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

### **4.3 Deploy**

```bash
firebase deploy --only functions:fi-mcp
```

**Live URL**: `https://us-central1-juno-financial-assistant.cloudfunctions.net/fi-mcp`

---

## **Phase 5: Context Agent Service (Firebase Functions)** 🧠

### **5.1 Repository: `juno-context-agent`**

```bash
mkdir -p ~/Juno-Microservices/juno-context-agent/functions
cd ~/Juno-Microservices/juno-context-agent

# Copy context agent code
cp -r ~/Desktop/Projects/Juno/backend/context_agent_mcp/* functions/

firebase init functions
```

### **5.2 Adapt for Functions**

**functions/main.go**
```go
func init() {
    functions.HTTP("context-agent", contextAgentHandler)
}

func contextAgentHandler(w http.ResponseWriter, r *http.Request) {
    server := NewContextAgentServer()
    server.handleRequest(w, r)
}
```

### **5.3 Deploy**

```bash
firebase deploy --only functions:context-agent
```

---

## **Phase 6: Security Agent Service (Firebase Functions)** 🔒

### **6.1 Repository: `juno-security-agent`**

```bash
mkdir -p ~/Juno-Microservices/juno-security-agent/functions
cd ~/Juno-Microservices/juno-security-agent

# Copy security agent code  
cp -r ~/Desktop/Projects/Juno/backend/security_agent_mcp/* functions/

firebase init functions
```

### **6.2 Deploy**

```bash
firebase deploy --only functions:security-agent
```

---

## **Phase 7: Inter-Service Communication** 🔗

### **7.1 Update Service URLs**

**Coordinator Service URLs**
```go
// In coordinator functions/main.go
const (
    FI_MCP_URL = "https://us-central1-juno-financial-assistant.cloudfunctions.net/fi-mcp"
    CONTEXT_AGENT_URL = "https://us-central1-juno-financial-assistant.cloudfunctions.net/context-agent"  
    SECURITY_AGENT_URL = "https://us-central1-juno-financial-assistant.cloudfunctions.net/security-agent"
)
```

### **7.2 CORS Configuration**

**All Functions need CORS**
```go
func enableCORS(w http.ResponseWriter, r *http.Request) bool {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
    
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return true
    }
    return false
}
```

---

## **Phase 8: Development Workflow** 🛠️

### **8.1 Individual Service Development**

```bash
# Work on mobile app
cd ~/Juno-Microservices/juno-mobile-app
flutter run -d chrome
firebase deploy --only hosting

# Work on coordinator
cd ~/Juno-Microservices/juno-coordinator-mcp  
firebase functions:shell
firebase deploy --only functions:coordinator

# Work on Fi service
cd ~/Juno-Microservices/juno-fi-mcp
firebase deploy --only functions:fi-mcp
```

### **8.2 Local Testing**

```bash
# Run Firebase emulator suite
firebase emulators:start --only functions,hosting,firestore

# Test individual functions
curl http://localhost:5001/juno-financial-assistant/us-central1/coordinator
```

### **8.3 Environment Management**

**Development**: Firebase Emulators  
**Staging**: Firebase Functions (staging project)  
**Production**: Firebase Functions (production project)

---

## **Phase 9: Benefits Achieved** ✅

### **9.1 Development Benefits**
- ✅ **Independent Development**: Change one service without rebuilding others
- ✅ **No Docker Dependency**: Native Firebase development workflow
- ✅ **Hot Reload**: Instant deployment and testing
- ✅ **Scalable Architecture**: Each service scales independently

### **9.2 Hackathon Benefits**  
- ✅ **Maximum Google Points**: Full Firebase ecosystem usage
- ✅ **Production Ready**: Real-world scalable architecture
- ✅ **Demo URLs**: Live, shareable links for judges
- ✅ **Zero Downtime**: Firebase's global CDN and functions

### **9.3 Operational Benefits**
- ✅ **Auto-scaling**: Firebase Functions scale automatically
- ✅ **Monitoring**: Built-in Firebase Analytics and Logging
- ✅ **Security**: Firebase IAM and security rules
- ✅ **Cost Effective**: Pay-per-invocation pricing

---

## **🎯 Migration Timeline**

**Day 1**: Setup repositories and mobile app hosting  
**Day 2**: Migrate coordinator and Fi MCP services  
**Day 3**: Migrate context and security agents  
**Day 4**: Test inter-service communication  
**Day 5**: Production deployment and optimization

**Ready to start the migration?** 🚀