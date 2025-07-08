# Juno Mobile App Development Plan (Hackathon MVP Focus & Full Production)

## Overview

This document outlines a stage-by-stage development plan for the Juno Mobile App, specifically tailored for a hackathon MVP and then extending to a full production-ready application. The focus is on delivering a demonstrable, end-to-end voice interaction flow with minimal, yet functional, backend integration, while building a robust and scalable system for the long term. Each stage is designed to be chronological, modular, and deliver a tangible outcome.

## Core Architectural Principles (CTO Recommendations)

To ensure the success of the hackathon MVP and the long-term viability of the Juno application, the following architectural principles will be prioritized and integrated from the earliest stages:

1.  **MCP Protocol Compliance**: Strict adherence to the MCP JSON-RPC 2.0 protocol for all inter-service communication, ensuring standardization and future extensibility.
2.  **Robust Authentication & Security**: Implement proper Firebase JWT validation, service account authentication between MCP servers, and secure credential management from the outset.
3.  **Centralized Data Flow & Context Management**: Establish a shared context aggregation layer and session state management to ensure all agents operate with a unified and consistent understanding of user context.
4.  **Comprehensive Error Handling & Resilience**: Integrate circuit breaker patterns, graceful degradation, fallback mechanisms, and retry logic to ensure system stability and a smooth user experience even during failures.
5.  **Early Performance & Scalability**: Implement caching layers (Redis), connection pooling, and continuous performance monitoring to ensure the system can handle load and scale efficiently.

These principles will guide our implementation decisions throughout all development stages.

**Technology Stack**: Flutter (Mobile App - with Riverpod/Bloc, flutter_sound, Hive, Sentry), Python with FastAPI (Backend MCP Servers - with Redis for caching/messaging, PostgreSQL with connection pooling), Cloud Run for deployment

## Proposed Directory Structure

```
/Users/sreenivasg/Desktop/Projects/Juno/
├───mobile_app/                 # Flutter Mobile Application
│   ├───lib/
│   │   ├───main.dart
│   │   ├───screens/            # UI screens (e.g., home_screen.dart)
│   │   ├───services/           # API integrations, Firebase, etc. (e.g., audio_service.dart, stt_service.dart, tts_service.dart, websocket_service.dart)
│   │   └───models/             # Data models
│   ├───assets/                 # Images, fonts, etc.
│   ├───pubspec.yaml
│   └───...                     # Other Flutter project files
├───backend/                    # Backend MCP Servers
│   ├───coordinator_mcp/        # Coordinator MCP Server
│   │   ├───src/
│   │   │   ├───main.py         # or index.js (main application entry)
│   │   │   ├───websocket_handler.py # Handles WebSocket connections
│   │   │   └───agent_orchestrator.py # Basic orchestration logic
│   │   ├───config/             # Configuration files
│   │   ├───requirements.txt    # or package.json (dependencies)
│   │   └───...                 # Other project files
│   ├───context_agent_mcp/      # Context Agent MCP Server (Hackathon MVP Focus)
│   │   ├───src/
│   │   │   ├───main.py
│   │   │   └───fi_integration.py # Simple interaction with Fi's MCP
│   │   ├───config/
│   │   ├───requirements.txt
│   │   └───...
│   ├───fi_mcp_server/          # Placeholder/Reference for Fi's MCP (assumed existing and accessible)
│   │   └───...                 # (No code here, just a logical placeholder)
│   └───...                     # Other agents will be added here in future iterations
└───docs/                       # General documentation (e.g., API specifications, design docs)
```

## Development Tracks

To ensure focused development and clear dependencies, we will run two parallel development tracks:

1.  **Mobile App Development Track**: Focuses on the Flutter application.
2.  **Backend MCP Server Development Track**: Focuses on building the minimal Coordinator and Context Agent MCP Servers required for the MVP, and then expanding to full production.

Dependencies between tracks will be clearly marked.

## Backend MCP Server Development Track

This track focuses on building the backend infrastructure to support the mobile app's core voice interaction and financial intelligence.

### Backend Stage 0: Core Infrastructure & Coordinator Stub (Sprint 1)

**Objective**: Set up foundational backend infrastructure, deploy a minimal Coordinator MCP Server stub with proper MCP protocol implementation and authentication, and establish core performance components. This allows the mobile app to connect and test its communication flow early.

**Key Tasks**:
1.  **Cloud Infrastructure Setup**:
    *   Set up Google Cloud Project, basic networking.
    *   Configure Cloud Run for serverless deployment of the Coordinator stub.
2.  **Coordinator MCP Server (Stub)**:
    *   Develop a basic Coordinator MCP Server using Python/FastAPI with WebSockets.
    *   Implement a WebSocket endpoint (`/ws`) to receive queries from the mobile app, strictly adhering to **MCP JSON-RPC 2.0 protocol** for all incoming messages.
    *   Implement basic logic to return *hardcoded mock text responses* for *any* incoming query (e.g., "Hello from Juno!").
    *   Deploy the Coordinator stub to Cloud Run.
3.  **Authentication & Security (Backend - Initial)**:
    *   Implement **proper Firebase JWT validation** on the Coordinator stub for incoming mobile app requests.
    *   Implement secure credential storage patterns for any sensitive keys.
4.  **Performance & Scalability (Backend - Initial)**:
    *   Set up **Redis for caching and messaging** (e.g., for future inter-agent communication).
    *   Include basic performance monitoring from the start.

**Expected Outcome**:
*   A deployed Coordinator MCP Server stub accessible via WebSocket, enforcing MCP protocol and validating Firebase JWTs.
*   The stub can receive mobile app queries and send back predefined mock text responses.
*   Core performance components (Redis) are in place.

### Backend Stage 1: Minimal Context Agent, Fi's MCP Integration & Resilience (Sprint 2)

**Objective**: Implement a minimal Context Agent that can make a single, simple call to Fi's MCP (e.g., to get a user's current balance) and return this specific data point to the Coordinator. The Coordinator will then forward this to the mobile app, with initial resilience and context management features.

**Key Tasks**:
1.  **Context Agent MCP Server (Minimal)**:
    *   Develop a new MCP server for the Context Agent.
    *   Implement a JSON-RPC 2.0 over WebSocket endpoint to receive requests from the Coordinator, ensuring MCP compliance.
    *   Implement a *single function* (e.g., `get_user_balance`) that, when called, makes a simple internal API call to Fi's MCP Server.
    *   **Fi's MCP Interaction**: Assume Fi's MCP has a simple REST endpoint or internal function (e.g., `/api/v1/users/{user_id}/balance`) that returns a hardcoded or mock balance for a given user ID. The Context Agent will call this.
    *   Return the retrieved balance (or a mock balance) back to the Coordinator.
    *   Deploy the Context Agent to Cloud Run.
2.  **Coordinator Orchestration (Basic with Resilience & Context)**:
    *   Modify the Coordinator to, upon receiving a query from the mobile app, *always* call the `get_user_balance` function on the Context Agent using MCP message routing.
    *   Receive the balance from the Context Agent.
    *   Construct a simple text response (e.g., "Your current balance is $X.") and send it back to the mobile app.
    *   **Authentication & Security (Backend - Inter-MCP)**: Implement **service account authentication** between the Coordinator and Context Agent MCP servers.
    *   **Data Flow & Context Management (Initial)**: Implement a basic **context sharing layer** (e.g., using Redis) and **session state management** in the Coordinator to pass minimal user context to the Context Agent.
    *   **Error Handling & Resilience (Initial)**:
        *   Implement **circuit breaker pattern** for calls to the Context Agent.
        *   Add **graceful degradation** and **fallback response mechanisms** (e.g., return a generic error message if the agent fails).
        *   Include **retry logic with exponential backoff** for agent communication.
    *   **Performance & Scalability (Backend - Continued)**:
        *   Add **connection pooling** for database operations (if Coordinator or Context Agent directly interact with a database, e.g., for logging).

**Expected Outcome**:
*   End-to-end flow for a specific query: Mobile App -> Coordinator -> Context Agent -> Fi's MCP (mock/simple) -> Context Agent -> Coordinator -> Mobile App.
*   The mobile app receives a dynamic (even if simple) financial data point.
*   Initial security, context management, and resilience patterns are implemented in the backend.

### Backend Stage 2: Full Context Agent & Initial Specialized Agents (Sprint 3-5)

**Objective**: Implement the full capabilities of the Context Agent and develop initial versions of the Lifestyle, Security, and Integration Agents, allowing for more complex query processing.

**Key Tasks**:
1.  **Context Agent (Full)**:
    *   Implement environmental analysis (location-based spending patterns).
    *   Implement behavioral pattern recognition (user habit analysis, spending triggers).
    *   Implement temporal context processing (time-of-day financial behavior, seasonal patterns).
    *   Integrate with external data sources (market conditions, economic indicators) via Integration Agent.
2.  **Lifestyle Agent (Initial)**:
    *   Develop a new MCP server for the Lifestyle Agent.
    *   Implement core spending pattern analysis (category-wise expense tracking, trend identification).
    *   Integrate with Fi's MCP for historical data.
3.  **Security Agent (Initial)**:
    *   Develop a new MCP server for the Security Agent.
    *   Implement emergency fund adequacy assessment.
    *   Implement basic spending sustainability checks.
4.  **Integration Agent (Initial)**:
    *   Develop a new MCP server for the Integration Agent.
    *   Implement secure connections to mock/sandbox banking and investment APIs for real-time balance updates and transaction retrieval.
    *   Develop basic data normalization for external data.
5.  **Coordinator Orchestration (Intermediate)**:
    *   Enhance intent recognition to route queries to appropriate agents.
    *   Implement basic parallel processing coordination for Context, Lifestyle, Security, and Integration agents.
    *   Begin implementing decision arbitration logic for simple conflicts.

**Expected Outcome**:
*   Backend can process more complex queries requiring contextual, lifestyle, and security insights.
*   Initial multi-agent orchestration is functional.

### Backend Stage 3: Advanced Agents & Coordinator Logic (Sprint 6-9)

**Objective**: Develop the Growth, Learning, and Risk Assessment Agents, and significantly enhance the Coordinator's decision arbitration and response synthesis capabilities.

**Key Tasks**:
1.  **Growth Agent**: Implement investment opportunity identification, portfolio optimization, and wealth projection modeling.
2.  **Learning Agent**: Develop mechanisms for user feedback processing, decision quality assessment, and initial model refinement based on interaction data.
3.  **Risk Assessment Agent**: Implement portfolio risk analysis, credit risk evaluation, and liquidity risk assessment.
4.  **Coordinator Orchestration (Advanced)**:
    *   Implement full decision arbitration logic with priority weights and conflict resolution matrix.
    *   Develop sophisticated response synthesis from multiple agent inputs, ensuring coherence and personalization.
    *   Implement robust error handling and fallback mechanisms for agent failures.
5.  **Data Layer Enhancements**: Optimize data retrieval from Fi's MCP and caching strategies for all agents.

**Expected Outcome**:
*   All specialized agents are functional, providing comprehensive financial intelligence.
*   Coordinator can intelligently orchestrate and synthesize responses from all agents.

### Backend Stage 4: Scalability, Security & Production Readiness (Sprint 10-12)

**Objective**: Focus on hardening the backend for production, ensuring high availability, scalability, security, and comprehensive monitoring.

**Key Tasks**:
1.  **Scalability**: Implement advanced auto-scaling configurations for all MCP servers on Cloud Run.
    *   Optimize database queries and API calls for performance under load.
    *   Implement multi-layer caching (e.g., Redis) for frequently accessed data.
2.  **Security**: Conduct thorough security audits and penetration testing.
    *   Implement fine-grained IAM policies and service accounts.
    *   Ensure end-to-end encryption for all data at rest and in transit.
    *   Implement API key rotation and encrypted credential storage.
3.  **Monitoring & Logging**: Set up comprehensive monitoring dashboards (e.g., Google Cloud Monitoring, Prometheus/Grafana).
    *   Implement detailed logging for all interactions, errors, and performance metrics.
    *   Configure alerting for critical issues.
4.  **Fault Tolerance & Disaster Recovery**: Implement robust retry mechanisms and graceful degradation.
    *   Establish automated backup and point-in-time recovery for critical data.
5.  **CI/CD Pipeline (Advanced)**: Automate deployment, testing, and rollback procedures for all backend services.

**Expected Outcome**:
*   A highly performant, secure, scalable, and fault-tolerant backend system ready for production deployment.

## Mobile App Development Track

This track focuses on the Flutter application, with clear dependencies on the Backend MCP Server Development Track.

### Mobile Stage 0: Foundation & Setup (Sprint 1)

**Objective**: Establish the core mobile application project, integrate foundational services, and set up basic UI scaffolding, incorporating recommended libraries.

**Key Tasks**:
1.  **Project Initialization**: Create a new Flutter project (`mobile_app/`).
2.  **Basic UI Scaffolding**: Implement a placeholder main screen with a prominent microphone button.
3.  **Firebase Integration**:
    *   Set up Firebase project.
    *   Integrate Firebase Authentication (initial setup, no UI yet).
    *   Integrate Firebase Analytics for basic app usage tracking.
    *   Integrate Firebase Crashlytics for error reporting.
4.  **State Management Setup**: Integrate **Riverpod or Bloc** for robust state management.
5.  **Error Tracking**: Integrate **Sentry** for comprehensive error tracking alongside Crashlytics.
6.  **Version Control**: Initialize Git repository and establish branching strategy.

**Expected Outcome**:
*   Runnable Flutter app with a basic UI and a microphone button.
*   Firebase services initialized and connected.
*   Core state management and error tracking frameworks in place.

### Mobile Stage 1: Core Voice Input & Speech-to-Text (STT) (Sprint 1)

**Objective**: Enable the mobile app to capture user voice input and convert it into text using Google Cloud Speech-to-Text, with optimized audio processing.

**Key Tasks**:
1.  **Microphone Permissions**: Implement runtime permission requests for microphone access.
2.  **Audio Capture & Processing**: Develop functionality to record audio from the device microphone using **flutter_sound** with custom preprocessing (noise reduction, normalization, format conversion) as described in `FLOW.md` (Step 2).
3.  **Google Cloud Speech-to-Text Integration**: Integrate with Google Cloud Speech-to-Text API (streaming recognition).
4.  **Display Transcribed Text**: Show the transcribed text on the UI in real-time as the user speaks.
5.  **Optimize Audio Processing Pipeline**: Focus on efficiency and responsiveness of the audio pipeline.

**Expected Outcome**:
*   User can speak into the app.
*   App displays the transcribed text of the user's speech with optimized audio processing.

### Mobile Stage 2: Backend Communication & TTS (Sprint 2)

**Objective**: Establish secure, real-time communication between the mobile app and the Coordinator MCP Server, sending transcribed queries, and converting received text responses into natural speech for playback.

**Dependency**: Backend Stage 0 (Coordinator MCP Server Stub deployed with proper MCP protocol and JWT validation).

**Key Tasks**:
1.  **Custom MCP-Compliant WebSocket Client**: Implement a **custom MCP-compliant WebSocket client** to connect to the deployed Coordinator MCP Server, ensuring adherence to JSON-RPC 2.0.
2.  **Query Payload Construction**: Format the transcribed text into a simple JSON payload (e.g., `{"jsonrpc": "2.0", "method": "process_query", "params": {"query": "..."}, "id": "..."}`).
3.  **Firebase JWT Token**: Obtain a Firebase JWT token and include it in the WebSocket connection or initial message for authentication.
4.  **Send Query**: Transmit the constructed query payload over the WebSocket connection when the user finishes speaking.
5.  **Receive Raw Response**: Listen for and process raw text responses from the Coordinator MCP Server (these will initially be the hardcoded mock responses from Backend Stage 0).
6.  **Google Cloud Text-to-Speech Integration**: Integrate with Google Cloud Text-to-Speech API.
7.  **Synthesize Speech**: Send the received text response to the TTS API and get back an audio file.
8.  **Audio Playback**: Implement functionality to play the synthesized audio through the device speakers.
9.  **Basic Visual Feedback**: Provide a simple visual indicator (e.g., a "Juno is speaking" message) during audio playback.
10. **WebSocket Reconnection Logic**: Implement robust reconnection logic for the WebSocket client.
11. **Initial Error Handling**: Implement basic error handling for network issues and backend communication failures.

**Expected Outcome**:
*   App successfully sends transcribed queries to the Coordinator stub and receives responses.
*   Juno responds verbally to user queries (using mock responses).
*   Basic conversational flow (user speaks, Juno responds verbally) is established with initial resilience.

### Mobile Stage 3: Display Simple Financial Data (Hackathon MVP) (Sprint 3)

**Objective**: Display the specific, simple financial data point (e.g., account balance) received from the backend, demonstrating a real (though limited) interaction with Fi's MCP.

**Dependency**: Backend Stage 1 (Minimal Context Agent and Coordinator orchestration for Fi's MCP data).

**Key Tasks**:
1.  **Parse Simple Response**: Modify the mobile app to parse the simple text response from the Coordinator (e.g., "Your current balance is $X.").
2.  **Display Data**: Extract the relevant data (e.g., "$X") and display it prominently on the UI, perhaps next to the microphone button or in a dedicated text field.
3.  **Refine TTS Response**: Adjust the TTS playback to clearly state the retrieved financial data.

**Expected Outcome**:
*   The app can successfully query for a specific financial data point (e.g., balance) and display it to the user, demonstrating a minimal end-to-end functional flow.
*   This represents the core demonstrable MVP for the hackathon.

## Post-Hackathon Development: Towards Production Readiness

After the hackathon MVP is complete, we will transition into a more extensive development phase to build out the full vision of Juno, aiming for a production-ready application. This will involve expanding the functionality of both the backend MCP servers and the mobile application.

### Mobile Stage 4: Enhanced User Experience & Rich Data Display (Sprint 4-6)

**Objective**: Implement a visually rich and interactive user interface, capable of displaying complex financial data (charts, graphs) and providing comprehensive visual feedback.

**Dependency**: Backend Stage 2 (Full Context Agent & Initial Specialized Agents providing richer `display_data` and `metadata`).

**Key Tasks**:
1.  **"Hey Juno" Activation (Refined)**: Implement robust wake word detection (if feasible and performant) or a highly intuitive tap-to-speak mechanism.
2.  **Voice Input Visual Feedback (Advanced)**: Develop sophisticated animations and visual cues for active listening, processing, and response generation.
3.  **Rich `display_data` Rendering**: Implement charting libraries (e.g., `fl_chart`, `charts_flutter`) to render spending charts, budget progress, investment performance, etc.
    *   Design and implement detailed financial reports and visualizations within the app.
4.  **Metadata Integration**: Fully integrate and display all relevant `metadata` (confidence scores, data freshness, follow-up suggestions) in a user-friendly manner.
5.  **Error State UI (Comprehensive)**: Design and implement clear, actionable error messages and recovery options for all potential backend and network issues.

**Expected Outcome**:
*   A highly engaging and informative mobile application UI.
*   Users can visualize their financial data through interactive charts and reports.

### Mobile Stage 5: Full Authentication & User Management (Sprint 7-8)

**Objective**: Implement a complete and secure user authentication and profile management system, ensuring personalized and secure access to financial data.

**Dependency**: Backend Stage 4 (Production-ready authentication integration on Coordinator).

**Key Tasks**:
1.  **Login/Signup/Password Reset UI**: Develop comprehensive user interfaces for account creation, login, password reset, and multi-factor authentication (if required).
2.  **User Profile Management**: Implement features for users to view, edit, and manage their personal and financial profile information securely.
3.  **Account Linking**: Provide UI for users to securely link their bank and investment accounts (via Integration Agent).
4.  **Session Management**: Implement robust session management, token refresh, and secure storage of user credentials.
5.  **Personalization Settings**: Allow users to customize their Juno experience (e.g., preferred voice, notification settings, financial goals).

**Expected Outcome**:
*   Users can securely manage their accounts and personalize their app experience.
*   Seamless and secure access to all financial data.

### Mobile Stage 6: Offline Capabilities & Push Notifications (Sprint 9-10)

**Objective**: Enhance app reliability and user engagement through robust offline access to key features and timely, intelligent push notifications.

**Key Tasks**:
1.  **Local Data Persistence**: Implement local databases (e.g., Hive, SQLite) for caching critical financial data (transactions, balances, reports) for offline viewing.
2.  **Offline Query Handling**: Develop logic to queue user queries when offline and process them once connectivity is restored.
3.  **Firebase Cloud Messaging (FCM) Integration**: Fully integrate FCM for targeted push notifications (e.g., spending alerts, budget warnings, personalized insights).
4.  **Notification Management**: Allow users to customize notification preferences within the app.
5.  **Background Sync**: Implement background data synchronization to keep cached data up-to-date.

**Expected Outcome**:
*   App provides a resilient user experience even with intermittent or no connectivity.
*   Users receive timely and relevant financial alerts and insights.

### Mobile Stage 7: Performance, Security, Accessibility & Deployment (Sprint 11-12)

**Objective**: Finalize the application for production deployment, focusing on performance, security, accessibility, and comprehensive testing.

**Key Tasks**:
1.  **Performance Optimization**: Conduct extensive profiling and optimization of the entire mobile application (UI rendering, network calls, audio processing, data handling).
2.  **Security Hardening**: Implement mobile-specific security best practices (e.g., secure data storage, obfuscation, tamper detection).
    *   Conduct mobile application security audits.
3.  **Accessibility**: Implement full accessibility support (e.g., screen reader compatibility, keyboard navigation, high contrast modes, dynamic text sizing) to meet WCAG standards.
4.  **Comprehensive Testing**: Conduct extensive unit, widget, integration, and end-to-end testing.
    *   Perform user acceptance testing (UAT) with a diverse group of beta users.
    *   Conduct cross-device and cross-OS compatibility testing.
5.  **App Store Submission**: Prepare all necessary assets, metadata, and configurations for submission to Google Play Store and Apple App Store.
6.  **Post-Launch Monitoring**: Set up mobile app performance monitoring (e.g., Firebase Performance Monitoring, Crashlytics) and analytics for post-launch insights.

**Expected Outcome**:
*   A high-quality, performant, secure, and accessible mobile application ready for public release.
*   Successful deployment to app stores and ongoing monitoring in production.

## Scrum Master Notes (Updated for Full Development)

*   **Sprint Length**: Recommend 2-week sprints for consistent delivery and feedback in the longer development cycle.
*   **Backlog Refinement**: Regular, detailed backlog refinement sessions are crucial to break down complex features into manageable tasks.
*   **Daily Scrums**: Continue daily stand-ups to track progress, identify impediments, and synchronize efforts across both tracks.
*   **Sprint Reviews**: Conduct sprint reviews at the end of each sprint to demonstrate completed work and gather feedback from stakeholders, especially for cross-track dependencies.
*   **Sprint Retrospectives**: Hold retrospectives to continuously improve team processes, collaboration, and address technical debt.
*   **Definition of Done**: Establish a clear "Definition of Done" for each task and sprint, evolving from MVP to production-grade quality.
*   **Technical Debt**: Proactively manage technical debt, allocating dedicated time in sprints for refactoring, performance improvements, and security enhancements.
*   **Communication**: Maintain transparent and frequent communication with all stakeholders, managing expectations and providing regular updates on progress and challenges.
*   **Risk Management**: Continuously identify, assess, and mitigate risks, especially those related to external integrations and complex AI components.
*   **Quality Assurance**: Integrate QA activities throughout the development lifecycle, not just at the end of stages.
