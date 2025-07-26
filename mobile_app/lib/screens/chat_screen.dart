// lib/screens/chat_screen.dart

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/chat_provider.dart';
import '../services/auth_service.dart';
import '../services/voice_service.dart';
import '../widgets/message_widget.dart';
import '../widgets/typing_indicator.dart';
import '../widgets/user_selector_widget.dart';

class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key});

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _textController = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  final FocusNode _focusNode = FocusNode();
  final WebVoiceService _voiceService = WebVoiceService();
  String _selectedUserId = '1111111111'; // Default test user

  @override
  void initState() {
    super.initState();
    
    // Initialize voice service
    _initializeVoice();
    
    // Initialize chat
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final chatProvider = context.read<ChatProvider>();
      final authService = context.read<AuthService>();
      
      chatProvider.resetForNewAuth();
      chatProvider.initialize();
      
      if (authService.firebaseUID != null) {
        chatProvider.switchToUser(_selectedUserId, authService.firebaseUID!);
      }
    });
  }

  Future<void> _initializeVoice() async {
    final success = await _voiceService.initialize();
    debugPrint('Voice service initialized: $success');
    
    if (!success && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Voice input is available in Chrome and Edge browsers'),
          duration: Duration(seconds: 3),
        ),
      );
    }
  }

  @override
  void dispose() {
    _textController.dispose();
    _scrollController.dispose();
    _focusNode.dispose();
    _voiceService.dispose();
    super.dispose();
  }

  void _sendMessage() {
    final text = _textController.text.trim();
    if (text.isNotEmpty) {
      final authService = context.read<AuthService>();
      context.read<ChatProvider>().sendMessage(
        text, 
        _selectedUserId,
        firebaseUID: authService.firebaseUID,
      );
      _textController.clear();
      _scrollToBottom();
    }
  }

  void _onUserChanged(String userId) async {
    final authService = context.read<AuthService>();
    final chatProvider = context.read<ChatProvider>();
    
    setState(() {
      _selectedUserId = userId;
    });
    
    await chatProvider.switchToUser(userId, authService.firebaseUID ?? '');
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 300),
          curve: Curves.easeOut,
        );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    
    return Scaffold(
      backgroundColor: theme.colorScheme.surfaceVariant,
      appBar: AppBar(
        title: const Text('Juno - Web'),
        backgroundColor: theme.colorScheme.inversePrimary,
        actions: [
          UserSelectorWidget(
            selectedUserId: _selectedUserId,
            onUserChanged: _onUserChanged,
          ),
          const SizedBox(width: 8),
          Consumer<ChatProvider>(
            builder: (context, chatProvider, child) {
              return Row(
                children: [
                  Container(
                    width: 8,
                    height: 8,
                    margin: const EdgeInsets.only(right: 8),
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: chatProvider.isConnected
                          ? Colors.green
                          : Colors.red,
                    ),
                  ),
                  Text(
                    chatProvider.isConnected ? 'Connected' : 'Disconnected',
                    style: theme.textTheme.bodySmall,
                  ),
                  const SizedBox(width: 16),
                  if (chatProvider.messages.isNotEmpty)
                    IconButton(
                      icon: const Icon(Icons.clear_all),
                      onPressed: () => chatProvider.clearMessages(),
                      tooltip: 'Clear chat',
                    ),
                ],
              );
            },
          ),
        ],
        surfaceTintColor: theme.colorScheme.surfaceTint,
        elevation: 0,
      ),
      body: Column(
        children: [
          // Voice status indicator
          if (_voiceService.isListening)
            Container(
              padding: const EdgeInsets.all(12),
              color: theme.colorScheme.primaryContainer,
              child: Row(
                children: [
                  Icon(Icons.mic, color: theme.colorScheme.primary),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      _voiceService.currentTranscript.isEmpty 
                          ? 'Listening...' 
                          : _voiceService.currentTranscript,
                      style: TextStyle(color: theme.colorScheme.onPrimaryContainer),
                    ),
                  ),
                ],
              ),
            ),
          
          Expanded(
            child: Consumer<ChatProvider>(
              builder: (context, chatProvider, child) {
                if (chatProvider.connectionError != null) {
                  return _buildErrorState(chatProvider);
                }
                
                return _buildChatList(chatProvider);
              },
            ),
          ),
          
          _buildInputArea(),
        ],
      ),
    );
  }

  Widget _buildErrorState(ChatProvider chatProvider) {
    final theme = Theme.of(context);
    
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              Icons.cloud_off,
              size: 64,
              color: theme.colorScheme.error,
            ),
            const SizedBox(height: 16),
            Text(
              'Connection Error',
              style: theme.textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            Text(
              chatProvider.connectionError!,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 24),
            FilledButton.icon(
              onPressed: () => chatProvider.reconnect(),
              icon: const Icon(Icons.refresh),
              label: const Text('Try Again'),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildChatList(ChatProvider chatProvider) {
    return ListView.builder(
      controller: _scrollController,
      padding: const EdgeInsets.symmetric(vertical: 16),
      itemCount: chatProvider.messages.length + (chatProvider.isTyping ? 1 : 0),
      itemBuilder: (context, index) {
        if (index == chatProvider.messages.length && chatProvider.isTyping) {
          return const TypingIndicator();
        }
        
        final message = chatProvider.messages[index];
        final showTimestamp = index == 0 ||
            chatProvider.messages[index - 1].timestamp
                .difference(message.timestamp)
                .inMinutes
                .abs() > 5;
        
        return MessageWidget(
          message: message,
          showTimestamp: showTimestamp,
        );
      },
    );
  }

  Widget _buildInputArea() {
    final theme = Theme.of(context);
    
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        boxShadow: [
          BoxShadow(
            color: theme.shadowColor.withOpacity(0.1),
            blurRadius: 8,
            offset: const Offset(0, -2),
          ),
        ],
      ),
      child: SafeArea(
        child: Consumer<ChatProvider>(
          builder: (context, chatProvider, child) {
            return Row(
              children: [
                // Voice button with state management
                ListenableBuilder(
                  listenable: _voiceService,
                  builder: (context, child) {
                    return Container(
                      decoration: BoxDecoration(
                        color: _voiceService.isListening
                            ? theme.colorScheme.errorContainer
                            : _voiceService.isAvailable
                                ? theme.colorScheme.primaryContainer
                                : theme.colorScheme.surfaceVariant,
                        shape: BoxShape.circle,
                      ),
                      child: IconButton(
                        onPressed: chatProvider.isConnected
                            ? () {
                                if (_voiceService.isAvailable) {
                                  _voiceService.toggleListening(
                                    onResult: (text) {
                                      _textController.text = text;
                                      _sendMessage();
                                    },
                                  );
                                } else {
                                  ScaffoldMessenger.of(context).showSnackBar(
                                    const SnackBar(
                                      content: Text('Voice input requires Chrome or Edge browser'),
                                    ),
                                  );
                                }
                              }
                            : null,
                        icon: Icon(
                          _voiceService.isListening 
                              ? Icons.stop 
                              : _voiceService.isAvailable 
                                  ? Icons.mic 
                                  : Icons.mic_off,
                          color: _voiceService.isListening
                              ? theme.colorScheme.error
                              : _voiceService.isAvailable
                                  ? theme.colorScheme.primary
                                  : theme.colorScheme.onSurfaceVariant,
                        ),
                        tooltip: !_voiceService.isAvailable
                            ? 'Voice input requires Chrome or Edge'
                            : _voiceService.isListening
                                ? 'Stop listening'
                                : 'Start voice input',
                      ),
                    );
                  },
                ),
                
                const SizedBox(width: 12),
                
                // Text input field
                Expanded(
                  child: TextField(
                    controller: _textController,
                    focusNode: _focusNode,
                    enabled: chatProvider.isConnected,
                    decoration: InputDecoration(
                      hintText: chatProvider.isConnected
                          ? 'Ask me about your finances...'
                          : 'Connecting to Juno...',
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(24),
                        borderSide: BorderSide.none,
                      ),
                      filled: true,
                      fillColor: theme.colorScheme.surfaceContainerHighest,
                      contentPadding: const EdgeInsets.symmetric(
                        horizontal: 20,
                        vertical: 12,
                      ),
                      suffixIcon: _textController.text.isNotEmpty
                          ? IconButton(
                              onPressed: () {
                                _textController.clear();
                                setState(() {});
                              },
                              icon: const Icon(Icons.clear),
                            )
                          : null,
                    ),
                    textInputAction: TextInputAction.send,
                    onSubmitted: chatProvider.isConnected ? (_) => _sendMessage() : null,
                    onChanged: (text) {
                      setState(() {}); // Rebuild to show/hide clear button
                    },
                    maxLines: null,
                    textCapitalization: TextCapitalization.sentences,
                  ),
                ),
                
                const SizedBox(width: 8),
                
                // Send button
                ValueListenableBuilder<TextEditingValue>(
                  valueListenable: _textController,
                  builder: (context, value, child) {
                    final hasText = value.text.trim().isNotEmpty;
                    return AnimatedContainer(
                      duration: const Duration(milliseconds: 200),
                      child: FloatingActionButton(
                        onPressed: hasText && chatProvider.isConnected
                            ? _sendMessage
                            : null,
                        backgroundColor: hasText && chatProvider.isConnected
                            ? theme.colorScheme.primary
                            : theme.colorScheme.surfaceContainerHighest,
                        foregroundColor: hasText && chatProvider.isConnected
                            ? theme.colorScheme.onPrimary
                            : theme.colorScheme.onSurfaceVariant,
                        mini: true,
                        child: chatProvider.isTyping
                            ? SizedBox(
                                width: 16,
                                height: 16,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                  valueColor: AlwaysStoppedAnimation<Color>(
                                    theme.colorScheme.onPrimary,
                                  ),
                                ),
                              )
                            : const Icon(Icons.send),
                      ),
                    );
                  },
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}