import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/chat_provider.dart';
import '../services/auth_service.dart';
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
  String _selectedUserId = '1111111111'; // Default test user

  @override
  void initState() {
    super.initState();
    // Initialize chat provider
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<ChatProvider>().initialize();
    });
  }

  @override
  void dispose() {
    _textController.dispose();
    _scrollController.dispose();
    _focusNode.dispose();
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

  void _onUserChanged(String userId) {
    setState(() {
      _selectedUserId = userId;
    });
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
                      chatProvider.clearMessages();
                      break;
                    case 'reconnect':
                      chatProvider.reconnect();
                      break;
                    case 'logout':
                      // Cleanup Fi clients for this Firebase user
                      if (authService.firebaseUID != null) {
                        await chatProvider.cleanupUser(authService.firebaseUID!);
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
      body: Column(
        children: [
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
            color: theme.shadowColor.withValues(alpha: 0.1),
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