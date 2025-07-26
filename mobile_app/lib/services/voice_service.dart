// lib/services/voice_service.dart

import 'dart:async';
import 'dart:io'; // Required for File access
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:google_speech/google_speech.dart';
import 'package:record/record.dart';
import 'package:permission_handler/permission_handler.dart';

class VoiceService extends ChangeNotifier {
  late SpeechToText _speechToText;
  final Record _recorder = Record();

  bool _isInitialized = false;
  bool _isListening = false;
  String _currentTranscript = '';

  // Getters
  bool get isAvailable => _isInitialized;
  bool get isListening => _isListening;
  String get currentTranscript => _currentTranscript;

  // Initialize with service account credentials
  Future<bool> initialize() async {
    try {
      final status = await Permission.microphone.request();
      if (status != PermissionStatus.granted) {
        debugPrint('Microphone permission denied');
        return false;
      }

      final serviceAccountJson = await rootBundle.loadString(
        'assets/credentials/juno-speech-credentials.json',
      );
      final serviceAccount = ServiceAccount.fromString(serviceAccountJson);

      _speechToText = SpeechToText.viaServiceAccount(serviceAccount);
      _isInitialized = true;
      notifyListeners();
      return true;
    } catch (e) {
      debugPrint('Failed to initialize Google Speech: $e');
      return false;
    }
  }

  // CHANGED: This method now controls the entire start/stop/process cycle.
  void toggleListening({required Function(String) onResult}) async {
    if (_isListening) {
      // --- STOPPING LOGIC ---
      _isListening = false;
      notifyListeners();

      final path = await _recorder.stop();
      if (path == null) {
        debugPrint('Failed to stop recording or no path found.');
        return;
      }

      debugPrint('Recording stopped. File at: $path');
      _currentTranscript = 'Processing...';
      notifyListeners();

      // Read the audio file as bytes
      final audioBytes = await File(path).readAsBytes();

      // Configure recognition for a non-streaming request
      final config = RecognitionConfig(
        encoding: AudioEncoding.LINEAR16,
        model: RecognitionModel.command_and_search,
        enableAutomaticPunctuation: true,
        sampleRateHertz: 16000,
        languageCode: 'en-US',
      );

      // Use the non-streaming recognize API
      try {
        final response = await _speechToText.recognize(config, audioBytes);
        if (response.results.isNotEmpty) {
          final transcript = response.results.first.alternatives.first.transcript;
          _currentTranscript = transcript;
          debugPrint('Final transcript: $transcript');
          onResult(transcript); // Send the final result back to the UI
        } else {
          _currentTranscript = 'Could not recognize speech.';
        }
      } catch (e) {
        debugPrint('Google Speech recognition error: $e');
        _currentTranscript = 'Error recognizing speech.';
      } finally {
        notifyListeners();
      }
    } else {
      // --- STARTING LOGIC ---
      if (!_isInitialized) {
        debugPrint('Voice service not initialized.');
        return;
      }
      
      try {
        await _recorder.start(
          encoder: AudioEncoder.wav, // Use a supported encoder
          samplingRate: 16000,
          numChannels: 1,
        );
        _isListening = true;
        _currentTranscript = 'Listening...';
        debugPrint('Recording started...');
        notifyListeners();
      } catch (e) {
        debugPrint('Error starting recording: $e');
      }
    }
  }
  
  // REMOVED: The old startListening and stopListening methods are now
  // consolidated into toggleListening for simplicity.

  @override
  void dispose() {
    _recorder.dispose();
    super.dispose();
  }
}
