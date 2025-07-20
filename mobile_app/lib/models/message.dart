class ChatMessage {
  final String id;
  final String text;
  final bool isUser;
  final DateTime timestamp;
  final MessageStatus status;

  ChatMessage({
    required this.id,
    required this.text,
    required this.isUser,
    required this.timestamp,
    this.status = MessageStatus.sent,
  });

  ChatMessage.fromJson(Map<String, dynamic> json)
      : id = json['id'] ?? '',
        text = json['text'] ?? '',
        isUser = json['isUser'] ?? false,
        timestamp = DateTime.parse(json['timestamp'] ?? DateTime.now().toIso8601String()),
        status = MessageStatus.values.firstWhere(
          (status) => status.toString() == 'MessageStatus.${json['status']}',
          orElse: () => MessageStatus.sent,
        );

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'text': text,
      'isUser': isUser,
      'timestamp': timestamp.toIso8601String(),
      'status': status.toString().split('.').last,
    };
  }

  ChatMessage copyWith({
    String? id,
    String? text,
    bool? isUser,
    DateTime? timestamp,
    MessageStatus? status,
  }) {
    return ChatMessage(
      id: id ?? this.id,
      text: text ?? this.text,
      isUser: isUser ?? this.isUser,
      timestamp: timestamp ?? this.timestamp,
      status: status ?? this.status,
    );
  }
}

enum MessageStatus {
  sending,
  sent,
  delivered,
  error,
}