import 'package:cloud_firestore/cloud_firestore.dart';

class ChatMessage {
  final String id;
  final String text;
  final bool isUser;
  final DateTime timestamp;
  final MessageStatus status;
  final Map<String, dynamic>? metadata;

  ChatMessage({
    required this.id,
    required this.text,
    required this.isUser,
    required this.timestamp,
    this.status = MessageStatus.sent,
    this.metadata,
  });

  ChatMessage.fromJson(Map<String, dynamic> json)
      : id = json['id'] ?? '',
        text = json['text'] ?? '',
        isUser = json['isUser'] ?? false,
        timestamp = DateTime.parse(json['timestamp'] ?? DateTime.now().toIso8601String()),
        status = MessageStatus.values.firstWhere(
          (status) => status.toString() == 'MessageStatus.${json['status']}',
          orElse: () => MessageStatus.sent,
        ),
        metadata = json['metadata'];

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'text': text,
      'isUser': isUser,
      'timestamp': timestamp.toIso8601String(),
      'status': status.toString().split('.').last,
      'metadata': metadata,
    };
  }

  // Firestore-specific serialization
  Map<String, dynamic> toFirestore() {
    return {
      'id': id,
      'text': text,
      'isUser': isUser,
      'timestamp': timestamp,  // Firestore handles DateTime directly
      'status': status.toString().split('.').last,
      'metadata': metadata,
    };
  }

  // Create from Firestore document
  ChatMessage.fromFirestore(Map<String, dynamic> data)
      : id = data['id'] ?? '',
        text = data['text'] ?? '',
        isUser = data['isUser'] ?? false,
        timestamp = (data['timestamp'] as Timestamp?)?.toDate() ?? DateTime.now(),
        status = MessageStatus.values.firstWhere(
          (status) => status.toString() == 'MessageStatus.${data['status']}',
          orElse: () => MessageStatus.sent,
        ),
        metadata = data['metadata'];

  ChatMessage copyWith({
    String? id,
    String? text,
    bool? isUser,
    DateTime? timestamp,
    MessageStatus? status,
    Map<String, dynamic>? metadata,
  }) {
    return ChatMessage(
      id: id ?? this.id,
      text: text ?? this.text,
      isUser: isUser ?? this.isUser,
      timestamp: timestamp ?? this.timestamp,
      status: status ?? this.status,
      metadata: metadata ?? this.metadata,
    );
  }
}

enum MessageStatus {
  sending,
  sent,
  delivered,
  error,
}