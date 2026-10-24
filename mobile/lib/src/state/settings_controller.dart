import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../i18n/strings.dart';

/// Persisted UI locale. First paint is EN (the default); a returning user's saved
/// choice is applied right after load. No hydration concerns on mobile, but the
/// EN-first default matches the web app.
class LocaleController extends StateNotifier<String> {
  LocaleController() : super(kDefaultLocale) {
    _load();
  }

  static const _key = 'locale';

  Future<void> _load() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final saved = prefs.getString(_key);
      if (saved != null && kLocales.contains(saved)) state = saved;
    } catch (_) {
      /* prefs may be unavailable */
    }
  }

  Future<void> set(String locale) async {
    if (!kLocales.contains(locale)) return;
    state = locale;
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_key, locale);
    } catch (_) {
      /* ignore */
    }
  }
}

final localeProvider =
    StateNotifierProvider<LocaleController, String>((ref) => LocaleController());

/// Convenience: a translator bound to the current locale. Usage:
///   final t = ref.watch(tProvider); t('plan.calc');
final tProvider =
    Provider<String Function(String, [Map<String, Object>?])>((ref) {
  final loc = ref.watch(localeProvider);
  return (key, [vars]) => tr(loc, key, vars);
});

/// Default display currency following the locale (overridable in the plan form).
final localeCurrencyProvider = Provider<String>((ref) {
  final loc = ref.watch(localeProvider);
  return kLocaleCurrency[loc] ?? 'EUR';
});

/// Light / dark / system theme, persisted.
class ThemeModeController extends StateNotifier<ThemeMode> {
  ThemeModeController() : super(ThemeMode.system) {
    _load();
  }

  static const _key = 'themeMode';

  Future<void> _load() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      switch (prefs.getString(_key)) {
        case 'light':
          state = ThemeMode.light;
        case 'dark':
          state = ThemeMode.dark;
        case 'system':
          state = ThemeMode.system;
      }
    } catch (_) {
      /* ignore */
    }
  }

  Future<void> set(ThemeMode mode) async {
    state = mode;
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_key, mode.name);
    } catch (_) {
      /* ignore */
    }
  }

  void toggle() {
    // system -> follows device; an explicit toggle flips light/dark.
    final brightness =
        WidgetsBinding.instance.platformDispatcher.platformBrightness;
    final effectivelyDark =
        state == ThemeMode.dark || (state == ThemeMode.system && brightness == Brightness.dark);
    set(effectivelyDark ? ThemeMode.light : ThemeMode.dark);
  }
}

final themeModeProvider =
    StateNotifierProvider<ThemeModeController, ThemeMode>(
        (ref) => ThemeModeController());
