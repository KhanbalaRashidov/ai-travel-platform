/// Runtime configuration. No secrets live in the app — the only thing injected is
/// the gateway base URL, provided at build time with
/// `--dart-define=API_BASE=http://<host>:8080`. The app only ever talks to the
/// gateway; all pricing/routing/AI logic stays server-side.
library;

class AppConfig {
  AppConfig._();

  /// Raw value from `--dart-define=API_BASE=...`. Empty when not provided.
  static const String _apiBaseDefine = String.fromEnvironment('API_BASE');

  /// Fallback used when no API_BASE is supplied. `10.0.2.2` is the Android
  /// emulator's alias for the host machine's `localhost`; for a real phone on the
  /// same Wi-Fi pass `--dart-define=API_BASE=http://<lan-ip>:8080`.
  static const String _fallbackHost = 'http://10.0.2.2:8080';

  /// The fully-qualified API v1 base, e.g. `http://192.168.1.10:8080/api/v1`.
  /// Accepts a bare host (`http://host:8080`) or one that already carries the
  /// `/api/v1` suffix, and normalizes both.
  static String get apiBase {
    var base = _apiBaseDefine.trim();
    if (base.isEmpty) base = _fallbackHost;
    while (base.endsWith('/')) {
      base = base.substring(0, base.length - 1);
    }
    if (base.endsWith('/api/v1')) return base;
    return '$base/api/v1';
  }
}
