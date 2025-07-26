import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/foundation.dart';

class AuthService extends ChangeNotifier {
  final FirebaseAuth _auth = FirebaseAuth.instance;
  
  User? get currentUser => _auth.currentUser;
  bool get isAuthenticated => currentUser != null;
  String? get firebaseUID => currentUser?.uid;
  String? get userEmail => currentUser?.email;
  bool get isAnonymous => currentUser?.isAnonymous ?? false;
  
  String get displayName {
    if (isAnonymous) {
      return 'Anonymous User';
    } else if (userEmail != null) {
      return userEmail!;
    } else {
      return 'User';
    }
  }

  AuthService() {
    // Listen to authentication state changes
    _auth.authStateChanges().listen((User? user) {
      notifyListeners();
    });
  }

  // Sign in anonymously
  Future<UserCredential?> signInAnonymously() async {
    try {
      UserCredential result = await _auth.signInAnonymously();
      debugPrint('Signed in anonymously: ${result.user?.uid}');
      return result;
    } catch (e) {
      debugPrint('Error signing in anonymously: $e');
      return null;
    }
  }

  // Sign in with email and password
  Future<UserCredential?> signInWithEmail(String email, String password) async {
    try {
      UserCredential result = await _auth.signInWithEmailAndPassword(
        email: email,
        password: password,
      );
      debugPrint('Signed in with email: ${result.user?.email}');
      return result;
    } catch (e) {
      debugPrint('Error signing in with email: $e');
      return null;
    }
  }

  // Create account with email and password
  Future<UserCredential?> createAccountWithEmail(String email, String password) async {
    try {
      UserCredential result = await _auth.createUserWithEmailAndPassword(
        email: email,
        password: password,
      );
      debugPrint('Created account with email: ${result.user?.email}');
      return result;
    } catch (e) {
      debugPrint('Error creating account: $e');
      return null;
    }
  }

  // Sign out
  Future<void> signOut() async {
    try {
      await _auth.signOut();
      debugPrint('User signed out');
    } catch (e) {
      debugPrint('Error signing out: $e');
    }
  }

  // Get error message from Firebase Auth exception
  String getErrorMessage(dynamic error) {
    if (error is FirebaseAuthException) {
      switch (error.code) {
        case 'user-not-found':
          return 'No user found for that email.';
        case 'wrong-password':
          return 'Wrong password provided.';
        case 'email-already-in-use':
          return 'The account already exists for that email.';
        case 'weak-password':
          return 'The password provided is too weak.';
        case 'invalid-email':
          return 'The email address is not valid.';
        default:
          return error.message ?? 'An error occurred';
      }
    }
    return 'An unexpected error occurred';
  }
}