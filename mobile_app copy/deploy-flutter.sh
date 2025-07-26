#!/bin/bash

echo "📱 Deploying Flutter app to Firebase Hosting..."

# Get current project
PROJECT_ID=$(gcloud config get-value project)
echo "📝 Using project: $PROJECT_ID"

# Get Cloud Run service URLs
echo "🔍 Getting backend service URLs..."
COORDINATOR_URL=$(gcloud run services describe coordinator-mcp --region=us-central1 --format='value(status.url)' 2>/dev/null)

if [ -z "$COORDINATOR_URL" ]; then
    echo "❌ Coordinator service not found. Please deploy backend services first."
    echo "Run: ./deploy-to-cloud.sh"
    exit 1
fi

echo "✅ Coordinator URL: $COORDINATOR_URL"

# Update Firebase project 
echo "🔥 Setting up Firebase..."
firebase use $PROJECT_ID --add

#cleaning previous dependencies
flutter clean

# Install dependencies
echo "📦 Installing Flutter dependencies..."
flutter pub get

# Build Flutter web app with production backend URLs
echo "🏗️ Building Flutter web app..."
COORDINATOR_HOST=$(echo $COORDINATOR_URL | sed 's|https://||' | sed 's|http://||')
flutter build web --release \
    --dart-define=COORDINATOR_HOST=$COORDINATOR_HOST \
    --dart-define=COORDINATOR_PORT=443 \
    --dart-define=USE_HTTPS=true

# Deploy to Firebase Hosting
echo "🚀 Deploying to Firebase Hosting..."
firebase deploy --only hosting

# Get the hosting URL
HOSTING_URL="https://$PROJECT_ID.web.app"
echo "🌐 Your app is live at: $HOSTING_URL"
echo "✅ Your Juno AI Assistant is now live!"