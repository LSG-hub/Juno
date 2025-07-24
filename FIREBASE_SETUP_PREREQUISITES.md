# 🔥 Firebase Prerequisites Setup Guide

**Complete Firebase Console Configuration Before Starting Microservices Migration**

## 🎯 **Overview**

Before following the deployment guide, you need to properly configure Firebase console settings, enable services, and set up billing for Firebase Functions.

---

## **Phase 1: Firebase Project Configuration** ⚙️

### **1.1 Verify Your Current Firebase Project**

You already have: `juno-financial-assistant`
- **Project ID**: `juno-financial-assistant`
- **Project Number**: `929051225142`

**Console URL**: https://console.firebase.google.com/project/juno-financial-assistant

### **1.2 Enable Required Firebase Services**

Navigate to your Firebase Console and enable these services:

#### **✅ Already Enabled (From Previous Setup)**
- [x] **Authentication** - Already configured with Email, Google, Anonymous
- [x] **Firestore Database** - Already created with test mode rules  
- [x] **Firebase Hosting** - May need configuration

#### **🆕 Need to Enable Now**

**1. Firebase Functions**
```
Console → Functions → Get Started
```
- Click "Get started" 
- Choose region: `us-central1` (recommended for GCP integration)
- Enable Cloud Functions API
- **Required for**: All backend services (coordinator, fi-mcp, agents)

**2. Firebase Hosting** 
```
Console → Hosting → Get Started
```
- Click "Get started"
- **Required for**: Mobile app deployment

**3. Cloud Storage** (Optional but recommended)
```
Console → Storage → Get Started  
```
- Choose test mode for now
- Select region: `us-central1`
- **Required for**: File uploads, logs, backups

---

## **Phase 2: Billing and Quotas Setup** 💳

### **2.1 Upgrade to Blaze Plan**

⚠️ **CRITICAL**: Firebase Functions require the **Blaze (Pay-as-you-go)** plan

```
Console → Settings (⚙️) → Usage and Billing → Details & Settings
```

**Why Blaze Plan is Required:**
- Firebase Functions can only run on Blaze plan
- Free quotas are generous for development:
  - **125K invocations/month FREE**
  - **40K GB-seconds/month FREE** 
  - **5K GB network/month FREE**

**Cost Estimate for Hackathon:**
- **Development/Testing**: $0-5/month (within free quotas)
- **Demo Day**: $0-1 (light usage)

### **2.2 Set Budget Alerts**

```
Console → Settings → Usage and Billing → Budget Alerts
```

**Recommended Budget:**
- **Alert at**: $10
- **Hard limit**: $25
- **Email**: Your email for notifications

---

## **Phase 3: Firebase CLI Setup** 💻

### **3.1 Install Firebase CLI**

```bash
# Install Firebase CLI globally
npm install -g firebase-tools

# Verify installation
firebase --version
```

### **3.2 Login and Select Project**

```bash
# Login to Firebase (opens browser)
firebase login

# List available projects
firebase projects:list

# Set default project
firebase use juno-financial-assistant
```

### **3.3 Verify CLI Access**

```bash
# Test CLI access
firebase projects:list

# Should show:
# ┌──────────────────────────┬─────────────────────────┬────────────────┬──────────────────────┐
# │ Project Display Name     │ Project ID              │ Project Number │ Resource Location ID │
# ├──────────────────────────┼─────────────────────────┼────────────────┼──────────────────────┤
# │ juno-financial-assistant │ juno-financial-assistant │ 929051225142   │ asia-south1          │
# └──────────────────────────┴─────────────────────────┴────────────────┴──────────────────────┘
```

---

## **Phase 4: Google Cloud Console Setup** ☁️

### **4.1 Enable Required APIs**

Firebase Functions run on Google Cloud Platform. Enable these APIs:

**Navigate to**: https://console.cloud.google.com/apis/library?project=juno-financial-assistant

**Enable the following APIs:**

1. **Cloud Functions API**
   ```
   https://console.cloud.google.com/apis/library/cloudfunctions.googleapis.com
   ```
   - **Required for**: All Firebase Functions
   - Click "Enable"

2. **Cloud Build API**  
   ```
   https://console.cloud.google.com/apis/library/cloudbuild.googleapis.com
   ```
   - **Required for**: Function deployments
   - Click "Enable"

3. **Cloud Logging API**
   ```
   https://console.cloud.google.com/apis/library/logging.googleapis.com  
   ```
   - **Required for**: Function logs and debugging
   - Click "Enable"

4. **Generative Language API** ✅ Already Enabled
   ```
   https://console.cloud.google.com/apis/library/generativelanguage.googleapis.com
   ```
   - **Status**: Already enabled for Gemini integration

### **4.2 Verify API Status**

```bash
# Check enabled APIs
gcloud services list --enabled --project=juno-financial-assistant

# Should include:
# - cloudfunctions.googleapis.com
# - cloudbuild.googleapis.com  
# - logging.googleapis.com
# - generativelanguage.googleapis.com
```

---

## **Phase 5: IAM and Permissions** 🔐

### **5.1 Service Account Setup**

Firebase automatically creates service accounts, but verify permissions:

**Navigate to**: https://console.cloud.google.com/iam-admin/iam?project=juno-financial-assistant

**Verify these service accounts exist:**
1. **Firebase Admin SDK Service Agent**
2. **Google Cloud Functions Service Agent**  
3. **Cloud Build Service Account**

### **5.2 Your User Permissions**

Ensure your account has these roles:
- **Firebase Admin** 
- **Cloud Functions Admin**
- **Cloud Build Editor**
- **Logging Admin**

---

## **Phase 6: Environment Variables & Secrets** 🔑

### **6.1 Firebase Functions Configuration**

Set up secure environment variables for Functions:

```bash
# Set Gemini API Key
firebase functions:config:set gemini.api_key="AIzaSyBvIzIMpPcqUduNF6rSUL2o-ClYWO4GtTA"

# Set service URLs (will be updated after deployment)
firebase functions:config:set \
  services.fi_mcp="https://us-central1-juno-financial-assistant.cloudfunctions.net/fi-mcp" \
  services.context_agent="https://us-central1-juno-financial-assistant.cloudfunctions.net/context-agent" \
  services.security_agent="https://us-central1-juno-financial-assistant.cloudfunctions.net/security-agent"

# Verify configuration
firebase functions:config:get
```

### **6.2 Firestore Security Rules Update**

Since you'll have public Function endpoints, update Firestore rules:

**Navigate to**: https://console.firebase.google.com/project/juno-financial-assistant/firestore/rules

**Update rules** (currently in test mode):
```javascript
rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    // Allow authenticated users to access their own data
    match /users/{userId}/chats/{chatId}/messages/{messageId} {
      allow read, write: if request.auth != null && request.auth.uid == userId;
    }
    
    // Allow access to chat collections
    match /users/{userId}/chats/{document=**} {
      allow read, write: if request.auth != null && request.auth.uid == userId;
    }
  }
}
```

---

## **Phase 7: Pre-Migration Testing** 🧪

### **7.1 Test Firebase CLI Functions**

```bash
# Create test directory
mkdir firebase-test
cd firebase-test

# Initialize Functions
firebase init functions
# Choose:
# - Use existing project: juno-financial-assistant
# - Language: Go
# - Initialize git repo: No

# Test deployment
cd functions
# Create simple test function in main.go:
```

**Test Function (functions/main.go)**:
```go
package main

import (
    "context"
    "fmt"
    "net/http"
    
    "github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
    functions.HTTP("test", testHandler)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello from Juno Firebase Functions!")
}

func main() {
    // This is only used for local testing
}
```

**Deploy Test**:
```bash
# Deploy test function
firebase deploy --only functions:test

# Test the deployed function
curl https://us-central1-juno-financial-assistant.cloudfunctions.net/test

# Should return: "Hello from Juno Firebase Functions!"

# Clean up test
firebase functions:delete test
```

---

## **Phase 8: Final Verification Checklist** ✅

Before starting the migration, verify:

### **8.1 Firebase Console Checklist**
- [ ] **Authentication**: Enabled with Email, Google, Anonymous
- [ ] **Firestore**: Created and accessible  
- [ ] **Functions**: Service enabled
- [ ] **Hosting**: Service enabled
- [ ] **Billing**: Blaze plan active with budget alerts

### **8.2 Google Cloud Console Checklist**  
- [ ] **Cloud Functions API**: Enabled
- [ ] **Cloud Build API**: Enabled
- [ ] **Cloud Logging API**: Enabled
- [ ] **Generative Language API**: Enabled ✅

### **8.3 Local Environment Checklist**
- [ ] **Firebase CLI**: Installed and logged in
- [ ] **Project Selected**: `juno-financial-assistant` 
- [ ] **Test Function**: Successfully deployed and tested
- [ ] **Environment Variables**: Configured

### **8.4 Permissions Checklist**
- [ ] **Your Account**: Has necessary Firebase/GCP roles
- [ ] **Service Accounts**: Auto-created by Firebase
- [ ] **Firestore Rules**: Updated for Function access

---

## **🚀 Ready to Start Migration!**

Once all checkboxes above are complete, you're ready to follow the **DEPLOYMENT_GUIDE.md**!

**Estimated Setup Time**: 30-45 minutes
**One-time Setup**: Yes, these settings persist across all services

**Next Step**: Follow **Phase 1** in `DEPLOYMENT_GUIDE.md` to create repository structure! 🎯