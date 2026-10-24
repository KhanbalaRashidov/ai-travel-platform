import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/models.dart';
import '../../../state/providers.dart';
import '../../../state/settings_controller.dart';
import '../../../widgets/common.dart';
import '../../../widgets/links.dart';
import '../roadmap_state.dart';

/// AI day-by-day roadmap: per-city cards (nights, summary, best season) and each
/// day's POIs (type, free/paid, ~price, open-in-Maps), with remove and an "other
/// places" adder. "Every day full" and proximity grouping are the server's job;
/// this shows and lightly edits the result. Regenerate re-runs for the current
/// cities (and clears the nights-changed hint, D9).
class RoadmapPanel extends ConsumerWidget {
  const RoadmapPanel({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final rs = ref.watch(roadmapControllerProvider);
    final ctrl = ref.read(roadmapControllerProvider.notifier);
    final scheme = Theme.of(context).colorScheme;
    final r = rs.roadmap;

    return SectionCard(
      title: t('ai.itin.title').toUpperCase(),
      trailing: TextButton.icon(
        onPressed: rs.loading ? null : ctrl.generate,
        icon: rs.loading
            ? const SizedBox(
                width: 14,
                height: 14,
                child: CircularProgressIndicator(strokeWidth: 2))
            : const Icon(Icons.refresh, size: 16),
        label: Text(rs.loading ? t('ai.itin.loading') : t('ai.itin.regen')),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (rs.staleNights)
            Container(
              margin: const EdgeInsets.only(bottom: 10),
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: const Color(0xFFF59E0B).withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(10),
              ),
              child: Row(
                children: [
                  const Icon(Icons.info_outline,
                      size: 16, color: Color(0xFFB45309)),
                  const SizedBox(width: 8),
                  Expanded(
                      child: Text(t('ai.itin.regenHint'),
                          style: const TextStyle(fontSize: 12))),
                ],
              ),
            ),
          if (rs.error != null)
            _InlineError(messageKey: rs.error!, onRetry: ctrl.generate)
          else if (r == null && !rs.loading)
            Text(t('ai.itin.empty'),
                style: TextStyle(
                    fontSize: 13,
                    color: scheme.onSurface.withValues(alpha: 0.6)))
          else if (r != null)
            ..._cities(context, ref, r, ctrl),
          if (rs.loading && r == null)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 16),
              child: Center(child: CircularProgressIndicator()),
            ),
        ],
      ),
    );
  }

  List<Widget> _cities(BuildContext context, WidgetRef ref, Roadmap r,
      RoadmapController ctrl) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;

    // Flat day index map so edits target the right day.
    final out = <Widget>[];
    for (final city in r.cities) {
      final dayEntries = <({int index, RoadmapDay day})>[];
      for (var i = 0; i < r.days.length; i++) {
        if (r.days[i].city.toLowerCase() == city.city.toLowerCase()) {
          dayEntries.add((index: i, day: r.days[i]));
        }
      }
      out.add(Padding(
        padding: const EdgeInsets.only(top: 6, bottom: 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(city.city,
                      style: const TextStyle(
                          fontWeight: FontWeight.w800, fontSize: 16)),
                ),
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                  decoration: BoxDecoration(
                    color: scheme.primary.withValues(alpha: 0.12),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(t('res.nights', {'n': city.nights}),
                      style: TextStyle(
                          fontSize: 11,
                          fontWeight: FontWeight.w700,
                          color: scheme.primary)),
                ),
              ],
            ),
            if ((city.bestSeason ?? '').isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 3),
                child: Text('${t('res.season')}: ${city.bestSeason}',
                    style: TextStyle(
                        fontSize: 12,
                        color: scheme.onSurface.withValues(alpha: 0.6))),
              ),
            if ((city.summary ?? '').isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(city.summary!,
                    style: const TextStyle(fontSize: 13, height: 1.35)),
              ),
            const SizedBox(height: 8),
            for (final e in dayEntries)
              _DayBlock(index: e.index, day: e.day, city: city.city),
          ],
        ),
      ));
    }
    return out;
  }
}

class _DayBlock extends ConsumerWidget {
  const _DayBlock(
      {required this.index, required this.day, required this.city});
  final int index;
  final RoadmapDay day;
  final String city;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final ctrl = ref.read(roadmapControllerProvider.notifier);
    final scheme = Theme.of(context).colorScheme;

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: scheme.surfaceContainerHighest.withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: scheme.outline),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            day.title?.isNotEmpty == true
                ? '${t('res.day', {'n': day.day})} · ${day.title}'
                : t('res.day', {'n': day.day}),
            style: const TextStyle(fontWeight: FontWeight.w700, fontSize: 13),
          ),
          const SizedBox(height: 6),
          for (var p = 0; p < day.pois.length; p++)
            _poiRow(context, day.pois[p], () => ctrl.removePoi(index, p), t, scheme),
          const SizedBox(height: 4),
          Align(
            alignment: Alignment.centerLeft,
            child: TextButton.icon(
              style: TextButton.styleFrom(
                  visualDensity: VisualDensity.compact,
                  padding: const EdgeInsets.symmetric(horizontal: 6)),
              onPressed: () => _showAddSheet(context, ref),
              icon: const Icon(Icons.add, size: 16),
              label: Text(t('roadmap.otherPlaces'),
                  style: const TextStyle(fontSize: 12)),
            ),
          ),
        ],
      ),
    );
  }

  Widget _poiRow(BuildContext context, RoadmapPoi poi, VoidCallback onRemove,
      String Function(String, [Map<String, Object>?]) t, ColorScheme scheme) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(top: 2, right: 8),
            child: Icon(Icons.place, size: 14, color: scheme.secondary),
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(poi.name,
                    style: const TextStyle(
                        fontSize: 13, fontWeight: FontWeight.w600)),
                Wrap(
                  spacing: 6,
                  children: [
                    if ((poi.type ?? '').isNotEmpty)
                      Text(poi.type!,
                          style: TextStyle(
                              fontSize: 11,
                              color: scheme.onSurface.withValues(alpha: 0.6))),
                    if (poi.free)
                      Text(t('res.free'),
                          style: const TextStyle(
                              fontSize: 11,
                              color: Color(0xFF16A34A),
                              fontWeight: FontWeight.w600))
                    else if ((poi.price ?? '').isNotEmpty)
                      Text(poi.price!,
                          style: TextStyle(
                              fontSize: 11,
                              color: scheme.onSurface.withValues(alpha: 0.7))),
                  ],
                ),
              ],
            ),
          ),
          InkWell(
            onTap: () => openUrl(context,
                mapsUrl(name: poi.name, lat: poi.lat, lng: poi.lng)),
            child: Padding(
              padding: const EdgeInsets.all(4),
              child: Icon(Icons.map_outlined,
                  size: 15, color: scheme.onSurface.withValues(alpha: 0.5)),
            ),
          ),
          InkWell(
            onTap: onRemove,
            child: Padding(
              padding: const EdgeInsets.all(4),
              child: Icon(Icons.close,
                  size: 15, color: scheme.onSurface.withValues(alpha: 0.5)),
            ),
          ),
        ],
      ),
    );
  }

  void _showAddSheet(BuildContext context, WidgetRef ref) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (_) => _AddPlacesSheet(city: city, dayIndex: index),
    );
  }
}

/// Bottom sheet listing real POIs for the city (OSM + AI) to add to a day.
class _AddPlacesSheet extends ConsumerStatefulWidget {
  const _AddPlacesSheet({required this.city, required this.dayIndex});
  final String city;
  final int dayIndex;

  @override
  ConsumerState<_AddPlacesSheet> createState() => _AddPlacesSheetState();
}

class _AddPlacesSheetState extends ConsumerState<_AddPlacesSheet> {
  List<PoiItem> _items = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final items =
          await ref.read(travelApiProvider).fetchPOIs(widget.city, limit: 16);
      if (!mounted) return;
      setState(() {
        _items = items;
        _loading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;
    return SafeArea(
      child: ConstrainedBox(
        constraints: BoxConstraints(
            maxHeight: MediaQuery.of(context).size.height * 0.7),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('${t('roadmap.otherPlaces')} · ${widget.city}',
                      style: const TextStyle(
                          fontWeight: FontWeight.w800, fontSize: 16)),
                  Text(t('poi.sourceAI'),
                      style: TextStyle(
                          fontSize: 11,
                          color: scheme.onSurface.withValues(alpha: 0.6))),
                ],
              ),
            ),
            if (_loading)
              const Padding(
                padding: EdgeInsets.all(24),
                child: Center(child: CircularProgressIndicator()),
              )
            else if (_items.isEmpty)
              Padding(
                padding: const EdgeInsets.all(24),
                child: Text(t('roadmap.noMore')),
              )
            else
              Flexible(
                child: ListView.separated(
                  shrinkWrap: true,
                  itemCount: _items.length,
                  separatorBuilder: (_, __) => const Divider(height: 1),
                  itemBuilder: (context, i) {
                    final it = _items[i];
                    return ListTile(
                      title: Text(it.name),
                      subtitle: Text([
                        if (it.category.isNotEmpty) it.category,
                        if (it.rating != null) '★ ${it.rating!.toStringAsFixed(1)}',
                      ].join(' · ')),
                      trailing: const Icon(Icons.add_circle_outline),
                      onTap: () {
                        ref.read(roadmapControllerProvider.notifier).addPoi(
                              widget.dayIndex,
                              RoadmapPoi(
                                name: it.name,
                                type: it.category,
                                free: it.free ?? false,
                                price: it.price,
                                lat: it.lat,
                                lng: it.lng,
                              ),
                            );
                        Navigator.of(context).pop();
                      },
                    );
                  },
                ),
              ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
  }
}

class _InlineError extends ConsumerWidget {
  const _InlineError({required this.messageKey, required this.onRetry});
  final String messageKey;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;
    return Row(
      children: [
        Icon(Icons.error_outline, size: 18, color: scheme.error),
        const SizedBox(width: 8),
        Expanded(child: Text(t(messageKey))),
        TextButton(onPressed: onRetry, child: Text(t('ai.retry'))),
      ],
    );
  }
}
