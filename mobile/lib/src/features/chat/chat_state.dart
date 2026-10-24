import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/api_client.dart';
import '../../state/providers.dart';
import '../../state/settings_controller.dart';

enum ChatRole { user, assistant }

@immutable
class ChatMessage {
  const ChatMessage(this.role, this.text);
  final ChatRole role;
  final String text;
}

@immutable
class ChatState {
  const ChatState({
    this.messages = const [],
    this.loading = false,
    this.error,
    this.aiOff = false,
  });
  final List<ChatMessage> messages;
  final bool loading;
  final String? error; // i18n key
  final bool aiOff;

  ChatState copyWith({
    List<ChatMessage>? messages,
    bool? loading,
    Object? error = _s,
    bool? aiOff,
  }) =>
      ChatState(
        messages: messages ?? this.messages,
        loading: loading ?? this.loading,
        error: error == _s ? this.error : error as String?,
        aiOff: aiOff ?? this.aiOff,
      );

  static const _s = Object();
}

/// Conversational assistant over POST /ai/chat. Shows the answer as Markdown; a
/// 503 means the server has no AI key (ai.off). Supports retry of the last turn.
class ChatController extends StateNotifier<ChatState> {
  ChatController(this._ref) : super(const ChatState());
  final Ref _ref;

  Future<void> send(String text) async {
    final msg = text.trim();
    if (msg.isEmpty || state.loading) return;
    state = state.copyWith(
      messages: [...state.messages, ChatMessage(ChatRole.user, msg)],
      loading: true,
      error: null,
    );
    await _ask(msg);
  }

  /// Resend the most recent user message (after an error / for edit-resend).
  Future<void> retry() async {
    final lastUser = state.messages.lastWhere(
      (m) => m.role == ChatRole.user,
      orElse: () => const ChatMessage(ChatRole.user, ''),
    );
    if (lastUser.text.isEmpty || state.loading) return;
    state = state.copyWith(loading: true, error: null);
    await _ask(lastUser.text);
  }

  Future<void> _ask(String msg) async {
    final locale = _ref.read(localeProvider);
    try {
      final res = await _ref.read(travelApiProvider).chat(msg, locale: locale);
      state = state.copyWith(
        messages: [...state.messages, ChatMessage(ChatRole.assistant, res.text)],
        loading: false,
        error: null,
      );
    } on ApiException catch (e) {
      if (e.isAiNotConfigured) {
        state = state.copyWith(loading: false, aiOff: true, error: null);
      } else {
        state = state.copyWith(
            loading: false, error: e.status == 429 ? 'ai.busy' : 'ai.error');
      }
      if (kDebugMode) debugPrint('chat error: $e');
    } catch (e) {
      state = state.copyWith(loading: false, error: 'ai.error');
      if (kDebugMode) debugPrint('chat error: $e');
    }
  }
}

final chatControllerProvider =
    StateNotifierProvider<ChatController, ChatState>((ref) => ChatController(ref));
