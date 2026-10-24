import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../state/settings_controller.dart';
import '../../../widgets/common.dart';
import '../plan_state.dart';

/// The chosen destinations: drag to reorder (first = first stop), a per-city
/// nights stepper, and remove. Mirrors the web "How many nights per city?" list.
class DestinationsList extends ConsumerWidget {
  const DestinationsList({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final state = ref.watch(planControllerProvider);
    final ctrl = ref.read(planControllerProvider.notifier);
    final scheme = Theme.of(context).colorScheme;
    final dests = state.destinations;

    if (dests.isEmpty) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Text(t('plan.pickCities'),
            style: TextStyle(
                color: scheme.onSurface.withValues(alpha: 0.6), fontSize: 13)),
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(bottom: 4),
          child: Text(
            '${t('plan.nightsTitle')}  ·  ${t('plan.dragHint')}',
            style: TextStyle(
                fontSize: 11.5,
                color: scheme.onSurface.withValues(alpha: 0.55)),
          ),
        ),
        ReorderableListView.builder(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          buildDefaultDragHandles: false,
          itemCount: dests.length,
          onReorder: ctrl.reorder,
          itemBuilder: (context, i) {
            final d = dests[i];
            return Padding(
              key: ValueKey(d.name),
              padding: const EdgeInsets.only(bottom: 8),
              child: Container(
                padding: const EdgeInsets.fromLTRB(6, 6, 10, 6),
                decoration: BoxDecoration(
                  color: scheme.surfaceContainerHighest.withValues(alpha: 0.5),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: scheme.outline),
                ),
                child: Row(
                  children: [
                    ReorderableDragStartListener(
                      index: i,
                      child: Padding(
                        padding: const EdgeInsets.all(6),
                        child: Icon(Icons.drag_indicator,
                            size: 20,
                            color: scheme.onSurface.withValues(alpha: 0.4)),
                      ),
                    ),
                    if (i == 0)
                      Container(
                        margin: const EdgeInsets.only(right: 6),
                        padding: const EdgeInsets.symmetric(
                            horizontal: 7, vertical: 2),
                        decoration: BoxDecoration(
                          color: scheme.primary.withValues(alpha: 0.15),
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Text(t('plan.start'),
                            style: TextStyle(
                                fontSize: 9.5,
                                fontWeight: FontWeight.w700,
                                color: scheme.primary)),
                      ),
                    Expanded(
                      child: Text(d.name,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(fontWeight: FontWeight.w600)),
                    ),
                    NumberStepper(
                      value: d.nights,
                      min: 1,
                      max: 30,
                      suffix: t('res.perNight'),
                      onChanged: (v) => ctrl.setNights(d.name, v),
                    ),
                    IconButton(
                      visualDensity: VisualDensity.compact,
                      icon: Icon(Icons.close,
                          size: 18, color: scheme.onSurface.withValues(alpha: 0.5)),
                      onPressed: () => ctrl.removeDestination(d.name),
                    ),
                  ],
                ),
              ),
            );
          },
        ),
      ],
    );
  }
}
