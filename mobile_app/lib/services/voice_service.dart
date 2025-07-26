import 'dart:async';
import 'package:flutter/material.dart';
import 'package:speech_to_text/speech_to_text.dart' as stt;

class VoiceService extends ChangeNotifier {
  late stt.SpeechToText _speechToText;

  bool _isInitialized = false;
  bool _isListening = false;
  String _currentTranscript = '';
  bool _speechEnabled = false;

  // Getters
  bool get isAvailable => _isInitialized && _speechEnabled;
  bool get isListening => _isListening;
  bool get isInitialized => _isInitialized;
  String get currentTranscript => _currentTranscript;

  // Initialize with browser's Web Speech API
  Future<bool> initialize() async {
    try {
      _speechToText = stt.SpeechToText();
      
      _speechEnabled = await _speechToText.initialize(
        onError: (errorNotification) {
          debugPrint('Speech error: ${errorNotification.errorMsg}');
          _isListening = false;
          notifyListeners();
        },
        onStatus: (status) {
          debugPrint('Speech status: $status');
          if (status == 'notListening') {
            _isListening = false;
            notifyListeners();
          }
        },
      );

      _isInitialized = _speechEnabled;
      notifyListeners();
      
      if (!_speechEnabled) {
        debugPrint('Web Speech API not available. Make sure you are using Chrome/Edge and have HTTPS.');
      }
      
      return _speechEnabled;
    } catch (e) {
      debugPrint('Failed to initialize speech recognition: $e');
      return false;
    }
  }

  // Start listening for speech
  Future<void> startListening({
    required Function(String) onFinalResult,
    Function(String)? onInterimResult,
  }) async {
    if (!_speechEnabled || _isListening) return;

    try {
      _isListening = true;
      _currentTranscript = '';
      notifyListeners();

      await _speechToText.listen(
        onResult: (result) {
          _currentTranscript = result.recognizedWords;
          
          if (result.finalResult) {
            debugPrint('Final speech result: ${result.recognizedWords}');
            onFinalResult(result.recognizedWords);
            _isListening = false;
          } else if (onInterimResult != null) {
            onInterimResult(result.recognizedWords);
          }
          notifyListeners();
        },
        listenFor: const Duration(seconds: 30),
        pauseFor: const Duration(seconds: 3),
        partialResults: true,
        localeId: 'en_US',
        cancelOnError: true,
        listenMode: stt.ListenMode.confirmation,
      );
    } catch (e) {
      debugPrint('Error starting speech recognition: $e');
      _isListening = false;
      notifyListeners();
    }
  }

  // Stop listening
  Future<void> stopListening() async {
    if (!_isListening) return;

    try {
      await _speechToText.stop();
      _isListening = false;
      notifyListeners();
    } catch (e) {
      debugPrint('Error stopping speech recognition: $e');
    }
  }

  // Toggle listening state
  void toggleListening({required Function(String) onResult}) async {
    if (_isListening) {
      await stopListening();
    } else {
      await startListening(
        onFinalResult: onResult,
        onInterimResult: (text) {
          // Optional: Show interim results
        },
      );
    }
  }

  @override
  void dispose() {
    if (_speechToText.isAvailable) {
      _speechToText.stop();
    }
    super.dispose();
  }
}