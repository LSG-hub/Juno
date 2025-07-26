import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/chat_provider.dart';
import '../services/auth_service.dart';
import '../services/voice_service.dart';
import '../widgets/message_widget.dart';
import '../widgets/typing_indicator.dart';
import '../widgets/user_selector_widget.dart';
import '../widgets/voice_button.dart';

class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key});

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _textController = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  final FocusNode _focusNode = FocusNode();
  final VoiceService _voiceService = VoiceService();
  String _selectedUserId = '1111111111'; // Default test user

  // All 16 test phone numbers from Fi MCP server
  static const List<String> _testUsers = [
    '1010101010',
    '1111111111',
    '1212121212',
    '1313131313',
    '1414141414',
    '2020202020',
    '2121212121',
    '2222222222',
    '2525252525',
    '3333333333',
    '4444444444',
    '5555555555',
    '6666666666',
    '7777777777',
    '8888888888',
    '9999999999',
  ];

  @override
  void initState() {
    super.initState();

    // Initialize voice service for web
    _initializeVoice();

    // Initialize chat provider and reset for new auth session
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final chatProvider = context.read<ChatProvider>();
      final authService = context.read<AuthService>();
      
      // Reset chat for clean session
      chatProvider.resetForNewAuth();
      chatProvider.initialize();
      
      // Initialize with current Firebase user and default Fi user
      if (authService.firebaseUID != null) {
        chatProvider.switchToUser(_selectedUserId, authService.firebaseUID!);
      }

      _textController.addListener(() => setState(() {}));
    });
  }

  Future<void> _initializeVoice() async {
    final success = await _voiceService.initialize();
    debugPrint('Voice service initialized: $success');

    if (!success && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Voice input requires Chrome/Edge browser with HTTPS'),
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

  void _onVoiceResult(String text) {
    if (text.isNotEmpty) {
      _textController.text = text;
      _sendMessage();
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
      appBar: AppBar(
        title: Row(
          children: [
            Container(
              width: 40,
              height: 40,
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  colors: [
                    theme.colorScheme.primary,
                    theme.colorScheme.primaryContainer,
                  ],
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                ),
                shape: BoxShape.circle,
              ),
              child: const Icon(
                Icons.psychology,
                color: Colors.white,
                size: 24,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Juno',
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  Consumer2<ChatProvider, AuthService>(
                    builder: (context, chatProvider, authService, child) {
                      String subtitle = '';
                      Color color = theme.colorScheme.error;
                      
                      if (chatProvider.isConnected) {
                        subtitle = '${authService.displayName} • AI Financial Assistant';
                        color = theme.colorScheme.primary;
                      } else if (chatProvider.isConnecting) {
                        subtitle = 'Connecting...';
                        color = theme.colorScheme.onSurfaceVariant;
                      } else {
                        subtitle = 'Offline';
                        color = theme.colorScheme.error;
                      }
                      
                      return Text(
                        subtitle,
                        style: theme.textTheme.bodySmall?.copyWith(color: color),
                        overflow: TextOverflow.ellipsis,
                      );
                    },
                  ),
                ],
              ),
            ),
            // User selector widget
            UserSelectorWidget(
              selectedUserId: _selectedUserId,
              onUserChanged: _onUserChanged,
            ),
            const SizedBox(width: 8),
          ],
        ),
        actions: [
          Consumer2<ChatProvider, AuthService>(
            builder: (context, chatProvider, authService, child) {
              return PopupMenuButton<String>(
                onSelected: (value) async {
                  switch (value) {
                    case 'clear':
                      await chatProvider.clearCurrentUserChat();
                      break;
                    case 'clear_all':
                      await chatProvider.clearAllUsersChats();
                      break;
                    case 'reconnect':
                      chatProvider.reconnect();
                      break;
                    case 'logout':
                      // Cleanup Fi clients for this Firebase user
                      if (authService.firebaseUID != null) {
                        await chatProvider.cleanupUser(
                          authService.firebaseUID!, 
                          isAnonymous: authService.isAnonymous,
                        );
                      }
                      // Sign out from Firebase
                      await authService.signOut();
                      break;
                  }
                },
                itemBuilder: (context) => [
                  const PopupMenuItem(
                    value: 'clear',
                    child: Row(
                      children: [
                        Icon(Icons.clear_all),
                        SizedBox(width: 8),
                        Text('Clear Chat'),
                      ],
                    ),
                  ),
                  const PopupMenuItem(
                    value: 'clear_all',
                    child: Row(
                      children: [
                        Icon(Icons.delete_sweep),
                        SizedBox(width: 8),
                        Text('Clear All Chats'),
                      ],
                    ),
                  ),
                  if (!chatProvider.isConnected)
                    const PopupMenuItem(
                      value: 'reconnect',
                      child: Row(
                        children: [
                          Icon(Icons.refresh),
                          SizedBox(width: 8),
                          Text('Reconnect'),
                        ],
                      ),
                    ),
                  const PopupMenuItem(
                    value: 'logout',
                    child: Row(
                      children: [
                        Icon(Icons.logout),
                        SizedBox(width: 8),
                        Text('Sign Out'),
                      ],
                    ),
                  ),
                ],
              );
            },
          ),
        ],
        backgroundColor: theme.colorScheme.surface,
        surfaceTintColor: theme.colorScheme.surfaceTint,
        elevation: 0,
      ),
      body: Container(
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [
              const Color(0xFF1A4A3A), // Deep green
              const Color(0xFF1A1A4A), // Deep blue
            ],
          ),
        ),
        child: Column(
          children: [
            // Voice status indicator
            ListenableBuilder(
              listenable: _voiceService,
              builder: (context, child) {
                if (!_voiceService.isListening) return const SizedBox.shrink();

                return Container(
                  padding: const EdgeInsets.all(16),
                  color: const Color(0xFF00C896).withOpacity(0.1),
                  child: Row(
                    children: [
                      Container(
                        width: 32,
                        height: 32,
                        decoration: BoxDecoration(
                          color: const Color(0xFF00C896),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: const Icon(
                          Icons.mic,
                          color: Colors.white,
                          size: 16,
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          _voiceService.currentTranscript.isEmpty
                              ? 'Listening...'
                              : _voiceService.currentTranscript,
                          style: TextStyle(
                            color: Colors.white,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                      ),
                    ],
                  ),
                );
              },
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
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [
            Colors.transparent,
            const Color(0xFF1A4A3A).withOpacity(0.9),
          ],
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.2),
            blurRadius: 10,
            offset: const Offset(0, -2),
          ),
        ],
      ),
      child: SafeArea(
        child: Consumer<ChatProvider>(
          builder: (context, chatProvider, child) {
            return Row(
              children: [
                // Voice button
                ChangeNotifierProvider.value(
                  value: _voiceService,
                  child: VoiceButton(
                    onTextReceived: _onVoiceResult,
                    isEnabled: chatProvider.isConnected,
                  ),
                ),
                const SizedBox(width: 12),

                // Text input
                Expanded(
                  child: Container(
                    decoration: BoxDecoration(
                      color: const Color(0xFF2A2A2A).withOpacity(0.8),
                      borderRadius: BorderRadius.circular(24),
                      border: Border.all(
                        color: const Color(0xFF404040),
                        width: 1,
                      ),
                    ),
                    child: TextField(
                      controller: _textController,
                      focusNode: _focusNode,
                      enabled: chatProvider.isConnected,
                      decoration: InputDecoration(
                        hintText: chatProvider.isConnected
                            ? 'Ask Juno about your finances...'
                            : 'Connecting...',
                        hintStyle: TextStyle(
                          color: const Color(0xFFB0B0B0),
                        ),
                        border: InputBorder.none,
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
                                icon: Icon(
                                  Icons.clear_rounded,
                                  color: const Color(0xFFB0B0B0),
                                  size: 20,
                                ),
                              )
                            : null,
                      ),
                      style: const TextStyle(color: Colors.white),
                      onSubmitted: (_) => _sendMessage(),
                      maxLines: null,
                      textCapitalization: TextCapitalization.sentences,
                    ),
                  ),
                ),
                const SizedBox(width: 12),

                // Send button
                Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    gradient: LinearGradient(
                      colors: _textController.text.trim().isNotEmpty && chatProvider.isConnected
                          ? [const Color(0xFF00C896), const Color(0xFF6366F1)]
                          : [Colors.grey.shade300, Colors.grey.shade400],
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                    ),
                    borderRadius: BorderRadius.circular(24),
                  ),
                  child: Material(
                    color: Colors.transparent,
                    child: InkWell(
                      borderRadius: BorderRadius.circular(24),
                      onTap: _textController.text.trim().isNotEmpty && chatProvider.isConnected
                          ? _sendMessage
                          : null,
                      child: const Icon(
                        Icons.send_rounded,
                        color: Colors.white,
                        size: 20,
                      ),
                    ),
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}