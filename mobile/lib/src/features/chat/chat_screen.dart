import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../state/settings_controller.dart';
import '../../theme/brand.dart';
import '../../widgets/links.dart';
import 'chat_state.dart';

/// AI assistant chat. Answers render as Markdown (no raw "**"), links are tappable,
/// a typing indicator shows while waiting, and failures offer retry. Tap a user
/// message to edit & resend.
class ChatScreen extends ConsumerStatefulWidget {
  const ChatScreen({super.key});

  @override
  ConsumerState<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends ConsumerState<ChatScreen> {
  final _ctrl = TextEditingController();
  final _scroll = ScrollController();

  @override
  void dispose() {
    _ctrl.dispose();
    _scroll.dispose();
    super.dispose();
  }

  void _send() {
    final text = _ctrl.text;
    if (text.trim().isEmpty) return;
    _ctrl.clear();
    ref.read(chatControllerProvider.notifier).send(text);
    _scrollToBottom();
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scroll.hasClients) {
        _scroll.animateTo(_scroll.position.maxScrollExtent,
            duration: const Duration(milliseconds: 250), curve: Curves.easeOut);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final t = ref.watch(tProvider);
    final state = ref.watch(chatControllerProvider);
    final scheme = Theme.of(context).colorScheme;

    ref.listen(chatControllerProvider.select((s) => s.messages.length),
        (_, __) => _scrollToBottom());

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        titleSpacing: 0,
        title: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const LogoMark(size: 24),
            const SizedBox(width: 8),
            Text(t('ai.chat.title'),
                style: const TextStyle(fontWeight: FontWeight.w700)),
          ],
        ),
      ),
      body: SafeArea(
        child: Column(
          children: [
            Expanded(
              child: state.aiOff
                  ? _aiOff(context, t, scheme)
                  : state.messages.isEmpty && !state.loading
                      ? _empty(context, t, scheme)
                      : ListView(
                          controller: _scroll,
                          padding: const EdgeInsets.all(16),
                          children: [
                            for (final m in state.messages) _bubble(context, m, t),
                            if (state.loading) _typing(context, t),
                            if (state.error != null) _errorRow(context, t, state.error!),
                          ],
                        ),
            ),
            if (!state.aiOff) _inputBar(context, t, scheme, state.loading),
          ],
        ),
      ),
    );
  }

  Widget _bubble(BuildContext context, ChatMessage m, String Function(String, [Map<String, Object>?]) t) {
    final scheme = Theme.of(context).colorScheme;
    final isUser = m.role == ChatRole.user;
    return Align(
      alignment: isUser ? Alignment.centerRight : Alignment.centerLeft,
      child: GestureDetector(
        onTap: isUser ? () => setState(() => _ctrl.text = m.text) : null,
        child: Container(
          margin: const EdgeInsets.only(bottom: 10),
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          constraints: BoxConstraints(
              maxWidth: MediaQuery.of(context).size.width * 0.82),
          decoration: BoxDecoration(
            color: isUser
                ? scheme.primary
                : scheme.surfaceContainerHighest.withValues(alpha: 0.6),
            borderRadius: BorderRadius.only(
              topLeft: const Radius.circular(16),
              topRight: const Radius.circular(16),
              bottomLeft: Radius.circular(isUser ? 16 : 4),
              bottomRight: Radius.circular(isUser ? 4 : 16),
            ),
          ),
          child: isUser
              ? Text(m.text, style: TextStyle(color: scheme.onPrimary))
              : MarkdownBody(
                  data: m.text,
                  selectable: true,
                  onTapLink: (text, href, title) => openUrl(context, href),
                  styleSheet: MarkdownStyleSheet.fromTheme(Theme.of(context))
                      .copyWith(
                    p: TextStyle(color: scheme.onSurface, height: 1.4),
                    a: TextStyle(
                        color: scheme.primary,
                        decoration: TextDecoration.underline),
                    code: TextStyle(
                        backgroundColor:
                            scheme.surface.withValues(alpha: 0.6),
                        fontSize: 13),
                  ),
                ),
        ),
      ),
    );
  }

  Widget _typing(BuildContext context, String Function(String, [Map<String, Object>?]) t) {
    final scheme = Theme.of(context).colorScheme;
    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.only(bottom: 10),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        decoration: BoxDecoration(
          color: scheme.surfaceContainerHighest.withValues(alpha: 0.6),
          borderRadius: BorderRadius.circular(16),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const SizedBox(
                width: 14,
                height: 14,
                child: CircularProgressIndicator(strokeWidth: 2)),
            const SizedBox(width: 10),
            Text(t('ai.itin.loading'),
                style: TextStyle(
                    fontSize: 13,
                    color: scheme.onSurface.withValues(alpha: 0.7))),
          ],
        ),
      ),
    );
  }

  Widget _errorRow(BuildContext context, String Function(String, [Map<String, Object>?]) t, String key) {
    final scheme = Theme.of(context).colorScheme;
    return Row(
      children: [
        Icon(Icons.error_outline, size: 18, color: scheme.error),
        const SizedBox(width: 8),
        Expanded(child: Text(t(key))),
        TextButton(
          onPressed: () => ref.read(chatControllerProvider.notifier).retry(),
          child: Text(t('ai.retry')),
        ),
      ],
    );
  }

  Widget _empty(BuildContext context, String Function(String, [Map<String, Object>?]) t, ColorScheme scheme) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.chat_bubble_outline,
                size: 40, color: scheme.onSurface.withValues(alpha: 0.3)),
            const SizedBox(height: 12),
            Text(t('ai.chat.empty'),
                textAlign: TextAlign.center,
                style: TextStyle(
                    color: scheme.onSurface.withValues(alpha: 0.6))),
          ],
        ),
      ),
    );
  }

  Widget _aiOff(BuildContext context, String Function(String, [Map<String, Object>?]) t, ColorScheme scheme) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.cloud_off,
                size: 40, color: scheme.onSurface.withValues(alpha: 0.4)),
            const SizedBox(height: 12),
            Text(t('ai.off.title'),
                style: const TextStyle(fontWeight: FontWeight.w700)),
            const SizedBox(height: 4),
            Text(t('ai.off.desc'),
                textAlign: TextAlign.center,
                style: TextStyle(
                    fontSize: 13,
                    color: scheme.onSurface.withValues(alpha: 0.6))),
          ],
        ),
      ),
    );
  }

  Widget _inputBar(BuildContext context, String Function(String, [Map<String, Object>?]) t,
      ColorScheme scheme, bool loading) {
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
      decoration: BoxDecoration(
        color: scheme.surface,
        border: Border(top: BorderSide(color: scheme.outline)),
      ),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: _ctrl,
              minLines: 1,
              maxLines: 4,
              textInputAction: TextInputAction.send,
              onSubmitted: (_) => _send(),
              decoration: InputDecoration(hintText: t('ai.chat.placeholder')),
            ),
          ),
          const SizedBox(width: 8),
          FilledButton(
            onPressed: loading ? null : _send,
            style: FilledButton.styleFrom(
                shape: const CircleBorder(),
                padding: const EdgeInsets.all(14)),
            child: const Icon(Icons.send, size: 18),
          ),
        ],
      ),
    );
  }
}
