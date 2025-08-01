// File generated by FlutterFire CLI.
// ignore_for_file: lines_longer_than_80_chars, avoid_classes_with_only_static_members
import 'package:firebase_core/firebase_core.dart' show FirebaseOptions;
import 'package:flutter/foundation.dart'
    show defaultTargetPlatform, kIsWeb, TargetPlatform;

/// Default [FirebaseOptions] for use with your Firebase apps.
///
/// Example:
/// ```dart
/// import 'firebase_options.dart';
/// // ...
/// await Firebase.initializeApp(
///   options: DefaultFirebaseOptions.currentPlatform,
/// );
/// ```
class DefaultFirebaseOptions {
  static FirebaseOptions get currentPlatform {
    if (kIsWeb) {
      return web;
    }
    switch (defaultTargetPlatform) {
      case TargetPlatform.android:
        return android;
      case TargetPlatform.iOS:
        return ios;
      case TargetPlatform.macOS:
        return macos;
      case TargetPlatform.windows:
        throw UnsupportedError(
          'DefaultFirebaseOptions have not been configured for windows - '
          'you can reconfigure this by running the FlutterFire CLI again.',
        );
      case TargetPlatform.linux:
        throw UnsupportedError(
          'DefaultFirebaseOptions have not been configured for linux - '
          'you can reconfigure this by running the FlutterFire CLI again.',
        );
      default:
        throw UnsupportedError(
          'DefaultFirebaseOptions are not supported for this platform.',
        );
    }
  }

  static const FirebaseOptions web = FirebaseOptions(
    apiKey: 'AIzaSyCbFfZYvqbkeZlcK_Padg9hKnO7Xqbl1NI',
    appId: '1:929051225142:web:1d59d1710c38785ea0bc97',
    messagingSenderId: '929051225142',
    projectId: 'juno-financial-assistant',
    authDomain: 'juno-financial-assistant.firebaseapp.com',
    storageBucket: 'juno-financial-assistant.firebasestorage.app',
  );

  static const FirebaseOptions android = FirebaseOptions(
    apiKey: 'AIzaSyCbFfZYvqbkeZlcK_Padg9hKnO7Xqbl1NI',
    appId: '1:929051225142:android:1d59d1710c38785ea0bc97',
    messagingSenderId: '929051225142',
    projectId: 'juno-financial-assistant',
    storageBucket: 'juno-financial-assistant.firebasestorage.app',
  );

  static const FirebaseOptions ios = FirebaseOptions(
    apiKey: 'AIzaSyCbFfZYvqbkeZlcK_Padg9hKnO7Xqbl1NI',
    appId: '1:929051225142:ios:1d59d1710c38785ea0bc97',
    messagingSenderId: '929051225142',
    projectId: 'juno-financial-assistant',
    storageBucket: 'juno-financial-assistant.firebasestorage.app',
    iosClientId: '929051225142-abc123.apps.googleusercontent.com',
    iosBundleId: 'com.example.mobile-app',
  );

  static const FirebaseOptions macos = FirebaseOptions(
    apiKey: 'AIzaSyCbFfZYvqbkeZlcK_Padg9hKnO7Xqbl1NI',
    appId: '1:929051225142:ios:1d59d1710c38785ea0bc97',
    messagingSenderId: '929051225142',
    projectId: 'juno-financial-assistant',
    storageBucket: 'juno-financial-assistant.firebasestorage.app',
    iosClientId: '929051225142-abc123.apps.googleusercontent.com',
    iosBundleId: 'com.example.mobile-app',
  );
}