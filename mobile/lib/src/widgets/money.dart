import 'package:intl/intl.dart';

const Map<String, String> _symbols = {
  'EUR': '€',
  'USD': '\$',
  'AZN': '₼',
  'TRY': '₺',
  'RUB': '₽',
  'GBP': '£',
};

/// Format a money amount with a grouped number and the currency symbol/code.
/// Whole-ish amounts drop the decimals; small ones keep two. Never invents a
/// value — it only formats what the API returned.
String money(num amount, String currency) {
  final sym = _symbols[currency.toUpperCase()] ?? '';
  final abs = amount.abs();
  final decimals = abs >= 100 || amount == amount.roundToDouble() ? 0 : 2;
  final nf = NumberFormat('#,##0${decimals == 2 ? '.00' : ''}', 'en_US');
  final text = nf.format(amount);
  if (sym.isNotEmpty) return '$sym$text';
  return '$text ${currency.toUpperCase()}';
}

const List<String> _monthAbbr = [
  'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
  'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
];

/// Compact, locale-agnostic date label (e.g. "5 Jun 2026").
String fmtDateDisplay(DateTime d) =>
    '${d.day} ${_monthAbbr[d.month - 1]} ${d.year}';
