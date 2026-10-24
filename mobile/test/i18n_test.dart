import 'package:ai_travel/src/i18n/strings.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('i18n', () {
    test('default and fallback are English', () {
      expect(kDefaultLocale, 'en');
      expect(kFallbackLocale, 'en');
    });

    test('switcher order is EN, AZ, RU, TR', () {
      expect(kLocales, ['en', 'az', 'ru', 'tr']);
    });

    test('every locale defines the same keys as English', () {
      final enKeys = kMessages['en']!.keys.toSet();
      for (final loc in kLocales) {
        final keys = kMessages[loc]!.keys.toSet();
        expect(keys, enKeys, reason: 'locale "$loc" key set differs from EN');
      }
    });

    test('tr interpolates placeholders and falls back to EN then key', () {
      expect(tr('en', 'res.day', {'n': 3}), 'Day 3');
      expect(tr('az', 'res.nights', {'n': 2}), '2 gecə');
      // Unknown key returns the key itself.
      expect(tr('en', 'does.not.exist'), 'does.not.exist');
    });
  });
}
