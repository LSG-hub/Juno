import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/auth_service.dart';

class AuthScreen extends StatefulWidget {
  const AuthScreen({super.key});

  @override
  State<AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends State<AuthScreen> {
  final TextEditingController _emailController = TextEditingController();
  final TextEditingController _passwordController = TextEditingController();
  final GlobalKey<FormState> _formKey = GlobalKey<FormState>();
  
  bool _isSignUp = false;
  bool _isLoading = false;
  String? _errorMessage;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _signInAnonymously() async {
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      await context.read<AuthService>().signInAnonymously();
    } catch (e) {
      setState(() {
        _errorMessage = context.read<AuthService>().getErrorMessage(e);
      });
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }

  Future<void> _signInWithEmail() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      if (_isSignUp) {
        await context.read<AuthService>().createAccountWithEmail(
          _emailController.text.trim(),
          _passwordController.text,
        );
      } else {
        await context.read<AuthService>().signInWithEmail(
          _emailController.text.trim(),
          _passwordController.text,
        );
      }
    } catch (e) {
      setState(() {
        _errorMessage = context.read<AuthService>().getErrorMessage(e);
      });
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final size = MediaQuery.of(context).size;

    return Scaffold(
      body: Container(
        width: double.infinity,
        height: double.infinity,
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [
              const Color(0xFF00C896).withOpacity(0.15),
              const Color(0xFF121212),
              const Color(0xFF6366F1).withOpacity(0.15),
            ],
          ),
        ),
        child: SafeArea(
          child: SingleChildScrollView(
            child: SizedBox(
              height: size.height - MediaQuery.of(context).padding.top,
              child: Column(
                children: [
                  // Large App Name Section (Takes up less space)
                  Expanded(
                    flex: 2,
                    child: Container(
                      width: double.infinity,
                      padding: const EdgeInsets.symmetric(horizontal: 24),
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          // App Icon
                          Container(
                            width: 120,
                            height: 120,
                            decoration: BoxDecoration(
                              gradient: const LinearGradient(
                                colors: [
                                  Color(0xFF00C896),
                                  Color(0xFF6366F1),
                                ],
                                begin: Alignment.topLeft,
                                end: Alignment.bottomRight,
                              ),
                              shape: BoxShape.circle,
                              boxShadow: [
                                BoxShadow(
                                  color: const Color(0xFF00C896).withOpacity(0.3),
                                  blurRadius: 20,
                                  spreadRadius: 5,
                                ),
                              ],
                            ),
                            child: const Icon(
                              Icons.psychology_rounded,
                              color: Colors.white,
                              size: 60,
                            ),
                          ),
                          
                          const SizedBox(height: 32),
                          
                          // App Name - Large and Prominent
                          Text(
                            'Juno',
                            style: theme.textTheme.headlineLarge?.copyWith(
                              fontSize: 72,
                              fontWeight: FontWeight.w800,
                              color: Colors.white,
                              letterSpacing: -2,
                            ),
                          ),
                          
                          const SizedBox(height: 8),
                          
                          // Tagline
                          Text(
                            'Your Financial Assistant',
                            style: theme.textTheme.titleLarge?.copyWith(
                              color: const Color(0xFF6B7280),
                              fontWeight: FontWeight.w400,
                            ),
                          ),
                          
                          const SizedBox(height: 16),
                          
                          // Description
                          Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 32),
                            child: Text(
                              'Get personalized financial insights powered by Fi Money and Google',
                              textAlign: TextAlign.center,
                              style: theme.textTheme.bodyLarge?.copyWith(
                                color: const Color(0xFF6B7280),
                                height: 1.6,
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                  
                  // Compact Login Section 
                  Expanded(
                    flex: 2,
                    child: Container(
                      width: double.infinity,
                      padding: const EdgeInsets.all(16),
                      child: Center(
                        child: SingleChildScrollView(
                          child: Container(
                            width: double.infinity,
                            constraints: const BoxConstraints(maxWidth: 350),
                            child: Card(
                              elevation: 0,
                              shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(16),
                                side: BorderSide(
                                  color: const Color(0xFF404040),
                                  width: 1,
                                ),
                              ),
                              color: const Color(0xFF2A2A2A),
                              child: Padding(
                                padding: const EdgeInsets.all(20),
                                child: Column(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    // Quick Demo Button
                                    SizedBox(
                                      width: double.infinity,
                                      height: 44,
                                      child: ElevatedButton.icon(
                                        onPressed: _isLoading ? null : _signInAnonymously,
                                        icon: _isLoading
                                            ? const SizedBox(
                                                width: 14,
                                                height: 14,
                                                child: CircularProgressIndicator(
                                                  strokeWidth: 2,
                                                  color: Colors.black,
                                                ),
                                              )
                                            : const Icon(Icons.flash_on, size: 16, color: Colors.black),
                                        label: Text(
                                          _isLoading ? 'Connecting...' : 'Quick Demo Access',
                                          style: const TextStyle(
                                            fontSize: 14,
                                            fontWeight: FontWeight.w600,
                                            color: Colors.black,
                                          ),
                                        ),
                                        style: ElevatedButton.styleFrom(
                                          backgroundColor: const Color(0xFF00C896),
                                          foregroundColor: Colors.black,
                                          elevation: 0,
                                          shape: RoundedRectangleBorder(
                                            borderRadius: BorderRadius.circular(10),
                                          ),
                                        ),
                                      ),
                                    ),
                                    
                                    const SizedBox(height: 14),
                                    
                                    // Divider
                                    Row(
                                      children: [
                                        Expanded(
                                          child: Divider(color: const Color(0xFF404040)),
                                        ),
                                        Padding(
                                          padding: const EdgeInsets.symmetric(horizontal: 10),
                                          child: Text(
                                            'or',
                                            style: TextStyle(
                                              color: const Color(0xFFB0B0B0),
                                              fontSize: 12,
                                            ),
                                          ),
                                        ),
                                        Expanded(
                                          child: Divider(color: const Color(0xFF404040)),
                                        ),
                                      ],
                                    ),
                                    
                                    const SizedBox(height: 14),

                                    // Email/Password Form - Compact
                                    Form(
                                      key: _formKey,
                                      child: Column(
                                        children: [
                                          SizedBox(
                                            height: 42,
                                            child: TextFormField(
                                              controller: _emailController,
                                              enabled: !_isLoading,
                                              keyboardType: TextInputType.emailAddress,
                                              style: const TextStyle(fontSize: 13, color: Colors.white),
                                              decoration: InputDecoration(
                                                hintText: 'Email address',
                                                hintStyle: const TextStyle(fontSize: 13, color: Color(0xFF9CA3AF)),
                                                prefixIcon: const Icon(Icons.email_outlined, size: 16, color: Color(0xFF9CA3AF)),
                                                filled: true,
                                                fillColor: const Color(0xFF2A2A2A),
                                                contentPadding: const EdgeInsets.symmetric(
                                                  horizontal: 10,
                                                  vertical: 10,
                                                ),
                                                border: OutlineInputBorder(
                                                  borderRadius: BorderRadius.circular(8),
                                                  borderSide: const BorderSide(color: Color(0xFF404040)),
                                                ),
                                                enabledBorder: OutlineInputBorder(
                                                  borderRadius: BorderRadius.circular(8),
                                                  borderSide: const BorderSide(color: Color(0xFF404040)),
                                                ),
                                                focusedBorder: OutlineInputBorder(
                                                  borderRadius: BorderRadius.circular(8),
                                                  borderSide: const BorderSide(color: Color(0xFF00C896), width: 2),
                                                ),
                                              ),
                                              validator: (value) {
                                                if (value?.isEmpty ?? true) {
                                                  return 'Please enter your email';
                                                }
                                                if (!value!.contains('@')) {
                                                  return 'Please enter a valid email';
                                                }
                                                return null;
                                              },
                                            ),
                                          ),
                                          
                                          const SizedBox(height: 10),
                                          
                                          SizedBox(
                                            height: 42,
                                            child: TextFormField(
                                              controller: _passwordController,
                                              enabled: !_isLoading,
                                              obscureText: true,
                                              style: const TextStyle(fontSize: 13, color: Colors.white),
                                              decoration: InputDecoration(
                                                hintText: 'Password',
                                                hintStyle: const TextStyle(fontSize: 13, color: Color(0xFF9CA3AF)),
                                                prefixIcon: const Icon(Icons.lock_outline, size: 16, color: Color(0xFF9CA3AF)),
                                                filled: true,
                                                fillColor: const Color(0xFF2A2A2A),
                                                contentPadding: const EdgeInsets.symmetric(
                                                  horizontal: 10,
                                                  vertical: 10,
                                                ),
                                                border: OutlineInputBorder(
                                                  borderRadius: BorderRadius.circular(8),
                                                  borderSide: const BorderSide(color: Color(0xFF404040)),
                                                ),
                                                enabledBorder: OutlineInputBorder(
                                                  borderRadius: BorderRadius.circular(8),
                                                  borderSide: const BorderSide(color: Color(0xFF404040)),
                                                ),
                                                focusedBorder: OutlineInputBorder(
                                                  borderRadius: BorderRadius.circular(8),
                                                  borderSide: const BorderSide(color: Color(0xFF00C896), width: 2),
                                                ),
                                              ),
                                              validator: (value) {
                                                if (value?.isEmpty ?? true) {
                                                  return 'Please enter your password';
                                                }
                                                if (value!.length < 6) {
                                                  return 'Password must be at least 6 characters';
                                                }
                                                return null;
                                              },
                                            ),
                                          ),
                                          
                                          const SizedBox(height: 14),
                                          
                                          // Sign In/Up Button
                                          SizedBox(
                                            width: double.infinity,
                                            height: 40,
                                            child: ElevatedButton(
                                              onPressed: _isLoading ? null : _signInWithEmail,
                                              style: ElevatedButton.styleFrom(
                                                backgroundColor: const Color(0xFF404040),
                                                foregroundColor: Colors.white,
                                                shape: RoundedRectangleBorder(
                                                  borderRadius: BorderRadius.circular(10),
                                                ),
                                              ),
                                              child: Text(
                                                _isSignUp ? 'Create Account' : 'Sign In',
                                                style: const TextStyle(
                                                  fontSize: 14,
                                                  fontWeight: FontWeight.w600,
                                                ),
                                              ),
                                            ),
                                          ),
                                        ],
                                      ),
                                    ),
                                    
                                    const SizedBox(height: 10),
                                    
                                    // Toggle Sign Up/In
                                    TextButton(
                                      onPressed: _isLoading
                                          ? null
                                          : () {
                                              setState(() {
                                                _isSignUp = !_isSignUp;
                                                _errorMessage = null;
                                              });
                                            },
                                      child: Text(
                                        _isSignUp
                                            ? 'Already have an account? Sign in'
                                            : 'Need an account? Sign up',
                                        style: const TextStyle(
                                          color: Color(0xFFB0B0B0),
                                          fontSize: 12,
                                        ),
                                      ),
                                    ),
                                    
                                    // Error Message
                                    if (_errorMessage != null) ...[
                                      const SizedBox(height: 10),
                                      Container(
                                        width: double.infinity,
                                        padding: const EdgeInsets.all(8),
                                        decoration: BoxDecoration(
                                          color: Colors.red.shade900.withOpacity(0.3),
                                          borderRadius: BorderRadius.circular(6),
                                          border: Border.all(color: Colors.red.shade700),
                                        ),
                                        child: Text(
                                          _errorMessage!,
                                          style: TextStyle(
                                            color: Colors.red.shade300,
                                            fontSize: 12,
                                          ),
                                        ),
                                      ),
                                    ],
                                  ],
                                ),
                              ),
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}