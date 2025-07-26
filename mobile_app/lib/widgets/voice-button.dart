import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/voice_service.dart';

class VoiceButton extends StatefulWidget {
  final Function(String) onTextReceived;
  final bool isEnabled;
  
  const VoiceButton({
    Key? key,
    required this.onTextReceived,
    this.isEnabled = true,
  }) : super(key: key);
  
  @override
  State<VoiceButton> createState() => _VoiceButtonState();
}

class _VoiceButtonState extends State<VoiceButton>
    with TickerProviderStateMixin {
  late AnimationController _pulseController;
  late Animation<double> _pulseAnimation;
  
  @override
  void initState() {
    super.initState();
    
    _pulseController = AnimationController(
      duration: const Duration(milliseconds: 1000),
      vsync: this,
    );
    
    _pulseAnimation = Tween<double>(
      begin: 1.0,
      end: 1.2,
    ).animate(CurvedAnimation(
      parent: _pulseController,
      curve: Curves.easeInOut,
    ));
  }
  
  @override
  void dispose() {
    _pulseController.dispose();
    super.dispose();
  }
  
  void _handleVoiceInput(VoiceService voiceService) async {
    if (!voiceService.isAvailable) {
      _showBrowserCompatibilityMessage();
      return;
    }

    if (voiceService.isListening) {
      await voiceService.stopListening();
      _pulseController.stop();
    } else {
      _pulseController.repeat(reverse: true);
      
      voiceService.toggleListening(
        onResult: (text) {
          if (text.isNotEmpty) {
            widget.onTextReceived(text);
          }
          _pulseController.stop();
        },
      );
    }
  }
  
  void _showBrowserCompatibilityMessage() {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: const Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Voice input not available'),
            Text(
              'Please use Chrome or Edge browser with HTTPS',
              style: TextStyle(fontSize: 12, color: Colors.white70),
            ),
          ],
        ),
        action: SnackBarAction(
          label: 'OK',
          onPressed: () {
            ScaffoldMessenger.of(context).hideCurrentSnackBar();
          },
        ),
        duration: const Duration(seconds: 5),
      ),
    );
  }
  
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    
    return Consumer<VoiceService>(
      builder: (context, voiceService, child) {
        final isListening = voiceService.isListening;
        final isAvailable = voiceService.isAvailable && widget.isEnabled;
        
        return Stack(
          alignment: Alignment.center,
          children: [
            // Animated pulse when listening
            if (isListening)
              AnimatedBuilder(
                animation: _pulseAnimation,
                builder: (context, child) {
                  return Container(
                    width: 80 * _pulseAnimation.value,
                    height: 80 * _pulseAnimation.value,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: Colors.red.withOpacity(0.3),
                        width: 2,
                      ),
                    ),
                  );
                },
              ),
            
            // Main button
            Container(
              width: 56,
              height: 56,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                gradient: LinearGradient(
                  colors: isListening
                      ? [Colors.red, Colors.redAccent]
                      : isAvailable
                          ? [theme.colorScheme.primary, theme.colorScheme.primaryContainer]
                          : [Colors.grey, Colors.grey.shade300],
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                ),
                boxShadow: isAvailable ? [
                  BoxShadow(
                    color: (isListening ? Colors.red : theme.colorScheme.primary)
                        .withOpacity(0.3),
                    blurRadius: 12,
                    spreadRadius: 2,
                  ),
                ] : null,
              ),
              child: Material(
                color: Colors.transparent,
                child: InkWell(
                  borderRadius: BorderRadius.circular(28),
                  onTap: () => _handleVoiceInput(voiceService),
                  child: Icon(
                    isListening 
                        ? Icons.stop 
                        : isAvailable 
                            ? Icons.mic 
                            : Icons.mic_off,
                    color: Colors.white,
                    size: 28,
                  ),
                ),
              ),
            ),
          ],
        );
      },
    );
  }
}