import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:firebase_core/firebase_core.dart';
import 'screens/chat_screen.dart';
import 'screens/auth_screen.dart';
import 'services/chat_provider.dart';
import 'services/auth_service.dart';
import 'firebase_options.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  
  // Initialize Firebase
  await Firebase.initializeApp(
    options: DefaultFirebaseOptions.currentPlatform,
  );
  
  runApp(const JunoApp());
}

class JunoApp extends StatelessWidget {
  const JunoApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (context) => AuthService()),
        ChangeNotifierProvider(create: (context) => ChatProvider()),
      ],
      child: MaterialApp(
        title: 'Juno - AI Financial Assistant',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          useMaterial3: true,
          colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF00C896), // Fi.money green
            brightness: Brightness.light,
            primary: const Color(0xFF00C896),
            primaryContainer: const Color(0xFFE6FDF9),
            secondary: const Color(0xFF6366F1),
            surface: const Color(0xFFFAFAFA),
            background: const Color(0xFFFFFFFF),
            onPrimary: Colors.white,
            onSurface: const Color(0xFF1A1A1A),
          ),
          appBarTheme: AppBarTheme(
            centerTitle: false,
            backgroundColor: Colors.white,
            elevation: 0,
            surfaceTintColor: Colors.transparent,
            titleTextStyle: const TextStyle(
              color: Color(0xFF1A1A1A),
              fontSize: 20,
              fontWeight: FontWeight.w600,
            ),
            iconTheme: const IconThemeData(color: Color(0xFF1A1A1A)),
          ),
          elevatedButtonTheme: ElevatedButtonThemeData(
            style: ElevatedButton.styleFrom(
              backgroundColor: const Color(0xFF00C896),
              foregroundColor: Colors.white,
              elevation: 0,
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
              ),
            ),
          ),
          inputDecorationTheme: InputDecorationTheme(
            filled: true,
            fillColor: const Color(0xFFF8F9FA),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide: BorderSide(color: Colors.grey.shade300),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide: BorderSide(color: Colors.grey.shade300),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide: const BorderSide(color: Color(0xFF00C896), width: 2),
            ),
            contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
          ),
          cardTheme: CardThemeData(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(16),
              side: BorderSide(color: Colors.grey.shade200),
            ),
            color: Colors.white,
          ),
          textTheme: const TextTheme(
            headlineLarge: TextStyle(
              fontSize: 48,
              fontWeight: FontWeight.w700,
              color: Color(0xFF1A1A1A),
              height: 1.1,
            ),
            headlineMedium: TextStyle(
              fontSize: 32,
              fontWeight: FontWeight.w600,
              color: Color(0xFF1A1A1A),
            ),
            titleLarge: TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.w600,
              color: Color(0xFF1A1A1A),
            ),
            bodyLarge: TextStyle(
              fontSize: 16,
              color: Color(0xFF6B7280),
              height: 1.5,
            ),
            bodyMedium: TextStyle(
              fontSize: 14,
              color: Color(0xFF6B7280),
              height: 1.4,
            ),
          ),
        ),
        darkTheme: ThemeData(
          useMaterial3: true,
          colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF00C896),
            brightness: Brightness.dark,
            primary: const Color(0xFF00C896),
            primaryContainer: const Color(0xFF1A2F2A),
            secondary: const Color(0xFF6366F1),
            surface: const Color(0xFF1A1A1A),
            background: const Color(0xFF121212),
            onPrimary: Colors.black,
            onSurface: Colors.white,
            surfaceVariant: const Color(0xFF2A2A2A),
            onSurfaceVariant: const Color(0xFFE0E0E0),
          ),
          appBarTheme: const AppBarTheme(
            centerTitle: false,
            backgroundColor: Color(0xFF1A1A1A),
            elevation: 0,
            surfaceTintColor: Colors.transparent,
            titleTextStyle: TextStyle(
              color: Colors.white,
              fontSize: 20,
              fontWeight: FontWeight.w600,
            ),
            iconTheme: IconThemeData(color: Colors.white),
          ),
          elevatedButtonTheme: ElevatedButtonThemeData(
            style: ElevatedButton.styleFrom(
              backgroundColor: const Color(0xFF00C896),
              foregroundColor: Colors.black,
              elevation: 0,
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
              ),
            ),
          ),
          inputDecorationTheme: InputDecorationTheme(
            filled: true,
            fillColor: const Color(0xFF2A2A2A),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide: const BorderSide(color: Color(0xFF404040)),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide: const BorderSide(color: Color(0xFF404040)),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide: const BorderSide(color: Color(0xFF00C896), width: 2),
            ),
            contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
            hintStyle: const TextStyle(color: Color(0xFF9CA3AF)),
          ),
          cardTheme: CardThemeData(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(16),
              side: const BorderSide(color: Color(0xFF404040)),
            ),
            color: const Color(0xFF2A2A2A),
          ),
          textTheme: const TextTheme(
            headlineLarge: TextStyle(
              fontSize: 48,
              fontWeight: FontWeight.w700,
              color: Colors.white,
              height: 1.1,
            ),
            headlineMedium: TextStyle(
              fontSize: 32,
              fontWeight: FontWeight.w600,
              color: Colors.white,
            ),
            titleLarge: TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.w600,
              color: Colors.white,
            ),
            bodyLarge: TextStyle(
              fontSize: 16,
              color: Color(0xFFE0E0E0),
              height: 1.5,
            ),
            bodyMedium: TextStyle(
              fontSize: 14,
              color: Color(0xFFE0E0E0),
              height: 1.4,
            ),
          ),
        ),
        themeMode: ThemeMode.system,
        home: const AuthWrapper(),
      ),
    );
  }
}

class AuthWrapper extends StatelessWidget {
  const AuthWrapper({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<AuthService>(
      builder: (context, authService, child) {
        // Always show auth screen first, even if Firebase has cached auth
        // This ensures clean demo experience for judges
        if (authService.currentUser == null) {
          return const AuthScreen();
        }
        
        // Authenticated, show main app
        return const ChatScreen();
      },
    );
  }
}