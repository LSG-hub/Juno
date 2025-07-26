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
  late AnimationController _waveController;
  late Animation<double> _pulseAnimation;
  late Animation<double> _waveAnimation;
  
  @override
  void initState() {
    super.initState();
    
    // Pulse animation for listening state
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
    
    // Wave animation for visual feedback
    _waveController = AnimationController(
      duration: const Duration(milliseconds: 2000),
      vsync: this,
    );
    
    _waveAnimation = Tween<double>(
      begin: 0.0,
      end: 1.0,
    ).animate(_waveController);
  }
  
  @override
  void dispose() {
    _pulseController.dispose();
    _waveController.dispose();
    super.dispose();
  }
  
  void _handleVoiceInput(GoogleVoiceService voiceService) async {
    if (voiceService.isListening) {
      await voiceService.stopListening();
      _pulseController.stop();
      _waveController.stop();
    } else {
      _pulseController.repeat(reverse: true);
      _waveController.repeat();
      
      await voiceService.startListening(
        onFinalResult: (text) {
          if (text.isNotEmpty) {
            widget.onTextReceived(text);
          }
          _pulseController.stop();
          _waveController.stop();
        },
        onInterimResult: (text) {
          // Optional: Show interim results in UI
        },
      );
    }
  }
  
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    
    return Consumer<GoogleVoiceService>(
      builder: (context, voiceService, child) {
        final isListening = voiceService.isListening;
        final isAvailable = voiceService.isInitialized && widget.isEnabled;
        
        return Stack(
          alignment: Alignment.center,
          children: [
            // Animated waves when listening
            if (isListening)
              AnimatedBuilder(
                animation: _waveAnimation,
                builder: (context, child) {
                  return Container(
                    width: 80,
                    height: 80,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: theme.colorScheme.primary
                            .withOpacity(1 - _waveAnimation.value),
                        width: 2,
                      ),
                    ),
                    transform: Matrix4.identity()
                      ..scale(1 + _waveAnimation.value * 0.5),
                  );
                },
              ),
            
            // Main button
            ScaleTransition(
              scale: _pulseAnimation,
              child: Container(
                width: 56,
                height: 56,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  gradient: LinearGradient(
                    colors: isListening
                        ? [Colors.red, Colors.redAccent]
                        : [theme.colorScheme.primary, theme.colorScheme.primaryContainer],
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                  ),
                  boxShadow: [
                    BoxShadow(
                      color: (isListening ? Colors.red : theme.colorScheme.primary)
                          .withOpacity(0.3),
                      blurRadius: 12,
                      spreadRadius: 2,
                    ),
                  ],
                ),
                child: Material(
                  color: Colors.transparent,
                  child: InkWell(
                    borderRadius: BorderRadius.circular(28),
                    onTap: isAvailable
                        ? () => _handleVoiceInput(voiceService)
                        : null,
                    child: Icon(
                      isListening ? Icons.stop : Icons.mic,
                      color: Colors.white,
                      size: 28,
                    ),
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