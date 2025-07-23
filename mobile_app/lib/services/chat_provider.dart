import 'package:flutter/foundation.dart';
import '../models/message.dart';
import 'websocket_service.dart';

class ChatProvider extends ChangeNotifier {
  final WebSocketService _webSocketService = WebSocketService();
  final List<ChatMessage> _messages = [];
  bool _isConnecting = false;
  bool _isConnected = false;
  String? _connectionError;
  bool _isTyping = false;

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
      await _webSocketService.connect();
      _isConnected = true;
      _connectionError = null;
      
      // Listen to incoming messages
      _webSocketService.messageStream.listen(
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
      id: 'welcome',
      text: "Hello! I'm Juno, your AI financial assistant. I can help you access your financial data through Fi Money. How can I help you today?",
      isUser: false,
      timestamp: DateTime.now(),
    );
    _addMessage(welcomeMessage);
  }

  void _addMessage(ChatMessage message) {
    _messages.add(message);
    notifyListeners();
  }

  Future<void> sendMessage(String text, String userId, {String? firebaseUID}) async {
    if (text.trim().isEmpty || !_isConnected) return;

    try {
      _isTyping = true;
      notifyListeners();

      // Send message through WebSocket service with userId and optional firebaseUID
      // The service will add the user message and handle the response
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

  // Cleanup method for Firebase user logout
  Future<void> cleanupUser(String firebaseUID) async {
    await _webSocketService.cleanupUser(firebaseUID);
  }

  Future<void> reconnect() async {
    await _initializeWebSocket();
  }

  void clearMessages() {
    _messages.clear();
    _addWelcomeMessage();
    notifyListeners();
  }

  @override
  void dispose() {
    _webSocketService.dispose();
    super.dispose();
  }
}