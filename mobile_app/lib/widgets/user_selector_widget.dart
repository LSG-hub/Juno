import 'dart:ui';
import 'package:flutter/material.dart';

class UserSelectorWidget extends StatefulWidget {
  final String selectedUserId;
  final ValueChanged<String> onUserChanged;

  const UserSelectorWidget({
    super.key,
    required this.selectedUserId,
    required this.onUserChanged,
  });

  @override
  State<UserSelectorWidget> createState() => _UserSelectorWidgetState();
}

class _UserSelectorWidgetState extends State<UserSelectorWidget>
    with SingleTickerProviderStateMixin {
  bool _isDropdownOpen = false;
  late AnimationController _animationController;
  late Animation<double> _scaleAnimation;
  late Animation<double> _opacityAnimation;
  OverlayEntry? _overlayEntry;
  final LayerLink _layerLink = LayerLink();

  // All 16 test phone numbers from Fi MCP server
  static const List<String> _testUsers = [
    '1010101010',
    '1111111111',
    '1212121212',
    '1313131313',
    '1414141414',
    '2020202020',
    '2121212121',
    '2222222222',
    '2525252525',
    '3333333333',
    '4444444444',
    '5555555555',
    '6666666666',
    '7777777777',
    '8888888888',
    '9999999999',
  ];

  @override
  void initState() {
    super.initState();
    _animationController = AnimationController(
      duration: const Duration(milliseconds: 200),
      vsync: this,
    );
    _scaleAnimation = Tween<double>(
      begin: 0.8,
      end: 1.0,
    ).animate(CurvedAnimation(
      parent: _animationController,
      curve: Curves.easeOutBack,
    ));
    _opacityAnimation = Tween<double>(
      begin: 0.0,
      end: 1.0,
    ).animate(CurvedAnimation(
      parent: _animationController,
      curve: Curves.easeOut,
    ));
  }

  @override
  void dispose() {
    _removeOverlay();
    _animationController.dispose();
    super.dispose();
  }

  void _toggleDropdown() {
    setState(() {
      _isDropdownOpen = !_isDropdownOpen;
    });
    if (_isDropdownOpen) {
      _showOverlay();
      _animationController.forward();
    } else {
      _removeOverlay();
      _animationController.reverse();
    }
  }

  void _showOverlay() {
    _overlayEntry = _createOverlayEntry();
    Overlay.of(context).insert(_overlayEntry!);
  }

  void _removeOverlay() {
    _overlayEntry?.remove();
    _overlayEntry = null;
  }

  OverlayEntry _createOverlayEntry() {
    return OverlayEntry(
      builder: (context) => GestureDetector(
        onTap: _toggleDropdown, // Close dropdown when tapping outside
        child: Container(
          width: MediaQuery.of(context).size.width,
          height: MediaQuery.of(context).size.height,
          color: Colors.transparent,
          child: Stack(
            children: [
              Positioned(
                width: 200,
                child: CompositedTransformFollower(
                  link: _layerLink,
                  showWhenUnlinked: false,
                  offset: const Offset(-150, 45), // Position below and aligned right
                  child: AnimatedBuilder(
                    animation: _animationController,
                    builder: (context, child) {
                      return Transform.scale(
                        scale: _scaleAnimation.value,
                        alignment: Alignment.topRight,
                        child: Opacity(
                          opacity: _opacityAnimation.value,
                          child: GestureDetector(
                            onTap: () {}, // Prevent closing when tapping inside dropdown
                            child: _buildLiquidGlassDropdown(Theme.of(context)),
                          ),
                        ),
                      );
                    },
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  void _selectUser(String userId) {
    widget.onUserChanged(userId);
    _toggleDropdown();
  }

  String _formatUserDisplay(String userId) {
    return 'User: $userId';
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    
    return CompositedTransformTarget(
      link: _layerLink,
      child: GestureDetector(
        onTap: _toggleDropdown,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          decoration: BoxDecoration(
            gradient: LinearGradient(
              colors: [
                theme.colorScheme.primary.withValues(alpha: 0.1),
                theme.colorScheme.primaryContainer.withValues(alpha: 0.1),
              ],
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
            ),
            borderRadius: BorderRadius.circular(20),
            border: Border.all(
              color: theme.colorScheme.primary.withValues(alpha: 0.3),
              width: 1,
            ),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.person,
                size: 16,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 6),
              Text(
                _formatUserDisplay(widget.selectedUserId),
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.primary,
                  fontWeight: FontWeight.w500,
                ),
              ),
              const SizedBox(width: 4),
              AnimatedRotation(
                turns: _isDropdownOpen ? 0.5 : 0,
                duration: const Duration(milliseconds: 200),
                child: Icon(
                  Icons.keyboard_arrow_down,
                  size: 16,
                  color: theme.colorScheme.primary,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildLiquidGlassDropdown(ThemeData theme) {
    return Material(
      color: Colors.transparent,
      child: ClipRRect(
        borderRadius: BorderRadius.circular(16),
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 10, sigmaY: 10),
          child: Container(
            width: 200,
            constraints: const BoxConstraints(maxHeight: 320),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [
                  const Color(0xFF6750A4).withValues(alpha: 0.2),
                  Colors.white.withValues(alpha: 0.1),
                  const Color(0xFF6750A4).withValues(alpha: 0.1),
                ],
              ),
              borderRadius: BorderRadius.circular(16),
              border: Border.all(
                color: Colors.white.withValues(alpha: 0.2),
                width: 1,
              ),
              boxShadow: [
                BoxShadow(
                  color: const Color(0xFF6750A4).withValues(alpha: 0.15),
                  blurRadius: 20,
                  spreadRadius: -2,
                  offset: const Offset(0, 8),
                ),
                BoxShadow(
                  color: Colors.black.withValues(alpha: 0.1),
                  blurRadius: 10,
                  spreadRadius: -5,
                  offset: const Offset(0, 4),
                ),
              ],
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                // Header
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    border: Border(
                      bottom: BorderSide(
                        color: Colors.white.withValues(alpha: 0.1),
                        width: 1,
                      ),
                    ),
                  ),
                  child: Text(
                    'Select Test User',
                    style: theme.textTheme.titleSmall?.copyWith(
                      color: Colors.white,
                      fontWeight: FontWeight.w600,
                    ),
                    textAlign: TextAlign.center,
                  ),
                ),
                
                // User list
                Flexible(
                  child: ListView.builder(
                    shrinkWrap: true,
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    itemCount: _testUsers.length,
                    itemBuilder: (context, index) {
                      final userId = _testUsers[index];
                      final isSelected = userId == widget.selectedUserId;
                      
                      return _buildUserTile(theme, userId, isSelected);
                    },
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildUserTile(ThemeData theme, String userId, bool isSelected) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: () => _selectUser(userId),
        borderRadius: BorderRadius.circular(8),
        child: Container(
          margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            gradient: isSelected
                ? LinearGradient(
                    colors: [
                      Colors.white.withValues(alpha: 0.2),
                      Colors.white.withValues(alpha: 0.1),
                    ],
                  )
                : null,
            borderRadius: BorderRadius.circular(8),
            border: isSelected
                ? Border.all(
                    color: Colors.white.withValues(alpha: 0.3),
                    width: 1,
                  )
                : null,
          ),
          child: Row(
            children: [
              Container(
                width: 32,
                height: 32,
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    colors: [
                      theme.colorScheme.primary.withValues(alpha: 0.8),
                      theme.colorScheme.primaryContainer.withValues(alpha: 0.8),
                    ],
                  ),
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.person,
                  color: Colors.white,
                  size: 16,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      userId,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: Colors.white,
                        fontWeight: isSelected ? FontWeight.w600 : FontWeight.w500,
                      ),
                    ),
                    Text(
                      'Test Dataset ${_testUsers.indexOf(userId) + 1}',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: Colors.white.withValues(alpha: 0.7),
                        fontSize: 11,
                      ),
                    ),
                  ],
                ),
              ),
              if (isSelected)
                Icon(
                  Icons.check_circle,
                  color: Colors.white,
                  size: 16,
                ),
            ],
          ),
        ),
      ),
    );
  }
}