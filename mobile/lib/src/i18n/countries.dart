// Localized country autocomplete data — the Dart port of web/lib/countries.ts.
// The web relied on the browser's Intl.DisplayNames for EN/RU/TR and a hand dict
// for AZ; Dart has no region display-name API, so all four languages ship as
// dictionaries here. A country keeps its ISO code (canonical) for the AI and its
// local name for display. Search folds diacritics and matches local name, English
// name, or code, so typing "almaniy" (AZ) finds Almaniya.

/// Tourism-oriented ISO 3166-1 alpha-2 set (same list as the web).
const List<String> kCountryCodes = [
  'AL', 'AD', 'AM', 'AR', 'AU', 'AT', 'AZ', 'BH', 'BD', 'BY', 'BE', 'BA', 'BR',
  'BG', 'KH', 'CA', 'CL', 'CN', 'CO', 'CR', 'HR', 'CU', 'CY', 'CZ', 'DK', 'DO',
  'EC', 'EG', 'EE', 'FI', 'FR', 'GE', 'DE', 'GR', 'GT', 'HK', 'HU', 'IS', 'IN',
  'ID', 'IR', 'IQ', 'IE', 'IL', 'IT', 'JP', 'JO', 'KZ', 'KE', 'KW', 'KG', 'LA',
  'LV', 'LB', 'LT', 'LU', 'MO', 'MY', 'MV', 'MT', 'MX', 'MD', 'MC', 'ME', 'MA',
  'NP', 'NL', 'NZ', 'MK', 'NO', 'OM', 'PK', 'PA', 'PE', 'PH', 'PL', 'PT', 'QA',
  'RO', 'RU', 'SA', 'RS', 'SG', 'SK', 'SI', 'ZA', 'KR', 'ES', 'LK', 'SE', 'CH',
  'TW', 'TJ', 'TH', 'TN', 'TR', 'TM', 'UA', 'AE', 'GB', 'US', 'UY', 'UZ', 'VN',
];

const Map<String, String> _en = {
  'AL': 'Albania', 'AD': 'Andorra', 'AM': 'Armenia', 'AR': 'Argentina',
  'AU': 'Australia', 'AT': 'Austria', 'AZ': 'Azerbaijan', 'BH': 'Bahrain',
  'BD': 'Bangladesh', 'BY': 'Belarus', 'BE': 'Belgium',
  'BA': 'Bosnia and Herzegovina', 'BR': 'Brazil', 'BG': 'Bulgaria',
  'KH': 'Cambodia', 'CA': 'Canada', 'CL': 'Chile', 'CN': 'China',
  'CO': 'Colombia', 'CR': 'Costa Rica', 'HR': 'Croatia', 'CU': 'Cuba',
  'CY': 'Cyprus', 'CZ': 'Czechia', 'DK': 'Denmark',
  'DO': 'Dominican Republic', 'EC': 'Ecuador', 'EG': 'Egypt', 'EE': 'Estonia',
  'FI': 'Finland', 'FR': 'France', 'GE': 'Georgia', 'DE': 'Germany',
  'GR': 'Greece', 'GT': 'Guatemala', 'HK': 'Hong Kong', 'HU': 'Hungary',
  'IS': 'Iceland', 'IN': 'India', 'ID': 'Indonesia', 'IR': 'Iran', 'IQ': 'Iraq',
  'IE': 'Ireland', 'IL': 'Israel', 'IT': 'Italy', 'JP': 'Japan', 'JO': 'Jordan',
  'KZ': 'Kazakhstan', 'KE': 'Kenya', 'KW': 'Kuwait', 'KG': 'Kyrgyzstan',
  'LA': 'Laos', 'LV': 'Latvia', 'LB': 'Lebanon', 'LT': 'Lithuania',
  'LU': 'Luxembourg', 'MO': 'Macao', 'MY': 'Malaysia', 'MV': 'Maldives',
  'MT': 'Malta', 'MX': 'Mexico', 'MD': 'Moldova', 'MC': 'Monaco',
  'ME': 'Montenegro', 'MA': 'Morocco', 'NP': 'Nepal', 'NL': 'Netherlands',
  'NZ': 'New Zealand', 'MK': 'North Macedonia', 'NO': 'Norway', 'OM': 'Oman',
  'PK': 'Pakistan', 'PA': 'Panama', 'PE': 'Peru', 'PH': 'Philippines',
  'PL': 'Poland', 'PT': 'Portugal', 'QA': 'Qatar', 'RO': 'Romania',
  'RU': 'Russia', 'SA': 'Saudi Arabia', 'RS': 'Serbia', 'SG': 'Singapore',
  'SK': 'Slovakia', 'SI': 'Slovenia', 'ZA': 'South Africa', 'KR': 'South Korea',
  'ES': 'Spain', 'LK': 'Sri Lanka', 'SE': 'Sweden', 'CH': 'Switzerland',
  'TW': 'Taiwan', 'TJ': 'Tajikistan', 'TH': 'Thailand', 'TN': 'Tunisia',
  'TR': 'Turkey', 'TM': 'Turkmenistan', 'UA': 'Ukraine',
  'AE': 'United Arab Emirates', 'GB': 'United Kingdom', 'US': 'United States',
  'UY': 'Uruguay', 'UZ': 'Uzbekistan', 'VN': 'Vietnam',
};

const Map<String, String> _az = {
  'AL': 'Albaniya', 'AD': 'Andorra', 'AM': 'Ermənistan', 'AR': 'Argentina',
  'AU': 'Avstraliya', 'AT': 'Avstriya', 'AZ': 'Azərbaycan', 'BH': 'Bəhreyn',
  'BD': 'Banqladeş', 'BY': 'Belarus', 'BE': 'Belçika',
  'BA': 'Bosniya və Herseqovina', 'BR': 'Braziliya', 'BG': 'Bolqarıstan',
  'KH': 'Kamboca', 'CA': 'Kanada', 'CL': 'Çili', 'CN': 'Çin',
  'CO': 'Kolumbiya', 'CR': 'Kosta-Rika', 'HR': 'Xorvatiya', 'CU': 'Kuba',
  'CY': 'Kipr', 'CZ': 'Çexiya', 'DK': 'Danimarka',
  'DO': 'Dominikan Respublikası', 'EC': 'Ekvador', 'EG': 'Misir',
  'EE': 'Estoniya', 'FI': 'Finlandiya', 'FR': 'Fransa', 'GE': 'Gürcüstan',
  'DE': 'Almaniya', 'GR': 'Yunanıstan', 'GT': 'Qvatemala', 'HK': 'Honq-Konq',
  'HU': 'Macarıstan', 'IS': 'İslandiya', 'IN': 'Hindistan', 'ID': 'İndoneziya',
  'IR': 'İran', 'IQ': 'İraq', 'IE': 'İrlandiya', 'IL': 'İsrail', 'IT': 'İtaliya',
  'JP': 'Yaponiya', 'JO': 'İordaniya', 'KZ': 'Qazaxıstan', 'KE': 'Keniya',
  'KW': 'Küveyt', 'KG': 'Qırğızıstan', 'LA': 'Laos', 'LV': 'Latviya',
  'LB': 'Livan', 'LT': 'Litva', 'LU': 'Lüksemburq', 'MO': 'Makao',
  'MY': 'Malayziya', 'MV': 'Maldiv adaları', 'MT': 'Malta', 'MX': 'Meksika',
  'MD': 'Moldova', 'MC': 'Monako', 'ME': 'Monteneqro', 'MA': 'Mərakeş',
  'NP': 'Nepal', 'NL': 'Niderland', 'NZ': 'Yeni Zelandiya',
  'MK': 'Şimali Makedoniya', 'NO': 'Norveç', 'OM': 'Oman', 'PK': 'Pakistan',
  'PA': 'Panama', 'PE': 'Peru', 'PH': 'Filippin', 'PL': 'Polşa',
  'PT': 'Portuqaliya', 'QA': 'Qatar', 'RO': 'Rumıniya', 'RU': 'Rusiya',
  'SA': 'Səudiyyə Ərəbistanı', 'RS': 'Serbiya', 'SG': 'Sinqapur',
  'SK': 'Slovakiya', 'SI': 'Sloveniya', 'ZA': 'Cənubi Afrika',
  'KR': 'Cənubi Koreya', 'ES': 'İspaniya', 'LK': 'Şri-Lanka', 'SE': 'İsveç',
  'CH': 'İsveçrə', 'TW': 'Tayvan', 'TJ': 'Tacikistan', 'TH': 'Tailand',
  'TN': 'Tunis', 'TR': 'Türkiyə', 'TM': 'Türkmənistan', 'UA': 'Ukrayna',
  'AE': 'Birləşmiş Ərəb Əmirlikləri', 'GB': 'Böyük Britaniya',
  'US': 'Amerika Birləşmiş Ştatları', 'UY': 'Uruqvay', 'UZ': 'Özbəkistan',
  'VN': 'Vyetnam',
};

const Map<String, String> _ru = {
  'AL': 'Албания', 'AD': 'Андорра', 'AM': 'Армения', 'AR': 'Аргентина',
  'AU': 'Австралия', 'AT': 'Австрия', 'AZ': 'Азербайджан', 'BH': 'Бахрейн',
  'BD': 'Бангладеш', 'BY': 'Беларусь', 'BE': 'Бельгия',
  'BA': 'Босния и Герцеговина', 'BR': 'Бразилия', 'BG': 'Болгария',
  'KH': 'Камбоджа', 'CA': 'Канада', 'CL': 'Чили', 'CN': 'Китай',
  'CO': 'Колумбия', 'CR': 'Коста-Рика', 'HR': 'Хорватия', 'CU': 'Куба',
  'CY': 'Кипр', 'CZ': 'Чехия', 'DK': 'Дания',
  'DO': 'Доминиканская Республика', 'EC': 'Эквадор', 'EG': 'Египет',
  'EE': 'Эстония', 'FI': 'Финляндия', 'FR': 'Франция', 'GE': 'Грузия',
  'DE': 'Германия', 'GR': 'Греция', 'GT': 'Гватемала', 'HK': 'Гонконг',
  'HU': 'Венгрия', 'IS': 'Исландия', 'IN': 'Индия', 'ID': 'Индонезия',
  'IR': 'Иран', 'IQ': 'Ирак', 'IE': 'Ирландия', 'IL': 'Израиль', 'IT': 'Италия',
  'JP': 'Япония', 'JO': 'Иордания', 'KZ': 'Казахстан', 'KE': 'Кения',
  'KW': 'Кувейт', 'KG': 'Киргизия', 'LA': 'Лаос', 'LV': 'Латвия',
  'LB': 'Ливан', 'LT': 'Литва', 'LU': 'Люксембург', 'MO': 'Макао',
  'MY': 'Малайзия', 'MV': 'Мальдивы', 'MT': 'Мальта', 'MX': 'Мексика',
  'MD': 'Молдова', 'MC': 'Монако', 'ME': 'Черногория', 'MA': 'Марокко',
  'NP': 'Непал', 'NL': 'Нидерланды', 'NZ': 'Новая Зеландия',
  'MK': 'Северная Македония', 'NO': 'Норвегия', 'OM': 'Оман', 'PK': 'Пакистан',
  'PA': 'Панама', 'PE': 'Перу', 'PH': 'Филиппины', 'PL': 'Польша',
  'PT': 'Португалия', 'QA': 'Катар', 'RO': 'Румыния', 'RU': 'Россия',
  'SA': 'Саудовская Аравия', 'RS': 'Сербия', 'SG': 'Сингапур',
  'SK': 'Словакия', 'SI': 'Словения', 'ZA': 'ЮАР', 'KR': 'Южная Корея',
  'ES': 'Испания', 'LK': 'Шри-Ланка', 'SE': 'Швеция', 'CH': 'Швейцария',
  'TW': 'Тайвань', 'TJ': 'Таджикистан', 'TH': 'Таиланд', 'TN': 'Тунис',
  'TR': 'Турция', 'TM': 'Туркменистан', 'UA': 'Украина', 'AE': 'ОАЭ',
  'GB': 'Великобритания', 'US': 'США', 'UY': 'Уругвай', 'UZ': 'Узбекистан',
  'VN': 'Вьетнам',
};

const Map<String, String> _tr = {
  'AL': 'Arnavutluk', 'AD': 'Andorra', 'AM': 'Ermenistan', 'AR': 'Arjantin',
  'AU': 'Avustralya', 'AT': 'Avusturya', 'AZ': 'Azerbaycan', 'BH': 'Bahreyn',
  'BD': 'Bangladeş', 'BY': 'Belarus', 'BE': 'Belçika', 'BA': 'Bosna-Hersek',
  'BR': 'Brezilya', 'BG': 'Bulgaristan', 'KH': 'Kamboçya', 'CA': 'Kanada',
  'CL': 'Şili', 'CN': 'Çin', 'CO': 'Kolombiya', 'CR': 'Kosta Rika',
  'HR': 'Hırvatistan', 'CU': 'Küba', 'CY': 'Kıbrıs', 'CZ': 'Çekya',
  'DK': 'Danimarka', 'DO': 'Dominik Cumhuriyeti', 'EC': 'Ekvador', 'EG': 'Mısır',
  'EE': 'Estonya', 'FI': 'Finlandiya', 'FR': 'Fransa', 'GE': 'Gürcistan',
  'DE': 'Almanya', 'GR': 'Yunanistan', 'GT': 'Guatemala', 'HK': 'Hong Kong',
  'HU': 'Macaristan', 'IS': 'İzlanda', 'IN': 'Hindistan', 'ID': 'Endonezya',
  'IR': 'İran', 'IQ': 'Irak', 'IE': 'İrlanda', 'IL': 'İsrail', 'IT': 'İtalya',
  'JP': 'Japonya', 'JO': 'Ürdün', 'KZ': 'Kazakistan', 'KE': 'Kenya',
  'KW': 'Kuveyt', 'KG': 'Kırgızistan', 'LA': 'Laos', 'LV': 'Letonya',
  'LB': 'Lübnan', 'LT': 'Litvanya', 'LU': 'Lüksemburg', 'MO': 'Makao',
  'MY': 'Malezya', 'MV': 'Maldivler', 'MT': 'Malta', 'MX': 'Meksika',
  'MD': 'Moldova', 'MC': 'Monako', 'ME': 'Karadağ', 'MA': 'Fas', 'NP': 'Nepal',
  'NL': 'Hollanda', 'NZ': 'Yeni Zelanda', 'MK': 'Kuzey Makedonya',
  'NO': 'Norveç', 'OM': 'Umman', 'PK': 'Pakistan', 'PA': 'Panama', 'PE': 'Peru',
  'PH': 'Filipinler', 'PL': 'Polonya', 'PT': 'Portekiz', 'QA': 'Katar',
  'RO': 'Romanya', 'RU': 'Rusya', 'SA': 'Suudi Arabistan', 'RS': 'Sırbistan',
  'SG': 'Singapur', 'SK': 'Slovakya', 'SI': 'Slovenya', 'ZA': 'Güney Afrika',
  'KR': 'Güney Kore', 'ES': 'İspanya', 'LK': 'Sri Lanka', 'SE': 'İsveç',
  'CH': 'İsviçre', 'TW': 'Tayvan', 'TJ': 'Tacikistan', 'TH': 'Tayland',
  'TN': 'Tunus', 'TR': 'Türkiye', 'TM': 'Türkmenistan', 'UA': 'Ukrayna',
  'AE': 'Birleşik Arap Emirlikleri', 'GB': 'Birleşik Krallık',
  'US': 'Amerika Birleşik Devletleri', 'UY': 'Uruguay', 'UZ': 'Özbekistan',
  'VN': 'Vietnam',
};

const Map<String, Map<String, String>> _dicts = {
  'en': _en,
  'az': _az,
  'ru': _ru,
  'tr': _tr,
};

/// Localized country name; falls back to English, then the raw code.
String countryName(String code, String locale) {
  final c = code.toUpperCase();
  return _dicts[locale]?[c] ?? _en[c] ?? c;
}

/// English name — the stable canonical form sent to the AI so suggestions
/// resolve regardless of UI language.
String canonicalName(String code) => _en[code.toUpperCase()] ?? code;

class CountryOption {
  const CountryOption(this.code, this.name);
  final String code;
  final String name; // localized
}

/// Fold for accent-insensitive matching: lowercase + a targeted map of the
/// Latin diacritics used by our languages (Cyrillic just lowercases).
String _fold(String s) {
  final lower = s.toLowerCase().trim();
  const map = {
    'ə': 'e', 'ü': 'u', 'ö': 'o', 'ç': 'c', 'ş': 's', 'ğ': 'g', 'ı': 'i',
    'i̇': 'i', 'â': 'a', 'î': 'i', 'û': 'u', 'é': 'e', 'è': 'e', 'ê': 'e',
    'á': 'a', 'à': 'a', 'ä': 'a', 'å': 'a', 'ã': 'a', 'í': 'i', 'ï': 'i',
    'ó': 'o', 'ô': 'o', 'õ': 'o', 'ú': 'u', 'ñ': 'n', 'ß': 's', 'ý': 'y',
  };
  final sb = StringBuffer();
  for (final ch in lower.split('')) {
    sb.write(map[ch] ?? ch);
  }
  return sb.toString();
}

/// Localized matches for a query (local name, English name, or code), prefix
/// matches first. Empty for a blank query.
List<CountryOption> searchCountries(String query, String locale,
    {int limit = 10}) {
  final nq = _fold(query);
  if (nq.isEmpty) return [];
  final scored = <({CountryOption opt, int score})>[];
  for (final code in kCountryCodes) {
    final local = countryName(code, locale);
    final en = _en[code] ?? code;
    final nl = _fold(local);
    final ne = _fold(en);
    var score = -1;
    if (nl.startsWith(nq) || ne.startsWith(nq) || code.toLowerCase() == nq) {
      score = 2;
    } else if (nl.contains(nq) || ne.contains(nq)) {
      score = 1;
    }
    if (score >= 0) scored.add((opt: CountryOption(code, local), score: score));
  }
  scored.sort((a, b) =>
      b.score != a.score ? b.score - a.score : a.opt.name.compareTo(b.opt.name));
  return scored.take(limit).map((s) => s.opt).toList();
}

const List<String> _allLocales = ['az', 'en', 'ru', 'tr'];

/// Reverse-map a country name (any of the 4 languages, English, or a code) to
/// its code, so an AI panel heading can be re-localized. "" when unknown.
String codeByName(String name) {
  final n = _fold(name);
  if (n.isEmpty) return '';
  for (final code in kCountryCodes) {
    if (code.toLowerCase() == n) return code;
    for (final loc in _allLocales) {
      if (_fold(countryName(code, loc)) == n) return code;
    }
  }
  return '';
}

/// Render `name` in `locale` when it maps to a known country, else unchanged.
String localizeCountry(String name, String locale) {
  final code = codeByName(name);
  return code.isNotEmpty ? countryName(code, locale) : name;
}
