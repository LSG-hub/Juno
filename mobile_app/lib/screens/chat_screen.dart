// lib/screens/chat_screen.dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/chat_provider.dart';
import '../services/auth_service.dart';
import '../services/voice_service.dart';
import '../widgets/message_widget.dart';
import '../widgets/typing_indicator.dart';
import '../widgets/voice-button.dart';

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

    // Initialize chat
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final chatProvider = context.read<ChatProvider>();
      final authService = context.read<AuthService>();

      chatProvider.resetForNewAuth();
      chatProvider.initialize();

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

  Future<void> _clearAllChats() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Clear All Chats'),
        content: const Text('Are you sure you want to clear all chat history? This action cannot be undone.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () => Navigator.of(context).pop(true),
            style: ElevatedButton.styleFrom(
              backgroundColor: Colors.red,
              foregroundColor: Colors.white,
            ),
            child: const Text('Clear All'),
          ),
        ],
      ),
    );

    if (confirmed == true && mounted) {
      await context.read<ChatProvider>().clearAllUsersChats();
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('All chats cleared successfully'),
          backgroundColor: Color(0xFF00C896),
        ),
      );
    }
  }

  Future<void> _signOut() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Sign Out'),
        content: const Text('Are you sure you want to sign out?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('Sign Out'),
          ),
        ],
      ),
    );

    if (confirmed == true && mounted) {
      await context.read<AuthService>().signOut();
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final authService = context.watch<AuthService>();

    return Scaffold(
      appBar: AppBar(
        title: Row(
          children: [
            Container(
              width: 32,
              height: 32,
              decoration: BoxDecoration(
                gradient: const LinearGradient(
                  colors: [Color(0xFF00C896), Color(0xFF6366F1)],
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                ),
                borderRadius: BorderRadius.circular(8),
              ),
              child: const Icon(
                Icons.psychology_rounded,
                color: Colors.white,
                size: 20,
              ),
            ),
            const SizedBox(width: 12),
            Text(
              'Juno',
              style: theme.textTheme.titleLarge?.copyWith(
                fontWeight: FontWeight.w700,
                color: Colors.white,
              ),
            ),
          ],
        ),
        backgroundColor: const Color(0xFF1A1A2E), // Dark background color
        elevation: 0,
        surfaceTintColor: Colors.transparent,
        leading: _buildMenuDropdown(context, authService),
        actions: [
          // User selector dropdown
          _buildUserSelector(),
          const SizedBox(width: 8),

          // Connection status
          Consumer<ChatProvider>(
            builder: (context, chatProvider, child) {
              return Container(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                decoration: BoxDecoration(
                  color: chatProvider.isConnected
                      ? const Color(0xFF00C896).withOpacity(0.1)
                      : Colors.red.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(20),
                  border: Border.all(
                    color: chatProvider.isConnected
                        ? const Color(0xFF00C896)
                        : Colors.red,
                    width: 1,
                  ),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Container(
                      width: 6,
                      height: 6,
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: chatProvider.isConnected
                            ? const Color(0xFF00C896)
                            : Colors.red,
                      ),
                    ),
                    const SizedBox(width: 6),
                    Text(
                      chatProvider.isConnected ? 'Connected' : 'Disconnected',
                      style: TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.w500,
                        color: chatProvider.isConnected
                            ? const Color(0xFF00C896)
                            : Colors.red,
                      ),
                    ),
                  ],
                ),
              );
            },
          ),
          const SizedBox(width: 16),
        ],
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

  Widget _buildMenuDropdown(BuildContext context, AuthService authService) {
    final theme = Theme.of(context);
    return PopupMenuButton<String>(
      icon: Icon(
        Icons.menu_rounded,
        color: theme.colorScheme.onSurface,
      ),
      offset: const Offset(0, 50),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
      ),
      onSelected: (value) async {
        switch (value) {
          case 'clear_all':
            await _clearAllChats();
            break;
          case 'sign_out':
            await _signOut();
            break;
        }
      },
      itemBuilder: (context) => [
        PopupMenuItem<String>(
          value: 'clear_all',
          child: Row(
            children: [
              Icon(
                Icons.delete_sweep_rounded,
                color: Colors.red.shade600,
                size: 20,
              ),
              const SizedBox(width: 12),
              const Text('Clear All Chats'),
            ],
          ),
        ),
        const PopupMenuDivider(),
        PopupMenuItem<String>(
          value: 'sign_out',
          child: Row(
            children: [
              Icon(
                Icons.logout_rounded,
                color: Colors.grey.shade600,
                size: 20,
              ),
              const SizedBox(width: 12),
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text('Sign Out'),
                  if (authService.currentUser?.email != null)
                    Text(
                      authService.currentUser!.email!,
                      style: TextStyle(
                        fontSize: 12,
                        color: Colors.grey.shade600,
                      ),
                    )
                  else
                    Text(
                      'Anonymous User',
                      style: TextStyle(
                        fontSize: 12,
                        color: Colors.grey.shade600,
                      ),
                    ),
                ],
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildUserSelector() {
    return PopupMenuButton<String>(
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          gradient: LinearGradient(
            colors: [
              const Color(0xFF00C896).withOpacity(0.1),
              const Color(0xFF6366F1).withOpacity(0.1),
            ],
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
          ),
          borderRadius: BorderRadius.circular(20),
          border: Border.all(
            color: const Color(0xFF00C896).withOpacity(0.3),
            width: 1,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.person_rounded,
              size: 16,
              color: const Color(0xFF00C896),
            ),
            const SizedBox(width: 6),
            Text(
              'User: $_selectedUserId',
              style: const TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w500,
                color: Color(0xFF00C896),
              ),
            ),
            const SizedBox(width: 4),
            Icon(
              Icons.keyboard_arrow_down_rounded,
              size: 16,
              color: const Color(0xFF00C896),
            ),
          ],
        ),
      ),
      offset: const Offset(0, 50),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
      ),
      onSelected: _onUserChanged,
      itemBuilder: (context) => _testUsers.map((userId) {
        return PopupMenuItem<String>(
          value: userId,
          child: Row(
            children: [
              Icon(
                _selectedUserId == userId ? Icons.check_circle : Icons.person_outline,
                size: 16,
                color: _selectedUserId == userId
                    ? const Color(0xFF00C896)
                    : Colors.grey.shade600,
              ),
              const SizedBox(width: 8),
              Text(
                'User: $userId',
                style: TextStyle(
                  fontWeight: _selectedUserId == userId
                      ? FontWeight.w600
                      : FontWeight.normal,
                  color: _selectedUserId == userId
                      ? const Color(0xFF00C896)
                      : Colors.white,
                ),
              ),
            ],
          ),
        );
      }).toList(),
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
            Container(
              width: 80,
              height: 80,
              decoration: BoxDecoration(
                color: Colors.red.shade50,
                borderRadius: BorderRadius.circular(20),
              ),
              child: Icon(
                Icons.cloud_off_rounded,
                size: 40,
                color: Colors.red.shade600,
              ),
            ),
            const SizedBox(height: 24),
            Text(
              'Connection Error',
              style: theme.textTheme.titleLarge?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              chatProvider.connectionError!,
              style: theme.textTheme.bodyMedium,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 24),
            ElevatedButton.icon(
              onPressed: () => chatProvider.reconnect(),
              icon: const Icon(Icons.refresh_rounded),
              label: const Text('Try Again'),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildChatList(ChatProvider chatProvider) {
    WidgetsBinding.instance.addPostFrameCallback((_) => _scrollToBottom());

    return ListView.builder(
      controller: _scrollController,
      padding: const EdgeInsets.all(16),
      itemCount: chatProvider.messages.length + (chatProvider.isTyping ? 1 : 0),
      itemBuilder: (context, index) {
        if (index == chatProvider.messages.length && chatProvider.isTyping) {
          return const TypingIndicator();
        }

        final message = chatProvider.messages[index];
        return Padding(
          padding: const EdgeInsets.only(bottom: 16),
          child: MessageWidget(message: message),
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
