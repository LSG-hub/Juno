import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:geolocator/geolocator.dart';
import 'package:geocoding/geocoding.dart';

class LocationService {
  static LocationService? _instance;
  static LocationService get instance => _instance ??= LocationService._();
  LocationService._();

  Position? _lastKnownPosition;
  String? _lastKnownCity;
  String? _lastKnownState;
  String? _lastKnownCountry;
  DateTime? _lastLocationUpdate;

  // Cache location for 10 minutes to avoid repeated API calls
  static const Duration _cacheTimeout = Duration(minutes: 10);

  Future<Map<String, dynamic>?> getCurrentLocation() async {
    try {
      debugPrint('LocationService: Starting web geolocation...');
      
      // Check if we have cached location that's still valid
      if (_lastKnownPosition != null && 
          _lastLocationUpdate != null && 
          DateTime.now().difference(_lastLocationUpdate!) < _cacheTimeout) {
        debugPrint('LocationService: Using cached location data');
        return _buildLocationResponse(fromCache: true);
      }

      // For web, permissions work differently - just try to get position
      debugPrint('LocationService: Requesting current position for web...');
      Position position = await Geolocator.getCurrentPosition(
        desiredAccuracy: LocationAccuracy.medium,
        timeLimit: const Duration(seconds: 15),
      );

      debugPrint('LocationService: Position obtained: ${position.latitude}, ${position.longitude}');
      _lastKnownPosition = position;
      _lastLocationUpdate = DateTime.now();

      // Try reverse geocoding
      await _performReverseGeocoding(position.latitude, position.longitude);

      debugPrint('LocationService: Successfully obtained location: ${_lastKnownCity}, ${_lastKnownState}');
      return _buildLocationResponse();

    } catch (e) {
      debugPrint('LocationService: Location error: $e');
      debugPrint('LocationService: Error type: ${e.runtimeType}');
      
      // For web, show user-friendly error message
      if (e.toString().contains('User denied')) {
        debugPrint('LocationService: User denied geolocation permission');
      } else if (e.toString().contains('Not supported')) {
        debugPrint('LocationService: Geolocation not supported in this browser');
      } else {
        debugPrint('LocationService: Generic geolocation error');
      }
      
      return _getLastKnownLocationResponse();
    }
  }


  Future<void> _performReverseGeocoding(double latitude, double longitude) async {
    try {
      debugPrint('LocationService: Starting reverse geocoding...');
      
      // For web, geocoding might not work reliably - let's try with timeout and fallback
      List<Placemark> placemarks = await placemarkFromCoordinates(latitude, longitude)
          .timeout(const Duration(seconds: 10));

      if (placemarks.isNotEmpty) {
        Placemark place = placemarks.first;
        _lastKnownCity = place.locality ?? place.subAdministrativeArea ?? place.administrativeArea;
        _lastKnownState = place.administrativeArea ?? place.subAdministrativeArea;
        _lastKnownCountry = place.country ?? place.isoCountryCode;
        
        debugPrint('LocationService: Reverse geocoding successful: ${_lastKnownCity}, ${_lastKnownState}, ${_lastKnownCountry}');
        debugPrint('LocationService: Full placemark data: $place');
      } else {
        debugPrint('LocationService: No placemarks found from geocoding service');
        _setFallbackLocation(latitude, longitude);
      }
    } catch (e) {
      debugPrint('LocationService: Reverse geocoding failed: $e');
      // Use fallback location detection based on coordinates
      _setFallbackLocation(latitude, longitude);
    }
  }

  void _setFallbackLocation(double latitude, double longitude) {
    // Simple fallback based on coordinate ranges for major cities
    // This is a basic fallback when geocoding fails on web
    if (latitude >= 12.8 && latitude <= 13.2 && longitude >= 77.4 && longitude <= 77.9) {
      _lastKnownCity = 'Bangalore';
      _lastKnownState = 'Karnataka';
      _lastKnownCountry = 'India';
      debugPrint('LocationService: Using fallback location: Bangalore, Karnataka, India');
    } else if (latitude >= 28.4 && latitude <= 28.8 && longitude >= 77.0 && longitude <= 77.4) {
      _lastKnownCity = 'New Delhi';
      _lastKnownState = 'Delhi';
      _lastKnownCountry = 'India';
      debugPrint('LocationService: Using fallback location: New Delhi, Delhi, India');
    } else if (latitude >= 19.0 && latitude <= 19.3 && longitude >= 72.7 && longitude <= 73.1) {
      _lastKnownCity = 'Mumbai';
      _lastKnownState = 'Maharashtra';
      _lastKnownCountry = 'India';
      debugPrint('LocationService: Using fallback location: Mumbai, Maharashtra, India');
    } else {
      // Generic fallback - at least we have coordinates
      _lastKnownCity = 'Unknown City';
      _lastKnownState = 'Unknown State';
      _lastKnownCountry = 'Unknown Country';
      debugPrint('LocationService: Using generic fallback location for coordinates: $latitude, $longitude');
    }
  }

  Map<String, dynamic>? getLastKnownLocation() {
    return _getLastKnownLocationResponse();
  }

  Map<String, dynamic>? _getLastKnownLocationResponse() {
    if (_lastKnownPosition != null) {
      return _buildLocationResponse(fromCache: true);
    }
    return null;
  }

  Map<String, dynamic> _buildLocationResponse({bool fromCache = false}) {
    if (_lastKnownPosition == null) return {};

    return {
      'coordinates': {
        'latitude': _lastKnownPosition!.latitude,
        'longitude': _lastKnownPosition!.longitude,
      },
      'city': _lastKnownCity,
      'state': _lastKnownState,
      'country': _lastKnownCountry,
      'accuracy': _lastKnownPosition!.accuracy,
      'timestamp': (_lastLocationUpdate ?? _lastKnownPosition!.timestamp)?.toIso8601String(),
      'cached': fromCache,
    };
  }

  // Clear cached location (useful for testing or manual refresh)
  void clearCache() {
    _lastKnownPosition = null;
    _lastKnownCity = null;
    _lastKnownState = null;
    _lastKnownCountry = null;
    _lastLocationUpdate = null;
    debugPrint('Location cache cleared');
  }

  // Get a formatted location string for UI display
  String getLocationDisplayString() {
    if (_lastKnownCity != null && _lastKnownState != null) {
      return '$_lastKnownCity, $_lastKnownState';
    } else if (_lastKnownState != null) {
      return _lastKnownState!;
    } else if (_lastKnownCountry != null) {
      return _lastKnownCountry!;
    }
    return 'Location unavailable';
  }

  // Check if we have any location data
  bool get hasLocationData => _lastKnownPosition != null;

  // Check if location data is fresh (less than cache timeout)
  bool get isLocationFresh {
    if (_lastLocationUpdate == null) return false;
    return DateTime.now().difference(_lastLocationUpdate!) < _cacheTimeout;
  }
}