import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/models.dart';
import '../../state/settings_controller.dart';
import '../../widgets/common.dart';
import 'plan_state.dart';
import 'widgets/cost_breakdown.dart';
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
        CostBreakdown(q: q),
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

