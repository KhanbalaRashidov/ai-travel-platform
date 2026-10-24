import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/models.dart';
import '../../state/settings_controller.dart';
import '../../widgets/common.dart';
import '../../widgets/money.dart';
import 'plan_state.dart';
import 'widgets/route_timeline.dart';

/// The computed plan. F3 shows the optimal order + headline totals; the route
/// timeline, detailed live cost, AI roadmap, map, chat and export are layered in
/// by later stages.
class ResultSection extends ConsumerWidget {
  const ResultSection({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final q = ref.watch(planControllerProvider.select((s) => s.quote));

    if (q == null) {
      return SectionCard(
        child: Column(
          children: [
            const SizedBox(height: 8),
            Icon(Icons.map_outlined,
                size: 40,
                color: Theme.of(context)
                    .colorScheme
                    .onSurface
                    .withValues(alpha: 0.3)),
            const SizedBox(height: 12),
            Text(t('res.emptyTitle'),
                style: const TextStyle(fontWeight: FontWeight.w700)),
            const SizedBox(height: 4),
            Text(t('res.emptyDesc'),
                textAlign: TextAlign.center,
                style: TextStyle(
                    fontSize: 13,
                    color: Theme.of(context)
                        .colorScheme
                        .onSurface
                        .withValues(alpha: 0.6))),
            const SizedBox(height: 8),
          ],
        ),
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _OrderCard(q: q),
        const SizedBox(height: 12),
        RouteTimeline(q: q),
        const SizedBox(height: 12),
        _TotalsCard(q: q),
      ],
    );
  }
}

class _OrderCard extends ConsumerWidget {
  const _OrderCard({required this.q});
  final PlanQuote q;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;
    final variantLabel = q.variant == 'open_jaw'
        ? t('res.openjaw')
        : q.variant == 'round_trip'
            ? t('res.roundtrip')
            : q.variant;
    return SectionCard(
      title: t('res.optimalOrder').toUpperCase(),
      trailing: variantLabel.isEmpty
          ? null
          : Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
              decoration: BoxDecoration(
                color: scheme.secondary.withValues(alpha: 0.15),
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(variantLabel,
                  style: TextStyle(
                      fontSize: 11,
                      fontWeight: FontWeight.w700,
                      color: scheme.secondary)),
            ),
      child: Wrap(
        spacing: 6,
        runSpacing: 6,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          for (var i = 0; i < q.order.length; i++) ...[
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
              decoration: BoxDecoration(
                color: scheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(999),
                border: Border.all(color: scheme.outline),
              ),
              child: Text(q.order[i],
                  style: const TextStyle(
                      fontWeight: FontWeight.w600, fontSize: 13)),
            ),
            if (i < q.order.length - 1)
              Icon(Icons.arrow_forward,
                  size: 14, color: scheme.onSurface.withValues(alpha: 0.4)),
          ],
        ],
      ),
    );
  }
}

class _TotalsCard extends ConsumerWidget {
  const _TotalsCard({required this.q});
  final PlanQuote q;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;
    final cur = q.currency;

    Widget row(String label, num value, {bool bold = false}) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 5),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(label,
                style: TextStyle(
                    fontWeight: bold ? FontWeight.w800 : FontWeight.w500,
                    fontSize: bold ? 15 : 13.5)),
            Text(money(value, cur),
                style: TextStyle(
                    fontWeight: bold ? FontWeight.w800 : FontWeight.w600,
                    fontSize: bold ? 16 : 14,
                    color: bold ? scheme.primary : scheme.onSurface)),
          ],
        ),
      );
    }

    return SectionCard(
      title: t('res.liveCost').toUpperCase(),
      child: Column(
        children: [
          row(t('res.transport'), q.transport),
          const Divider(height: 1),
          row(t('res.hotels'), q.hotels),
          const Divider(height: 1),
          row(t('res.activities'), q.activities),
          const Divider(height: 8),
          row(t('res.total'), q.total, bold: true),
          if (q.perPerson != null && q.travelers != null && q.travelers! > 1)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Align(
                alignment: Alignment.centerRight,
                child: Text(
                  '${money(q.perPerson!, cur)} · ${t('res.perPerson')}',
                  style: TextStyle(
                      fontSize: 12.5,
                      color: scheme.onSurface.withValues(alpha: 0.65)),
                ),
              ),
            ),
          if (q.estimated)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Row(
                children: [
                  Icon(Icons.info_outline,
                      size: 14,
                      color: scheme.onSurface.withValues(alpha: 0.5)),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(t('res.disclaimer'),
                        style: TextStyle(
                            fontSize: 11,
                            height: 1.3,
                            color: scheme.onSurface.withValues(alpha: 0.55))),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}
