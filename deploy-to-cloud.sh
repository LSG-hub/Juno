#!/bin/bash

echo "🚀 Deploying Juno to Google Cloud..."
export $(grep -v '^#' ./.env | xargs)
echo "Gemini API Key: ${GEMINI_API_KEY}"

# Get current project
PROJECT_ID=$(gcloud config get-value project)
echo "📝 Using project: $PROJECT_ID"

# Enable required APIs
echo "🔧 Enabling required APIs..."
gcloud services enable run.googleapis.com
gcloud services enable cloudbuild.googleapis.com
gcloud services enable firebase.googleapis.com

# Deploy fi-mcp-server FIRST (no dependencies)
echo "🏗️ Deploying fi-mcp-server..."
cd fi-mcp-server
gcloud run deploy fi-mcp-server \
    --source . \
    --platform managed \
    --region us-central1 \
    --allow-unauthenticated \
    --memory 512Mi \
    --cpu 1 \
    --max-instances 10
cd ..

# Get the fi-mcp-server URL
FI_SERVER_URL=$(gcloud run services describe fi-mcp-server --region=us-central1 --format='value(status.url)')
echo "✅ Fi-MCP Server deployed: $FI_SERVER_URL"

# Deploy context-agent-mcp (depends on fi-mcp-server)
echo "🏗️ Deploying context-agent-mcp..."
cd backend/context_agent_mcp
gcloud run deploy context-agent-mcp \
    --source . \
    --platform managed \
    --region us-central1 \
    --allow-unauthenticated \
    --set-env-vars FI_MCP_URL=$FI_SERVER_URL \
    --memory 512Mi \
    --cpu 1 \
    --max-instances 10
cd ../..

# Get the context agent URL
CONTEXT_URL=$(gcloud run services describe context-agent-mcp --region=us-central1 --format='value(status.url)')
echo "✅ Context Agent deployed: $CONTEXT_URL"

# Deploy security-agent-mcp (depends on fi-mcp-server)
echo "🏗️ Deploying security-agent-mcp..."
cd backend/security_agent_mcp
gcloud run deploy security-agent-mcp \
    --source . \
    --platform managed \
    --region us-central1 \
    --allow-unauthenticated \
    --set-env-vars FI_MCP_URL=$FI_SERVER_URL \
    --memory 512Mi \
    --cpu 1 \
    --max-instances 10
cd ../..

# Get the security agent URL
SECURITY_URL=$(gcloud run services describe security-agent-mcp --region=us-central1 --format='value(status.url)')
echo "✅ Security Agent deployed: $SECURITY_URL"

# Deploy coordinator-mcp LAST (depends on all other services)
echo "🏗️ Deploying coordinator-mcp..."
cd backend/coordinator_mcp

# Check if API keys are set
if [ -z "$GEMINI_API_KEY" ]; then
    echo "⚠️  WARNING: No AI API keys found. Set GEMINI_API_KEY "
    echo "The coordinator will deploy but AI responses will not work properly."
fi

gcloud run deploy coordinator-mcp \
    --source . \
    --platform managed \
    --region us-central1 \
    --allow-unauthenticated \
    --set-env-vars "GEMINI_API_KEY=$GEMINI_API_KEY,FI_MCP_URL=$FI_SERVER_URL,CONTEXT_AGENT_URL=$CONTEXT_URL,SECURITY_AGENT_URL=$SECURITY_URL,ENABLE_TRANSLATION=${ENABLE_TRANSLATION:-false},DEFAULT_LANGUAGE=${DEFAULT_LANGUAGE:-en},GOOGLE_TRANSLATE_API_KEY=${GOOGLE_TRANSLATE_API_KEY:-$GOOGLE_API_KEY}" \
    --memory 1Gi \
    --cpu 2 \
    --max-instances 20
cd ../..

# Get the coordinator URL
COORDINATOR_URL=$(gcloud run services describe coordinator-mcp --region=us-central1 --format='value(status.url)')
echo "✅ Coordinator deployed: $COORDINATOR_URL"

echo ""
echo "🎉 All backend services deployed successfully!"
echo ""
echo "📋 Service URLs:"
echo "Fi-MCP Server: $FI_SERVER_URL"
echo "Coordinator: $COORDINATOR_URL"
echo "Context Agent: $CONTEXT_URL"
echo "Security Agent: $SECURITY_URL"
echo ""
echo "📱 Update your Flutter app with the coordinator URL:"
echo "   $COORDINATOR_URL"
echo ""

# Optional: Deploy Flutter web app
read -p "Deploy Flutter app to Firebase Hosting? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🌐 Building Flutter web app..."
    cd mobile_app
    
    # Update the coordinator URL in the Flutter app
    echo "const String coordinatorUrl = '$COORDINATOR_URL';" > lib/config/api_config.dart
    
    # Build Flutter web
    flutter build web --release
    
    # Deploy to Firebase Hosting
    firebase deploy --only hosting
    
    echo "✅ Flutter web app deployed to Firebase Hosting!"
    cd ..
fi

echo ""
echo "🚀 Deployment complete!"