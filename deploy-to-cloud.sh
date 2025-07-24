#!/bin/bash

echo "🚀 Deploying Juno to Google Cloud..."

# Check if ANTHROPIC_API_KEY is set
if [ -z "$ANTHROPIC_API_KEY" ]; then
    echo "❌ Please set your ANTHROPIC_API_KEY:"
    echo "export ANTHROPIC_API_KEY=your_api_key_here"
    exit 1
fi

# Get current project
PROJECT_ID=$(gcloud config get-value project)
echo "📝 Using project: $PROJECT_ID"

# Enable required APIs
echo "🔧 Enabling required APIs..."
gcloud services enable run.googleapis.com
gcloud services enable cloudbuild.googleapis.com
gcloud services enable firebase.googleapis.com

# Deploy fi-mcp-server
echo "🏗️ Deploying fi-mcp-server..."
cd fi-mcp-server
gcloud run deploy fi-mcp-server \
    --source . \
    --platform managed \
    --region us-central1 \
    --allow-unauthenticated \
    --set-env-vars ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
    --memory 512Mi \
    --cpu 1 \
    --max-instances 10

# Get the fi-mcp-server URL
FI_SERVER_URL=$(gcloud run services describe fi-mcp-server --region=us-central1 --format='value(status.url)')
echo "✅ Fi-MCP Server deployed: $FI_SERVER_URL"

# Deploy coordinator-mcp
echo "🏗️ Deploying coordinator-mcp..."
cd ../coordinator-mcp
gcloud run deploy coordinator-mcp \
    --source . \
    --platform managed \
    --region us-central1 \
    --allow-unauthenticated \
    --set-env-vars ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
    --memory 512Mi \
    --cpu 1 \
    --max-instances 10

# Get the coordinator URL
COORDINATOR_URL=$(gcloud run services describe coordinator-mcp --region=us-central1 --format='value(status.url)')
echo "✅ Coordinator deployed: $COORDINATOR_URL"

# Deploy context-agent-mcp
echo "🏗️ Deploying context-agent-mcp..."
cd ../context-agent-mcp
gcloud run deploy context-agent-mcp \
    --source . \
    --platform managed \
    --region us-central1 \
    --allow-unauthenticated \
    --memory 512Mi \
    --cpu 1 \
    --max-instances 10

# Get the context agent URL
CONTEXT_URL=$(gcloud run services describe context-agent-mcp --region=us-central1 --format='value(status.url)')
echo "✅ Context Agent deployed: $CONTEXT_URL"

# Deploy security-agent-mcp
echo "🏗️ Deploying security-agent-mcp..."
cd ../security-agent-mcp
gcloud run deploy security-agent-mcp \
    --source . \
    --platform managed \
    --region us-central1 \
    --allow-unauthenticated \
    --memory 512Mi \
    --cpu 1 \
    --max-instances 10

# Get the security agent URL
SECURITY_URL=$(gcloud run services describe security-agent-mcp --region=us-central1 --format='value(status.url)')
echo "✅ Security Agent deployed: $SECURITY_URL"

echo ""
echo "🎉 All backend services deployed successfully!"
echo ""
echo "📋 Service URLs:"
echo "Fi-MCP Server: $FI_SERVER_URL"
echo "Coordinator: $COORDINATOR_URL"
echo "Context Agent: $CONTEXT_URL"
echo "Security Agent: $SECURITY_URL"
echo ""
echo "Next: Deploy Flutter app to Firebase Hosting"
echo "Run: cd mobile_app && ./deploy-flutter.sh"