import 'dart:convert';
import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:uuid/uuid.dart';
import '../models/message.dart';

class WebSocketService {
  // Use environment or fallback to localhost for development
  static String get _baseUrl {
    const String host = String.fromEnvironment('COORDINATOR_HOST', defaultValue: 'localhost');
    const String port = String.fromEnvironment('COORDINATOR_PORT', defaultValue: '8091');
    return 'ws://$host:$port/ws';
  }
  WebSocketChannel? _channel;
  final StreamController<ChatMessage> _messageController = StreamController<ChatMessage>.broadcast();
  final Map<String, Completer<Map<String, dynamic>>> _pendingRequests = {};
  final Uuid _uuid = const Uuid();
  bool _isConnected = false;

  Stream<ChatMessage> get messageStream => _messageController.stream;
  bool get isConnected => _isConnected;

  Future<void> connect() async {
    try {
      _channel = WebSocketChannel.connect(Uri.parse(_baseUrl));
      _isConnected = true;
      
      _channel!.stream.listen(
        _handleMessage,
        onError: (error) {
          debugPrint('WebSocket error: $error');
          _isConnected = false;
          _handleDisconnection();
        },
        onDone: () {
          debugPrint('WebSocket connection closed');
          _isConnected = false;
          _handleDisconnection();
        },
      );
      
      debugPrint('Connected to WebSocket at $_baseUrl');
    } catch (error) {
      debugPrint('Failed to connect to WebSocket: $error');
      _isConnected = false;
      throw Exception('Failed to connect to Juno backend');
    }
  }

  void _handleMessage(dynamic data) {
    try {
      final Map<String, dynamic> message = json.decode(data);
      debugPrint('Received WebSocket message: $message');
      
      // Handle MCP JSON-RPC responses
      if (message.containsKey('id') && message.containsKey('result')) {
        final String id = message['id'];
        
        // Send to UI if it's a chat response
        if (message['result'] is Map && message['result']['response'] != null) {
          final responseText = message['result']['response'];
          
          // Check if this is a login_required response from Fi
          if (_isLoginRequiredResponse(responseText)) {
            _handleLoginRequired(responseText);
          } else {
            final chatMessage = ChatMessage(
              id: _uuid.v4(),
              text: responseText,
              isUser: false,
              timestamp: DateTime.now(),
            );
            _messageController.add(chatMessage);
          }
        }
        
        // Complete the pending request (for any code that might await sendMessage)
        if (_pendingRequests.containsKey(id)) {
          _pendingRequests[id]!.complete(message['result']);
          _pendingRequests.remove(id);
        }
      }
      
      // Handle direct chat messages (for future use)
      if (message.containsKey('text') && message.containsKey('isUser')) {
        final chatMessage = ChatMessage.fromJson(message);
        _messageController.add(chatMessage);
      }
    } catch (error) {
      debugPrint('Error parsing WebSocket message: $error');
    }
  }

  void _handleDisconnection() {
    // Complete pending requests with error
    for (final completer in _pendingRequests.values) {
      completer.completeError('WebSocket disconnected');
    }
    _pendingRequests.clear();
    
    // Optionally attempt reconnection
    _attemptReconnection();
  }

  Future<void> _attemptReconnection() async {
    await Future.delayed(const Duration(seconds: 3));
    if (!_isConnected) {
      try {
        await connect();
      } catch (error) {
        debugPrint('Reconnection failed: $error');
        // Try again after delay
        _attemptReconnection();
      }
    }
  }

  Future<String> sendMessage(String message, String userId, {String? firebaseUID}) async {
    if (!_isConnected || _channel == null) {
      throw Exception('Not connected to backend');
    }

    // Don't add user message to stream here - let ChatProvider handle it
    // to avoid duplicates and ensure proper Firestore persistence

    // Create MCP JSON-RPC message with userId and optional firebaseUID
    final String requestId = _uuid.v4();
    final Map<String, dynamic> params = {
      'query': message,
      'userId': userId,
    };
    
    // Add firebaseUID if provided (Firebase-enabled mode)
    if (firebaseUID != null && firebaseUID.isNotEmpty) {
      params['firebaseUID'] = firebaseUID;
    }
    
    final Map<String, dynamic> mcpMessage = {
      'jsonrpc': '2.0',
      'method': 'process_query',
      'params': params,
      'id': requestId,
    };

    // Set up response handler
    final Completer<Map<String, dynamic>> completer = Completer<Map<String, dynamic>>();
    _pendingRequests[requestId] = completer;

    // Send message
    _channel!.sink.add(json.encode(mcpMessage));
    debugPrint('Sent WebSocket message: $mcpMessage');

    try {
      // Wait for response with timeout
      final response = await completer.future.timeout(
        const Duration(seconds: 30),
        onTimeout: () {
          _pendingRequests.remove(requestId);
          throw Exception('Request timeout');
        },
      );
      
      return response['response'] ?? 'No response received';
    } catch (error) {
      debugPrint('Error getting response: $error');
      rethrow;
    }
  }

  void disconnect() {
    _isConnected = false;
    _channel?.sink.close();
    _channel = null;
    
    // Complete pending requests with error
    for (final completer in _pendingRequests.values) {
      completer.completeError('Connection closed by user');
    }
    _pendingRequests.clear();
  }

  bool _isLoginRequiredResponse(String response) {
    try {
      final decoded = json.decode(response);
      return decoded is Map && decoded['status'] == 'login_required';
    } catch (e) {
      return false;
    }
  }
  
  void _handleLoginRequired(String response) {
    try {
      final decoded = json.decode(response);
      final loginUrl = decoded['login_url'] ?? '';
      final message = decoded['message'] ?? 'Please login to access your financial data.';
      
      debugPrint('Login required. URL: $loginUrl');
      
      // Create a special login_required message for the UI
      final loginMessage = ChatMessage(
        id: _uuid.v4(),
        text: message,
        isUser: false,
        timestamp: DateTime.now(),
        metadata: {
          'type': 'login_required',
          'login_url': loginUrl,
        },
      );
      _messageController.add(loginMessage);
    } catch (e) {
      debugPrint('Error parsing login_required response: $e');
      // Fallback: show the raw response
      final chatMessage = ChatMessage(
        id: _uuid.v4(),
        text: response,
        isUser: false,
        timestamp: DateTime.now(),
      );
      _messageController.add(chatMessage);
    }
  }

  // Send cleanup request for Firebase user logout
  Future<void> cleanupUser(String firebaseUID) async {
    if (!_isConnected || _channel == null) {
      return;
    }

    final String requestId = _uuid.v4();
    final Map<String, dynamic> mcpMessage = {
      'jsonrpc': '2.0',
      'method': 'cleanup_user',
      'params': {
        'firebaseUID': firebaseUID,
      },
      'id': requestId,
    };

    try {
      _channel!.sink.add(json.encode(mcpMessage));
      debugPrint('Sent cleanup request for Firebase user: $firebaseUID');
    } catch (error) {
      debugPrint('Error sending cleanup request: $error');
    }
  }

  void dispose() {
    disconnect();
    _messageController.close();
  }
}