import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'router/router.dart';
import 'state/settings_controller.dart';
import 'theme/app_theme.dart';

/// Root widget. Material 3, brand light/dark themes, theme mode driven by the
/// persisted setting; routing via go_router. The UI language is handled by our
/// own i18n layer (not MaterialLocalizations), so first paint is EN.
class AiTravelApp extends ConsumerWidget {
  const AiTravelApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final themeMode = ref.watch(themeModeProvider);
    return MaterialApp.router(
      title: 'AI Travel Platform',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.light(),
      darkTheme: AppTheme.dark(),
      themeMode: themeMode,
      routerConfig: appRouter,
    );
  }
}
