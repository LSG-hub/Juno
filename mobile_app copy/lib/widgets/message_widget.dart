import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:url_launcher/url_launcher.dart';
import '../models/message.dart';

class MessageWidget extends StatelessWidget {
  final ChatMessage message;
  final bool showTimestamp;

  const MessageWidget({
    super.key,
    required this.message,
    this.showTimestamp = false,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isUser = message.isUser;
    
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 4, horizontal: 16),
      child: Column(
        crossAxisAlignment: isUser ? CrossAxisAlignment.end : CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: isUser ? MainAxisAlignment.end : MainAxisAlignment.start,
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              if (!isUser) ...[
                _buildAvatar(theme),
                const SizedBox(width: 8),
              ],
              Flexible(
                child: _buildMessageBubble(theme),
              ),
              if (isUser) ...[
                const SizedBox(width: 8),
                _buildUserAvatar(theme),
              ],
            ],
          ),
          if (showTimestamp)
            Padding(
              padding: EdgeInsets.only(
                top: 4,
                left: isUser ? 0 : 48,
                right: isUser ? 48 : 0,
              ),
              child: Text(
                DateFormat('HH:mm').format(message.timestamp),
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                  fontSize: 12,
                ),
                textAlign: isUser ? TextAlign.end : TextAlign.start,
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildAvatar(ThemeData theme) {
    return Container(
      width: 32,
      height: 32,
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
        size: 18,
      ),
    );
  }

  Widget _buildUserAvatar(ThemeData theme) {
    return Container(
      width: 32,
      height: 32,
      decoration: BoxDecoration(
        color: theme.colorScheme.secondary,
        shape: BoxShape.circle,
      ),
      child: Icon(
        Icons.person,
        color: theme.colorScheme.onSecondary,
        size: 18,
      ),
    );
  }

  Widget _buildMessageBubble(ThemeData theme) {
    final isUser = message.isUser;
    final hasError = message.status == MessageStatus.error;
    
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: _getBubbleColor(theme, isUser, hasError),
        borderRadius: BorderRadius.only(
          topLeft: const Radius.circular(20),
          topRight: const Radius.circular(20),
          bottomLeft: Radius.circular(isUser ? 20 : 4),
          bottomRight: Radius.circular(isUser ? 4 : 20),
        ),
        border: hasError
            ? Border.all(color: theme.colorScheme.error, width: 1)
            : null,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            message.text,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: _getTextColor(theme, isUser, hasError),
              height: 1.4,
            ),
          ),
          // Show login button for login_required messages
          if (_isLoginRequiredMessage())
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: _buildLoginButton(theme),
            ),
          if (message.status == MessageStatus.sending)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  SizedBox(
                    width: 12,
                    height: 12,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      valueColor: AlwaysStoppedAnimation<Color>(
                        _getTextColor(theme, isUser, false).withValues(alpha: 0.6),
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Text(
                    'Sending...',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: _getTextColor(theme, isUser, false).withValues(alpha: 0.6),
                      fontSize: 11,
                    ),
                  ),
                ],
              ),
            ),
          if (hasError)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(
                    Icons.error_outline,
                    size: 16,
                    color: theme.colorScheme.error,
                  ),
                  const SizedBox(width: 4),
                  Text(
                    'Failed to send',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.error,
                      fontSize: 11,
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }

  Color _getBubbleColor(ThemeData theme, bool isUser, bool hasError) {
    if (hasError) {
      return theme.colorScheme.errorContainer.withValues(alpha: 0.3);
    }
    if (isUser) {
      return theme.colorScheme.primary;
    }
    return theme.colorScheme.surfaceContainerHighest;
  }

  Color _getTextColor(ThemeData theme, bool isUser, bool hasError) {
    if (hasError) {
      return theme.colorScheme.onErrorContainer;
    }
    if (isUser) {
      return theme.colorScheme.onPrimary;
    }
    return theme.colorScheme.onSurface;
  }

  bool _isLoginRequiredMessage() {
    return message.metadata != null && 
           message.metadata!['type'] == 'login_required';
  }

  Widget _buildLoginButton(ThemeData theme) {
    final loginUrl = message.metadata?['login_url'] ?? '';
    
    return SizedBox(
      width: double.infinity,
      child: ElevatedButton.icon(
        onPressed: () => _launchLoginUrl(loginUrl),
        icon: const Icon(Icons.login, size: 18),
        label: const Text('Login to Fi Money'),
        style: ElevatedButton.styleFrom(
          backgroundColor: theme.colorScheme.primary,
          foregroundColor: theme.colorScheme.onPrimary,
          padding: const EdgeInsets.symmetric(vertical: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
      ),
    );
  }

  Future<void> _launchLoginUrl(String url) async {
    if (url.isEmpty) return;
    
    try {
      final uri = Uri.parse(url);
      if (await canLaunchUrl(uri)) {
        await launchUrl(uri, mode: LaunchMode.externalApplication);
      } else {
        debugPrint('Could not launch $url');
      }
    } catch (e) {
      debugPrint('Error launching URL: $e');
    }
  }
}