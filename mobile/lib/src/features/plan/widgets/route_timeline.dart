import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/models.dart';
import '../../../state/settings_controller.dart';
import '../../../theme/app_theme.dart';
import '../../../widgets/common.dart';
import '../../../widgets/links.dart';
import '../../../widgets/money.dart';
import '../plan_state.dart';

const _modes = ['cheapest', 'flight', 'train', 'bus', 'ferry', 'car'];

/// Station-style vertical timeline of the priced route. Each leg shows its mode,
/// price and fare detail, a per-leg mode override (live re-price), and the source
/// comparison with prefilled booking links. The round-trip return leg (last ->
/// origin) is just another leg in the list, so it renders and totals naturally.
class RouteTimeline extends ConsumerWidget {
  const RouteTimeline({super.key, required this.q});
  final PlanQuote q;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final legs = q.legs;
    if (legs.isEmpty) return const SizedBox.shrink();

    // City chain = first leg's origin, then each leg's destination.
    final chain = <String>[legs.first.from, for (final l in legs) l.to];

    return SectionCard(
      title: t('res.route').toUpperCase(),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (var i = 0; i < chain.length; i++) ...[
            _Station(name: chain[i], first: i == 0, last: i == chain.length - 1),
            if (i < legs.length) _LegTile(leg: legs[i]),
          ],
        ],
      ),
    );
  }
}

class _Station extends StatelessWidget {
  const _Station({required this.name, required this.first, required this.last});
  final String name;
  final bool first;
  final bool last;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return IntrinsicHeight(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          SizedBox(
            width: 28,
            child: Column(
              children: [
                Container(
                  width: 14,
                  height: 14,
                  decoration: BoxDecoration(
                    gradient: AppColors.brandGradient,
                    shape: BoxShape.circle,
                    border: Border.all(color: scheme.surface, width: 2),
                  ),
                ),
                if (!last)
                  Expanded(
                    child: Container(
                        width: 2,
                        color: scheme.outline.withValues(alpha: 0.7)),
                  ),
              ],
            ),
          ),
          const SizedBox(width: 10),
          Padding(
            padding: const EdgeInsets.only(bottom: 10, top: 1),
            child: Text(name,
                style: const TextStyle(
                    fontWeight: FontWeight.w800, fontSize: 15)),
          ),
        ],
      ),
    );
  }
}

class _LegTile extends ConsumerWidget {
  const _LegTile({required this.leg});
  final PlanLeg leg;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final state = ref.watch(planControllerProvider);
    final ctrl = ref.read(planControllerProvider.notifier);
    final scheme = Theme.of(context).colorScheme;
    final cur = state.quote?.currency ?? state.currency;
    final override = state.legModes['${leg.from}|${leg.to}'] ?? 'cheapest';

    final details = <String>[
      if ((leg.operator ?? '').isNotEmpty) leg.operator!,
      if ((leg.depart ?? '').isNotEmpty) leg.depart!,
      if ((leg.duration ?? '').isNotEmpty) leg.duration!,
      if ((leg.stops ?? '').isNotEmpty) leg.stops!,
      if ((leg.baggage ?? '').isNotEmpty) '${t('res.baggage')}: ${leg.baggage}',
    ];

    return Padding(
      padding: const EdgeInsets.only(left: 28, bottom: 12),
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: scheme.surfaceContainerHighest.withValues(alpha: 0.4),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: scheme.outline),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(modeIcon(leg.mode), size: 18, color: scheme.primary),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(t('mode.${leg.mode}'),
                      style: const TextStyle(fontWeight: FontWeight.w600)),
                ),
                if (state.loading)
                  Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: Text(t('res.updating'),
                        style: TextStyle(
                            fontSize: 11,
                            color: scheme.onSurface.withValues(alpha: 0.5))),
                  ),
                Text(
                  leg.price > 0 ? money(leg.price, cur) : t('res.transfer'),
                  style: TextStyle(
                      fontWeight: FontWeight.w800, color: scheme.primary),
                ),
              ],
            ),
            if (leg.modeUnavailable == true)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(
                  t('res.modeUnavailable',
                      {'m': t('mode.${leg.requestedMode ?? leg.mode}')}),
                  style: TextStyle(fontSize: 11, color: scheme.error),
                ),
              ),
            if (details.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Wrap(
                  spacing: 6,
                  runSpacing: 6,
                  children: [for (final d in details) _chip(context, d)],
                ),
              )
            else if (leg.estimated)
              Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Text(t('res.estimatedLine'),
                    style: TextStyle(
                        fontSize: 11.5,
                        fontStyle: FontStyle.italic,
                        color: scheme.onSurface.withValues(alpha: 0.6))),
              ),
            // Source + date (with the chosen booking link).
            if (leg.source.isNotEmpty || (leg.url ?? '').isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Row(
                  children: [
                    Flexible(
                      child: Text(
                        [
                          if (leg.source.isNotEmpty)
                            '${t('res.source')}: ${leg.source}',
                          if ((leg.fetchedAt ?? '').isNotEmpty)
                            '${t('res.asOf')} ${leg.fetchedAt}',
                        ].join('  ·  '),
                        style: TextStyle(
                            fontSize: 11,
                            color: scheme.onSurface.withValues(alpha: 0.55)),
                      ),
                    ),
                    if ((leg.url ?? '').isNotEmpty)
                      TextButton.icon(
                        style: TextButton.styleFrom(
                            visualDensity: VisualDensity.compact,
                            padding: const EdgeInsets.symmetric(horizontal: 6)),
                        onPressed: () => openUrl(context, leg.url),
                        icon: const Icon(Icons.open_in_new, size: 14),
                        label: Text(t('res.searchInfo')),
                      ),
                  ],
                ),
              ),
            // Source comparison (prefilled deep-links).
            if ((leg.alternatives ?? []).isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(t('res.alt'),
                  style: TextStyle(
                      fontSize: 11,
                      fontWeight: FontWeight.w700,
                      color: scheme.onSurface.withValues(alpha: 0.6))),
              const SizedBox(height: 6),
              Wrap(
                spacing: 6,
                runSpacing: 6,
                children: [
                  for (final a in leg.alternatives!)
                    _AltChip(alt: a, currency: cur),
                ],
              ),
            ],
            // Per-leg mode override (live re-price).
            const SizedBox(height: 10),
            Row(
              children: [
                Icon(Icons.swap_horiz,
                    size: 16, color: scheme.onSurface.withValues(alpha: 0.5)),
                const SizedBox(width: 6),
                Expanded(
                  child: Wrap(
                    spacing: 6,
                    children: [
                      for (final m in _modes)
                        ChoiceChip(
                          label: Text(t('mode.$m'),
                              style: const TextStyle(fontSize: 11)),
                          selected: m == override,
                          visualDensity: VisualDensity.compact,
                          onSelected: (_) {
                            if (m != override) {
                              ctrl.setLegMode(leg.from, leg.to, m);
                            }
                          },
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _chip(BuildContext context, String text) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: scheme.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: scheme.outline),
      ),
      child: Text(text, style: const TextStyle(fontSize: 11.5)),
    );
  }
}

class _AltChip extends StatelessWidget {
  const _AltChip({required this.alt, required this.currency});
  final LegAlt alt;
  final String currency;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return InkWell(
      onTap: () => openUrl(context, alt.url),
      borderRadius: BorderRadius.circular(999),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
        decoration: BoxDecoration(
          color: alt.chosen
              ? scheme.primary.withValues(alpha: 0.12)
              : scheme.surface,
          borderRadius: BorderRadius.circular(999),
          border: Border.all(
              color: alt.chosen ? scheme.primary : scheme.outline),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (alt.chosen) ...[
              Icon(Icons.check, size: 13, color: scheme.primary),
              const SizedBox(width: 4),
            ],
            Text(alt.source,
                style: const TextStyle(fontSize: 11.5, fontWeight: FontWeight.w600)),
            if (alt.price > 0) ...[
              const SizedBox(width: 6),
              Text(money(alt.price, currency),
                  style: TextStyle(fontSize: 11.5, color: scheme.primary)),
            ],
            const SizedBox(width: 4),
            Icon(Icons.open_in_new,
                size: 12, color: scheme.onSurface.withValues(alpha: 0.5)),
          ],
        ),
      ),
    );
  }
}
