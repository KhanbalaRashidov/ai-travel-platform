import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/models.dart';
import '../../../i18n/countries.dart';
import '../../../state/providers.dart';
import '../../../state/settings_controller.dart';
import '../../../widgets/common.dart';
import '../plan_state.dart';
import 'pickers.dart';

/// Country-mode / minimal-mode AI suggestions. Type a country (or, in minimal
/// mode, just a budget) -> AI-ranked cities -> pick -> resolve coords and add to
/// the trip. Ratings render only when the API actually returned one (guardrail).
class SuggestPanel extends ConsumerStatefulWidget {
  const SuggestPanel({super.key, required this.minimal});
  final bool minimal;

  @override
  ConsumerState<SuggestPanel> createState() => _SuggestPanelState();
}

class _SuggestPanelState extends ConsumerState<SuggestPanel> {
  final List<CountryOption> _countries = [];
  DestinationSuggestions? _result;
  bool _loading = false;
  String? _error;
  final Set<String> _adding = {};
  final Set<String> _added = {};

  Future<void> _suggest() async {
    final locale = ref.read(localeProvider);
    final st = ref.read(planControllerProvider);
    final api = ref.read(travelApiProvider);
    if (!widget.minimal && _countries.isEmpty) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final res = await api.suggestDestinations(
        _countries.map((c) => canonicalName(c.code)).toList(),
        budgetEur: st.budgetEur ?? 0,
        currency: st.currency,
        locale: locale,
      );
      if (!mounted) return;
      setState(() {
        _result = res;
        _loading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _error = ref.read(tProvider)('ai.error');
        _loading = false;
      });
    }
  }

  Future<void> _add(SuggestedCity c) async {
    final locale = ref.read(localeProvider);
    final api = ref.read(travelApiProvider);
    final ctrl = ref.read(planControllerProvider.notifier);
    setState(() => _adding.add(c.city));
    try {
      final places = await api.searchPlaces(c.city, locale: locale);
      final p = places.isNotEmpty ? places.first : null;
      ctrl.addDestination(Destination(
        name: c.city,
        lat: p?.lat ?? 0,
        lng: p?.lng ?? 0,
      ));
      if (!mounted) return;
      setState(() {
        _adding.remove(c.city);
        _added.add(c.city);
      });
    } catch (_) {
      if (!mounted) return;
      setState(() => _adding.remove(c.city));
    }
  }

  @override
  Widget build(BuildContext context) {
    final t = ref.watch(tProvider);
    final locale = ref.watch(localeProvider);
    final scheme = Theme.of(context).colorScheme;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(widget.minimal ? t('plan.subtitle') : t('country.hint'),
            style: TextStyle(
                fontSize: 12.5,
                color: scheme.onSurface.withValues(alpha: 0.6))),
        const SizedBox(height: 10),
        if (!widget.minimal) ...[
          CountryPicker(onPick: (o) {
            if (!_countries.any((c) => c.code == o.code)) {
              setState(() => _countries.add(o));
            }
          }),
          if (_countries.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 10),
              child: Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final c in _countries)
                    Chip(
                      label: Text(countryName(c.code, locale)),
                      onDeleted: () =>
                          setState(() => _countries.remove(c)),
                    ),
                ],
              ),
            ),
          const SizedBox(height: 10),
        ],
        FilledButton.icon(
          onPressed: _loading ? null : _suggest,
          icon: _loading
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(
                      strokeWidth: 2, color: Colors.white))
              : const Icon(Icons.auto_awesome, size: 18),
          label: Text(_loading ? t('country.loading') : t('country.suggest')),
        ),
        if (_error != null)
          Padding(
            padding: const EdgeInsets.only(top: 10),
            child: Text(_error!, style: TextStyle(color: scheme.error)),
          ),
        if (_result?.budgetAdvice != null &&
            _result!.budgetAdvice!.isNotEmpty) ...[
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: scheme.surfaceContainerHighest.withValues(alpha: 0.5),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Icon(Icons.savings_outlined, size: 18),
                const SizedBox(width: 8),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(t('country.budgetAdvice'),
                          style: const TextStyle(
                              fontWeight: FontWeight.w700, fontSize: 12)),
                      const SizedBox(height: 2),
                      Text(_result!.budgetAdvice!,
                          style: const TextStyle(fontSize: 13, height: 1.35)),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
        if (_result != null) ...[
          const SizedBox(height: 12),
          for (final cs in _result!.countries) ...[
            Padding(
              padding: const EdgeInsets.only(top: 6, bottom: 6),
              child: Text(localizeCountry(cs.country, locale),
                  style: const TextStyle(
                      fontWeight: FontWeight.w700, fontSize: 14)),
            ),
            for (final city in cs.cities)
              _cityTile(context, city, t, scheme),
          ],
        ],
      ],
    );
  }

  Widget _cityTile(BuildContext context, SuggestedCity c,
      String Function(String, [Map<String, Object>?]) t, ColorScheme scheme) {
    final isAdded = _added.contains(c.city);
    final isAdding = _adding.contains(c.city);
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: scheme.outline),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Flexible(
                      child: Text(c.city,
                          style: const TextStyle(
                              fontWeight: FontWeight.w700, fontSize: 15)),
                    ),
                    // Rating ONLY if a real source provided it (no fake stars).
                    if (c.rating != null) ...[
                      const SizedBox(width: 8),
                      Icon(Icons.star, size: 13, color: Colors.amber[600]),
                      const SizedBox(width: 2),
                      Text(c.rating!.toStringAsFixed(1),
                          style: const TextStyle(fontSize: 12)),
                    ],
                  ],
                ),
                if (c.reason != null && c.reason!.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(top: 3),
                    child: Text(c.reason!,
                        style: TextStyle(
                            fontSize: 12.5,
                            height: 1.3,
                            color: scheme.onSurface.withValues(alpha: 0.7))),
                  ),
                if (c.bestSeason != null && c.bestSeason!.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(top: 3),
                    child: Text('${t('res.season')}: ${c.bestSeason}',
                        style: TextStyle(
                            fontSize: 11.5,
                            color: scheme.onSurface.withValues(alpha: 0.6))),
                  ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          isAdded
              ? Chip(
                  visualDensity: VisualDensity.compact,
                  label: Text(t('country.added')),
                  backgroundColor: scheme.primary.withValues(alpha: 0.12),
                )
              : FilledButton.tonal(
                  onPressed: isAdding ? null : () => _add(c),
                  child: Text(isAdding ? t('country.adding') : t('country.add')),
                ),
        ],
      ),
    );
  }
}
