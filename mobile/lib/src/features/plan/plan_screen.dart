import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../i18n/language_switcher.dart';
import '../../state/settings_controller.dart';
import '../../theme/brand.dart';

/// Planner screen. The form (origin, modes, destinations, dates, passengers,
/// comfort, transport, currency), the route/live-cost result, the AI roadmap,
/// map and chat are added in the following F-stages. This F1 shell wires routing,
/// the brand app bar, the language switcher and the theme toggle.
class PlanScreen extends ConsumerWidget {
  const PlanScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.go('/'),
        ),
        titleSpacing: 0,
        title: Row(
          mainAxisSize: MainAxisSize.min,
          children: const [
            LogoMark(size: 26),
            SizedBox(width: 8),
            BrandWordmark(fontSize: 16),
          ],
        ),
        actions: const [
          ThemeToggleButton(),
          Padding(
            padding: EdgeInsets.only(right: 12, left: 4),
            child: Center(child: LanguageSwitcher()),
          ),
        ],
      ),
      body: SafeArea(
        child: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Text(
              t('plan.subtitle'),
              textAlign: TextAlign.center,
              style: TextStyle(
                color: Theme.of(context)
                    .colorScheme
                    .onSurface
                    .withValues(alpha: 0.6),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
