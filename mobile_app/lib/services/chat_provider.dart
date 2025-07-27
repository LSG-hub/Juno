import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:cloud_firestore/cloud_firestore.dart';
import '../models/message.dart';
import 'websocket_service.dart';

class ChatProvider extends ChangeNotifier {
  final WebSocketService _webSocketService = WebSocketService();
  final FirebaseFirestore _firestore = FirebaseFirestore.instance;
  
  final List<ChatMessage> _messages = [];
  bool _isConnecting = false;
  bool _isConnected = false;
  String? _connectionError;
  bool _isTyping = false;
  
  // Per-user chat management
  String? _currentUserId;
  String? _firebaseUID;
  
  // Stream subscription management to prevent duplicates
  StreamSubscription<ChatMessage>? _messageSubscription;

  List<ChatMessage> get messages => List.unmodifiable(_messages);
  bool get isConnecting => _isConnecting;
  bool get isConnected => _isConnected;
  String? get connectionError => _connectionError;
  bool get isTyping => _isTyping;

  ChatProvider() {
    // Initialize automatically
  }

  void initialize() {
    _initializeWebSocket();
  }

  Future<void> _initializeWebSocket() async {
    _isConnecting = true;
    _connectionError = null;
    notifyListeners();

    try {
      // Cancel any existing message subscription to prevent duplicates
      await _messageSubscription?.cancel();
      
      await _webSocketService.connect();
      _isConnected = true;
      _connectionError = null;
      
      // Listen to incoming messages with single subscription
      _messageSubscription = _webSocketService.messageStream.listen(
        (message) {
          _addMessage(message);
        },
        onError: (error) {
          _connectionError = error.toString();
          notifyListeners();
        },
      );

      // Add welcome message
      _addWelcomeMessage();
    } catch (error) {
      _connectionError = error.toString();
      _isConnected = false;
    } finally {
      _isConnecting = false;
      notifyListeners();
    }
  }

  void _addWelcomeMessage() {
    final welcomeMessage = ChatMessage(
      id: 'welcome_${DateTime.now().millisecondsSinceEpoch}',
      text: "Hello! I'm Juno, your AI financial assistant. I can help you access your financial data through Fi Money. How can I help you today?",
      isUser: false,
      timestamp: DateTime.now(),
    );
    _addMessage(welcomeMessage);
  }

  // Switch to different Fi user's chat history
  Future<void> switchToUser(String userId, String firebaseUID) async {
    // Save current user's chat to Firestore before switching (if we have valid data)
    if (_currentUserId != null && _currentUserId != userId && _messages.isNotEmpty && _firebaseUID != null) {
      await _saveCurrentChatToFirestore();
    }
    
    // If Firebase UID changed (different auth method), clear everything
    if (_firebaseUID != null && _firebaseUID != firebaseUID) {
      if (kDebugMode) print('Firebase UID changed from $_firebaseUID to $firebaseUID - clearing chat');
      _messages.clear();
      _currentUserId = null;
    }
    
    _currentUserId = userId;
    _firebaseUID = firebaseUID;
    
    // Load chat history for the selected user
    await _loadChatFromFirestore(userId, firebaseUID);
    
    notifyListeners();
  }

  // Load chat history from Firestore
  Future<void> _loadChatFromFirestore(String userId, String firebaseUID) async {
    try {
      if (kDebugMode) print('Loading chat for user: $userId, firebaseUID: $firebaseUID');
      
      final chatDoc = await _firestore
          .collection('users')
          .doc(firebaseUID)
          .collection('chats')
          .doc(userId)
          .collection('messages')
          .orderBy('timestamp', descending: false)
          .get();

      _messages.clear();
      
      if (chatDoc.docs.isNotEmpty) {
        if (kDebugMode) print('Found ${chatDoc.docs.length} existing messages for user $userId');
        // Load existing messages
        for (var doc in chatDoc.docs) {
          final message = ChatMessage.fromFirestore(doc.data());
          _messages.add(message);
        }
      } else {
        if (kDebugMode) print('No existing messages for user $userId - adding welcome message');
        // First time for this user - add welcome message
        _addWelcomeMessage();
        // Save welcome message to Firestore
        if (_messages.isNotEmpty) {
          await _saveMessageToFirestore(_messages.last);
        }
      }
      
      if (kDebugMode) print('Loaded ${_messages.length} total messages for user $userId');
    } catch (e) {
      if (kDebugMode) print('Error loading chat from Firestore: $e');
      // Fallback to welcome message
      _messages.clear();
      _addWelcomeMessage();
    }
  }


  // Save current chat to Firestore
  Future<void> _saveCurrentChatToFirestore() async {
    if (_currentUserId == null || _firebaseUID == null) return;
    
    try {
      final batch = _firestore.batch();
      final chatRef = _firestore
          .collection('users')
          .doc(_firebaseUID!)
          .collection('chats')
          .doc(_currentUserId!)
          .collection('messages');

      // Save all messages in batch
      for (var message in _messages) {
        final docRef = chatRef.doc(message.id);
        batch.set(docRef, message.toFirestore());
      }

      await batch.commit();
    } catch (e) {
      if (kDebugMode) print('Error saving chat to Firestore: $e');
    }
  }

  // Save individual message to Firestore
  Future<void> _saveMessageToFirestore(ChatMessage message) async {
    if (_currentUserId == null || _firebaseUID == null) return;
    
    try {
      await _firestore
          .collection('users')
          .doc(_firebaseUID!)
          .collection('chats')
          .doc(_currentUserId!)
          .collection('messages')
          .doc(message.id)
          .set(message.toFirestore());
    } catch (e) {
      if (kDebugMode) print('Error saving message to Firestore: $e');
    }
  }

  void _addMessage(ChatMessage message) {
    _messages.add(message);
    // Save message to Firestore automatically
    _saveMessageToFirestore(message);
    notifyListeners();
  }

  Future<void> sendMessage(String text, String userId, {String? firebaseUID}) async {
    if (text.trim().isEmpty || !_isConnected) return;

    try {
      _isTyping = true;
      notifyListeners();

      // Add user message to chat immediately
      final userMessage = ChatMessage(
        id: DateTime.now().millisecondsSinceEpoch.toString(),
        text: text.trim(),
        isUser: true,
        timestamp: DateTime.now(),
      );
      _addMessage(userMessage);

      // Send message through WebSocket service with userId and optional firebaseUID
      await _webSocketService.sendMessage(text.trim(), userId, firebaseUID: firebaseUID);
    } catch (error) {
      // Add error message
      final errorMessage = ChatMessage(
        id: DateTime.now().millisecondsSinceEpoch.toString(),
        text: "Sorry, I'm having trouble processing your request. Please try again.",
        isUser: false,
        timestamp: DateTime.now(),
        status: MessageStatus.error,
      );
      _addMessage(errorMessage);
    } finally {
      _isTyping = false;
      notifyListeners();
    }
  }


  // Retry the last query after login
  Future<void> retryLastQuery() async {
    if (!_isConnected) return;
    
    try {
      _isTyping = true;
      notifyListeners();
      
      // Retry the last query through WebSocket service
      await _webSocketService.retryLastQuery();
    } catch (error) {
      // Add error message
      final errorMessage = ChatMessage(
        id: DateTime.now().millisecondsSinceEpoch.toString(),
        text: "Sorry, I'm having trouble processing your request. Please try again.",
        isUser: false,
        timestamp: DateTime.now(),
        status: MessageStatus.error,
      );
      _addMessage(errorMessage);
    } finally {
      _isTyping = false;
      notifyListeners();
    }
  }

  // Cleanup method for Firebase user logout
  Future<void> cleanupUser(String firebaseUID, {bool isAnonymous = false}) async {
    await _webSocketService.cleanupUser(firebaseUID);
    
    // For anonymous users, completely delete their data from Firestore (pure test mode)
    if (isAnonymous) {
      await _deleteAnonymousUserData(firebaseUID);
    }
  }

  // Delete all anonymous user data from Firestore (called on anonymous sign out)
  Future<void> _deleteAnonymousUserData(String firebaseUID) async {
    try {
      if (kDebugMode) print('Deleting anonymous user data for: $firebaseUID');
      
      // Delete entire user document and all subcollections
      final userRef = _firestore.collection('users').doc(firebaseUID);
      
      // First, delete all chats and their messages
      final chatsSnapshot = await userRef.collection('chats').get();
      final batch = _firestore.batch();
      
      for (var chatDoc in chatsSnapshot.docs) {
        // Delete all messages in this chat
        final messagesSnapshot = await chatDoc.reference.collection('messages').get();
        for (var messageDoc in messagesSnapshot.docs) {
          batch.delete(messageDoc.reference);
        }
        // Delete the chat document itself
        batch.delete(chatDoc.reference);
      }
      
      // Delete the user document
      batch.delete(userRef);
      
      await batch.commit();
      
      if (kDebugMode) print('Successfully deleted anonymous user data');
    } catch (e) {
      if (kDebugMode) print('Error deleting anonymous user data: $e');
    }
  }

  Future<void> reconnect() async {
    await _initializeWebSocket();
  }

  // Clear current user's chat only
  Future<void> clearCurrentUserChat() async {
    if (_currentUserId == null || _firebaseUID == null) return;
    
    try {
      // Delete from Firestore
      final messagesRef = _firestore
          .collection('users')
          .doc(_firebaseUID!)
          .collection('chats')
          .doc(_currentUserId!)
          .collection('messages');
      
      final snapshot = await messagesRef.get();
      final batch = _firestore.batch();
      
      for (var doc in snapshot.docs) {
        batch.delete(doc.reference);
      }
      
      await batch.commit();
      
      // Clear local messages and add welcome
      _messages.clear();
      _addWelcomeMessage();
      notifyListeners();
    } catch (e) {
      if (kDebugMode) print('Error clearing current user chat: $e');
      // Fallback to local clear
      _messages.clear();
      _addWelcomeMessage();
      notifyListeners();
    }
  }

  // Clear ALL users' chats (for fresh start)
  Future<void> clearAllUsersChats() async {
    if (_firebaseUID == null) return;
    
    try {
      // Delete entire chats collection for this Firebase user
      final chatsRef = _firestore
          .collection('users')
          .doc(_firebaseUID!)
          .collection('chats');
      
      final snapshot = await chatsRef.get();
      final batch = _firestore.batch();
      
      for (var chatDoc in snapshot.docs) {
        // Delete all messages in each chat
        final messagesSnapshot = await chatDoc.reference.collection('messages').get();
        for (var messageDoc in messagesSnapshot.docs) {
          batch.delete(messageDoc.reference);
        }
        // Delete the chat document itself
        batch.delete(chatDoc.reference);
      }
      
      await batch.commit();
      
      // Clear local messages and add welcome
      _messages.clear();
      _addWelcomeMessage();
      notifyListeners();
    } catch (e) {
      if (kDebugMode) print('Error clearing all users chats: $e');
      // Fallback to local clear
      _messages.clear();
      _addWelcomeMessage();
      notifyListeners();
    }
  }

  // Reset chat provider when auth changes (new login)
  void resetForNewAuth() {
    _messages.clear();
    _currentUserId = null;
    _firebaseUID = null;
    notifyListeners();
  }

  // Legacy method for backward compatibility
  void clearMessages() {
    clearCurrentUserChat();
  }

  @override
  void dispose() {
    _messageSubscription?.cancel();
    _webSocketService.dispose();
    super.dispose();
  }
}