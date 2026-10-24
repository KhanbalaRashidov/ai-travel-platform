import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/models.dart';
import '../../../i18n/countries.dart';
import '../../../state/providers.dart';
import '../../../state/settings_controller.dart';

/// City/airport search field (real coords via GET /data/places). Debounced; shows
/// an inline result list. Calls [onPick] with the chosen place.
class CityPicker extends ConsumerStatefulWidget {
  const CityPicker({super.key, required this.onPick, this.hint});
  final void Function(Place) onPick;
  final String? hint;

  @override
  ConsumerState<CityPicker> createState() => _CityPickerState();
}

class _CityPickerState extends ConsumerState<CityPicker> {
  final _ctrl = TextEditingController();
  Timer? _debounce;
  List<Place> _results = [];
  bool _loading = false;
  int _seq = 0;

  @override
  void dispose() {
    _debounce?.cancel();
    _ctrl.dispose();
    super.dispose();
  }

  void _onChanged(String v) {
    _debounce?.cancel();
    if (v.trim().length < 2) {
      setState(() => _results = []);
      return;
    }
    _debounce = Timer(const Duration(milliseconds: 300), () => _search(v.trim()));
  }

  Future<void> _search(String q) async {
    final locale = ref.read(localeProvider);
    final api = ref.read(travelApiProvider);
    final seq = ++_seq;
    setState(() => _loading = true);
    try {
      final res = await api.searchPlaces(q, locale: locale);
      if (seq != _seq || !mounted) return;
      setState(() {
        _results = res;
        _loading = false;
      });
    } catch (_) {
      if (seq != _seq || !mounted) return;
      setState(() {
        _results = [];
        _loading = false;
      });
    }
  }

  void _pick(Place p) {
    widget.onPick(p);
    _ctrl.clear();
    setState(() => _results = []);
    FocusScope.of(context).unfocus();
  }

  @override
  Widget build(BuildContext context) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TextField(
          controller: _ctrl,
          onChanged: _onChanged,
          textInputAction: TextInputAction.search,
          decoration: InputDecoration(
            hintText: widget.hint ?? t('plan.searchCity'),
            prefixIcon: const Icon(Icons.search, size: 20),
            suffixIcon: _loading
                ? const Padding(
                    padding: EdgeInsets.all(12),
                    child: SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2)),
                  )
                : null,
          ),
        ),
        if (_results.isNotEmpty)
          Container(
            margin: const EdgeInsets.only(top: 6),
            constraints: const BoxConstraints(maxHeight: 240),
            decoration: BoxDecoration(
              color: scheme.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: scheme.outline),
            ),
            child: ListView.separated(
              shrinkWrap: true,
              padding: EdgeInsets.zero,
              itemCount: _results.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, i) {
                final p = _results[i];
                final isAirport = p.kind == 'airport';
                return ListTile(
                  dense: true,
                  leading: Icon(
                      isAirport ? Icons.local_airport : Icons.location_city,
                      size: 20),
                  title: Text(p.name),
                  subtitle: Text(
                      [p.country, if (isAirport) t('res.airport'), if (p.code.isNotEmpty) p.code]
                          .where((e) => e.isNotEmpty)
                          .join(' · ')),
                  onTap: () => _pick(p),
                );
              },
            ),
          ),
      ],
    );
  }
}

/// Country search field (localized, offline dictionary). Calls [onPick] with the
/// chosen ISO code + localized name.
class CountryPicker extends ConsumerStatefulWidget {
  const CountryPicker({super.key, required this.onPick, this.hint});
  final void Function(CountryOption) onPick;
  final String? hint;

  @override
  ConsumerState<CountryPicker> createState() => _CountryPickerState();
}

class _CountryPickerState extends ConsumerState<CountryPicker> {
  final _ctrl = TextEditingController();
  List<CountryOption> _results = [];

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  void _onChanged(String v) {
    final locale = ref.read(localeProvider);
    setState(() => _results = searchCountries(v, locale));
  }

  void _pick(CountryOption o) {
    widget.onPick(o);
    _ctrl.clear();
    setState(() => _results = []);
    FocusScope.of(context).unfocus();
  }

  @override
  Widget build(BuildContext context) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TextField(
          controller: _ctrl,
          onChanged: _onChanged,
          textInputAction: TextInputAction.done,
          decoration: InputDecoration(
            hintText: widget.hint ?? t('country.placeholder'),
            prefixIcon: const Icon(Icons.public, size: 20),
          ),
        ),
        if (_results.isNotEmpty)
          Container(
            margin: const EdgeInsets.only(top: 6),
            constraints: const BoxConstraints(maxHeight: 240),
            decoration: BoxDecoration(
              color: scheme.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: scheme.outline),
            ),
            child: ListView.separated(
              shrinkWrap: true,
              padding: EdgeInsets.zero,
              itemCount: _results.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, i) {
                final o = _results[i];
                return ListTile(
                  dense: true,
                  leading: const Icon(Icons.flag_outlined, size: 20),
                  title: Text(o.name),
                  trailing: Text(o.code,
                      style: TextStyle(
                          color: scheme.onSurface.withValues(alpha: 0.5),
                          fontSize: 12)),
                  onTap: () => _pick(o),
                );
              },
            ),
          ),
      ],
    );
  }
}
