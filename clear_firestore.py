#!/usr/bin/env python3
"""
Script to clear all data from Juno Firestore database
"""

import firebase_admin
from firebase_admin import credentials, firestore
import sys

def clear_firestore():
    try:
        # Initialize Firebase Admin SDK for the specific project
        # Using project ID from mobile app firebase_options.dart
        
        # Try to initialize with project-specific credentials
        try:
            cred = credentials.ApplicationDefault()
            app = firebase_admin.initialize_app(cred, {
                'projectId': 'juno-financial-assistant'
            })
            print("✅ Initialized Firebase with project credentials for juno-financial-assistant")
        except Exception as e:
            print(f"❌ Failed to initialize Firebase: {e}")
            print("Please ensure you're authenticated with gcloud: gcloud auth application-default login")
            return False
        
        # Initialize Firestore client
        db = firestore.client()
        print("✅ Connected to Firestore")
        
        # Get all collections
        collections = db.collections()
        
        total_deleted = 0
        
        for collection in collections:
            collection_name = collection.id
            print(f"\n🗑️  Processing collection: {collection_name}")
            
            # Get all documents in this collection
            docs = collection.stream()
            
            batch_size = 100
            batch = db.batch()
            count_in_batch = 0
            collection_count = 0
            
            for doc in docs:
                # Add to batch for deletion
                batch.delete(doc.reference)
                count_in_batch += 1
                collection_count += 1
                
                # Execute batch when it reaches the limit
                if count_in_batch >= batch_size:
                    batch.commit()
                    print(f"   Deleted batch of {count_in_batch} documents")
                    batch = db.batch()
                    count_in_batch = 0
            
            # Commit remaining documents in batch
            if count_in_batch > 0:
                batch.commit()
                print(f"   Deleted final batch of {count_in_batch} documents")
            
            print(f"✅ Deleted {collection_count} documents from {collection_name}")
            total_deleted += collection_count
        
        print(f"\n🎉 Successfully cleared Firestore database!")
        print(f"📊 Total documents deleted: {total_deleted}")
        return True
        
    except Exception as e:
        print(f"❌ Error clearing Firestore: {e}")
        return False

if __name__ == "__main__":
    print("🧹 Clearing Juno Firestore Database...")
    print("⚠️  This will delete ALL data from the database!")
    print("🔄 Auto-proceeding to clear database for testing...")
    
    success = clear_firestore()
    if success:
        print("✅ Database cleared successfully!")
    else:
        print("❌ Failed to clear database")
        sys.exit(1)