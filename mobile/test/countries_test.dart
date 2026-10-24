import 'package:ai_travel/src/i18n/countries.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('countries', () {
    test('every code resolves to a name in every UI language', () {
      for (final code in kCountryCodes) {
        for (final loc in ['en', 'az', 'ru', 'tr']) {
          final name = countryName(code, loc);
          expect(name, isNot(code),
              reason: 'missing $loc name for $code');
          expect(name.trim(), isNotEmpty);
        }
      }
    });

    test('localized names differ per language for Germany', () {
      expect(countryName('DE', 'en'), 'Germany');
      expect(countryName('DE', 'az'), 'Almaniya');
      expect(countryName('DE', 'tr'), 'Almanya');
      expect(countryName('DE', 'ru'), 'Германия');
    });

    test('search matches local name, English, diacritic-insensitive', () {
      expect(searchCountries('georgia', 'en').map((o) => o.code), contains('GE'));
      expect(searchCountries('almaniy', 'az').map((o) => o.code), contains('DE'));
      expect(searchCountries('gürc', 'az').map((o) => o.code), contains('GE'));
      expect(searchCountries('', 'en'), isEmpty);
    });

    test('canonicalName is English; codeByName reverses any language', () {
      expect(canonicalName('DE'), 'Germany');
      expect(codeByName('Almaniya'), 'DE');
      expect(codeByName('Almanya'), 'DE');
      expect(codeByName('Германия'), 'DE');
      expect(localizeCountry('Germany', 'az'), 'Almaniya');
    });
  });
}
