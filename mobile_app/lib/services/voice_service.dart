// lib/services/web_voice_service.dart

import 'dart:html' as html;
import 'package:flutter/material.dart';

class WebVoiceService extends ChangeNotifier {
  html.SpeechRecognition? _speechRecognition;
  bool _isListening = false;
  bool _isAvailable = false;
  String _currentTranscript = '';
  String _browserInfo = '';
  
  bool get isListening => _isListening;
  bool get isAvailable => _isAvailable;
  String get currentTranscript => _currentTranscript;
  String get browserInfo => _browserInfo;
  
  WebVoiceService() {
    _checkBrowserSupport();
  }
  
  void _checkBrowserSupport() {
    _browserInfo = html.window.navigator.userAgent;
    debugPrint('Browser: $_browserInfo');
    
    // Check if browser supports Web Speech API
    try {
      if (html.window.navigator.userAgent.contains('Chrome') ||
          html.window.navigator.userAgent.contains('Edg/')) {
        _isAvailable = true;
        debugPrint('Web Speech API is supported');
      } else {
        _isAvailable = false;
        debugPrint('Web Speech API may not be fully supported in this browser');
      }
    } catch (e) {
      _isAvailable = false;
      debugPrint('Error checking browser support: $e');
    }
  }
  
  Future<bool> initialize() async {
    if (!_isAvailable) {
      debugPrint('Web Speech API not available');
      return false;
    }
    
    try {
      _speechRecognition = html.SpeechRecognition();
      
      // Configure speech recognition
      _speechRecognition!.continuous = false;
      _speechRecognition!.interimResults = true;
      _speechRecognition!.lang = 'en-US';
      _speechRecognition!.maxAlternatives = 1;
      
      // Set up event listeners
      _setupEventListeners();
      
      debugPrint('Web Speech API initialized successfully');
      notifyListeners();
      return true;
    } catch (e) {
      debugPrint('Failed to initialize Web Speech API: $e');
      _isAvailable = false;
      notifyListeners();
      return false;
    }
  }
  
  void _setupEventListeners() {
    if (_speechRecognition == null) return;
    
    // On result
    _speechRecognition!.onResult.listen((event) {
      final results = event.results;
      if (results != null && results.isNotEmpty) {
        final lastResult = results.last;
        if ((lastResult.length ?? 0) > 0) {
          final transcript = lastResult.item(0).transcript;
          _currentTranscript = transcript ?? '';
          
          debugPrint('Transcript: $_currentTranscript (Final: ${lastResult.isFinal})');
          
          if (lastResult.isFinal == true) {
            // Final result - stop listening
            _isListening = false;
            notifyListeners();
          } else {
            // Interim result - update UI
            notifyListeners();
          }
        }
      }
    });
    
    // On error
    _speechRecognition!.onError.listen((event) {
      debugPrint('Speech recognition error: ${event.error}');
      _isListening = false;
      _currentTranscript = '';
      
      // Show user-friendly error
      if (event.error == 'not-allowed') {
        _currentTranscript = 'Microphone permission denied';
      } else if (event.error == 'no-speech') {
        _currentTranscript = 'No speech detected';
      }
      
      notifyListeners();
    });
    
    // On end
    _speechRecognition!.onEnd.listen((event) {
      debugPrint('Speech recognition ended');
      _isListening = false;
      notifyListeners();
    });
    
    // On start
    _speechRecognition!.onStart.listen((event) {
      debugPrint('Speech recognition started');
      _isListening = true;
      _currentTranscript = '';
      notifyListeners();
    });
  }
  
  Future<void> startListening({
    required Function(String) onResult,
  }) async {
    if (!_isAvailable || _speechRecognition == null || _isListening) {
      debugPrint('Cannot start listening: Available=$_isAvailable, Listening=$_isListening');
      return;
    }
    
    try {
      // Set up result handler
      _speechRecognition!.onResult.listen((event) {
        final results = event.results;
        if (results != null && results.isNotEmpty) {
          final lastResult = results.last;
          if ((lastResult.length ?? 0) > 0) {
            final transcript = lastResult.item(0).transcript ?? '';
            _currentTranscript = transcript;
            
            if (lastResult.isFinal == true && transcript.isNotEmpty) {
              onResult(transcript);
              _currentTranscript = '';
            }
            
            notifyListeners();
          }
        }
      });
      
      // Start recognition
      _speechRecognition!.start();
      debugPrint('Started speech recognition');
    } catch (e) {
      debugPrint('Error starting speech recognition: $e');
      _isListening = false;
      notifyListeners();
    }
  }
  
  Future<void> stopListening() async {
    if (!_isListening || _speechRecognition == null) return;
    
    try {
      _speechRecognition!.stop();
      _isListening = false;
      _currentTranscript = '';
      notifyListeners();
      debugPrint('Stopped speech recognition');
    } catch (e) {
      debugPrint('Error stopping speech recognition: $e');
    }
  }
  
  Future<void> toggleListening({
    required Function(String) onResult,
  }) async {
    if (_isListening) {
      await stopListening();
    } else {
      await startListening(onResult: onResult);
    }
  }
  
  @override
  void dispose() {
    if (_speechRecognition != null && _isListening) {
      _speechRecognition!.stop();
    }
    super.dispose();
  }
}