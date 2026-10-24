import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/models.dart';
import '../../../state/settings_controller.dart';
import '../../../theme/app_theme.dart';
import '../../../widgets/common.dart';
import '../../../widgets/links.dart';
import '../../../widgets/money.dart';
import '../plan_state.dart';

/// Live cost: a donut of the categories plus expandable Transport / Hotels /
/// Activities / Food lists (amount + source + "as of" + link), a hotel star-tier
/// selector that re-prices, and the total + per-person. Food comes from the AI
/// roadmap (passed in once it has loaded).
class CostBreakdown extends ConsumerWidget {
  const CostBreakdown({super.key, required this.q, this.food});
  final PlanQuote q;
  final List<FoodLine>? food;

  double get _foodTotal =>
      (food ?? []).fold(0.0, (s, f) => s + (f.total ?? f.perDay * f.nights));

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;
    final cur = q.currency;
    final foodTotal = _foodTotal;
    final grandTotal = q.total + foodTotal;

    final cats = <({String label, double value, Color color})>[
      (label: t('res.transport'), value: q.transport, color: AppColors.violet),
      (label: t('res.hotels'), value: q.hotels, color: AppColors.teal),
      (
        label: t('res.activities'),
        value: q.activities,
        color: const Color(0xFFF59E0B)
      ),
      if (foodTotal > 0)
        (label: t('res.food'), value: foodTotal, color: const Color(0xFFEC4899)),
    ];

    return SectionCard(
      title: t('res.liveCost').toUpperCase(),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (grandTotal > 0) _donut(context, cats, grandTotal, cur),
          const SizedBox(height: 8),
          _Expandable(
            icon: Icons.directions_transit,
            label: t('res.transport'),
            amount: money(q.transport, cur),
            children: [for (final l in q.legs) _legRow(context, l, cur, t)],
          ),
          _Expandable(
            icon: Icons.hotel,
            label: t('res.hotels'),
            amount: money(q.hotels, cur),
            children: [
              for (final h in (q.hotelLines ?? []))
                _HotelRow(hotel: h, currency: cur),
            ],
          ),
          _Expandable(
            icon: Icons.local_activity_outlined,
            label: t('res.activities'),
            amount: money(q.activities, cur),
            children: _activityRows(context, t, cur),
          ),
          if (foodTotal > 0)
            _Expandable(
              icon: Icons.restaurant,
              label: t('res.food'),
              amount: money(foodTotal, cur),
              children: [
                for (final f in food!)
                  _kv(
                    context,
                    '${f.city} · ${t('res.nights', {'n': f.nights})}',
                    '${money(f.total ?? f.perDay * f.nights, cur)}  (~${money(f.perDay, cur)}/${t('res.perNight')})',
                  ),
              ],
            ),
          const Divider(height: 18),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(t('res.total'),
                  style:
                      const TextStyle(fontWeight: FontWeight.w800, fontSize: 15)),
              Text(money(grandTotal, cur),
                  style: TextStyle(
                      fontWeight: FontWeight.w800,
                      fontSize: 17,
                      color: scheme.primary)),
            ],
          ),
          if (q.perPerson != null && (q.travelers ?? 1) > 1)
            Align(
              alignment: Alignment.centerRight,
              child: Text(
                '${money((grandTotal) / (q.travelers ?? 1), cur)} · ${t('res.perPerson')}',
                style: TextStyle(
                    fontSize: 12.5,
                    color: scheme.onSurface.withValues(alpha: 0.65)),
              ),
            ),
          Padding(
            padding: const EdgeInsets.only(top: 10),
            child: Text(t('res.disclaimer'),
                style: TextStyle(
                    fontSize: 11,
                    height: 1.3,
                    color: scheme.onSurface.withValues(alpha: 0.55))),
          ),
        ],
      ),
    );
  }

  Widget _donut(
      BuildContext context,
      List<({String label, double value, Color color})> cats,
      double total,
      String cur) {
    return Row(
      children: [
        SizedBox(
          width: 130,
          height: 130,
          child: Stack(
            alignment: Alignment.center,
            children: [
              PieChart(PieChartData(
                sectionsSpace: 2,
                centerSpaceRadius: 42,
                sections: [
                  for (final c in cats)
                    if (c.value > 0)
                      PieChartSectionData(
                        value: c.value,
                        color: c.color,
                        radius: 16,
                        showTitle: false,
                      ),
                ],
              )),
              Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(money(total, cur),
                      style: const TextStyle(
                          fontWeight: FontWeight.w800, fontSize: 13)),
                ],
              ),
            ],
          ),
        ),
        const SizedBox(width: 16),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              for (final c in cats)
                if (c.value > 0)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 3),
                    child: Row(
                      children: [
                        Container(
                            width: 10,
                            height: 10,
                            decoration: BoxDecoration(
                                color: c.color, shape: BoxShape.circle)),
                        const SizedBox(width: 8),
                        Expanded(
                            child: Text(c.label,
                                style: const TextStyle(fontSize: 12.5))),
                        Text(money(c.value, cur),
                            style: const TextStyle(
                                fontSize: 12.5, fontWeight: FontWeight.w600)),
                      ],
                    ),
                  ),
            ],
          ),
        ),
      ],
    );
  }

  List<Widget> _activityRows(BuildContext context,
      String Function(String, [Map<String, Object>?]) t, String cur) {
    final order = q.order;
    final nights = q.nightsByCity;
    if (nights == null || nights.length != order.length) {
      return [
        _kv(context, t('res.activitiesEst'), money(q.activities, cur)),
      ];
    }
    return [
      for (var i = 0; i < order.length; i++)
        if (nights[i] > 0)
          _kv(
            context,
            '${order[i]} · ${t('res.nights', {'n': nights[i]})}',
            money(q.activitiesPerNight * nights[i], cur),
          ),
    ];
  }

  Widget _legRow(BuildContext context, PlanLeg l, String cur,
      String Function(String, [Map<String, Object>?]) t) {
    final scheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 5),
      child: Row(
        children: [
          Icon(modeIcon(l.mode), size: 15, color: scheme.primary),
          const SizedBox(width: 8),
          Expanded(child: Text('${l.from} → ${l.to}', style: const TextStyle(fontSize: 12.5))),
          if ((l.url ?? '').isNotEmpty)
            InkWell(
              onTap: () => openUrl(context, l.url),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 6),
                child: Icon(Icons.open_in_new,
                    size: 13, color: scheme.onSurface.withValues(alpha: 0.5)),
              ),
            ),
          Text(l.price > 0 ? money(l.price, cur) : t('res.transfer'),
              style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }

  Widget _kv(BuildContext context, String k, String v) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 5),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Expanded(child: Text(k, style: const TextStyle(fontSize: 12.5))),
          Text(v,
              style:
                  const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }
}

/// A collapsible category row with a headline amount.
class _Expandable extends StatelessWidget {
  const _Expandable({
    required this.icon,
    required this.label,
    required this.amount,
    required this.children,
  });
  final IconData icon;
  final String label;
  final String amount;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Theme(
      data: Theme.of(context).copyWith(dividerColor: Colors.transparent),
      child: ExpansionTile(
        tilePadding: EdgeInsets.zero,
        childrenPadding: const EdgeInsets.only(left: 4, bottom: 8),
        leading: Icon(icon, size: 20, color: scheme.primary),
        title: Text(label, style: const TextStyle(fontWeight: FontWeight.w600)),
        trailing: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(amount, style: const TextStyle(fontWeight: FontWeight.w700)),
            const SizedBox(width: 4),
            Icon(Icons.expand_more,
                size: 18, color: scheme.onSurface.withValues(alpha: 0.5)),
          ],
        ),
        children: children.isEmpty
            ? [const SizedBox.shrink()]
            : [
                Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: children)
              ],
      ),
    );
  }
}

class _HotelRow extends ConsumerWidget {
  const _HotelRow({required this.hotel});
  final HotelLine hotel;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final ctrl = ref.read(planControllerProvider.notifier);
    final scheme = Theme.of(context).colorScheme;
    final cur = ref.watch(planControllerProvider.select((s) => s.quote?.currency)) ?? 'EUR';
    final name = (hotel.name ?? '').isNotEmpty
        ? hotel.name!
        : '${hotel.stars}★ ${t('res.hotelWord')}';
    final booking = (hotel.url ?? '').isNotEmpty
        ? hotel.url!
        : hotelSites(hotel.city, hotel.stars).first.url;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('${hotel.city} · $name',
                        style: const TextStyle(
                            fontSize: 12.5, fontWeight: FontWeight.w600)),
                    Text(
                      '${'★' * hotel.stars}  ·  ${t('res.nights', {'n': hotel.nights})} × ${money(hotel.nightly, cur)}',
                      style: TextStyle(
                          fontSize: 11,
                          color: scheme.onSurface.withValues(alpha: 0.6)),
                    ),
                  ],
                ),
              ),
              Text(money(hotel.total, cur),
                  style: const TextStyle(
                      fontSize: 12.5, fontWeight: FontWeight.w700)),
            ],
          ),
          const SizedBox(height: 6),
          Wrap(
            spacing: 6,
            runSpacing: 6,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              _link(context, 'Booking', Icons.open_in_new, booking),
              _link(context, t('res.maps'), Icons.map_outlined,
                  mapsUrl(name: '$name ${hotel.city}')),
              // Star-tier alternatives -> live re-price.
              if ((hotel.alts ?? []).isNotEmpty)
                for (final a in hotel.alts!)
                  ChoiceChip(
                    label: Text('${a.stars}★ ${money(a.nightly, cur)}',
                        style: const TextStyle(fontSize: 10.5)),
                    selected: a.chosen,
                    visualDensity: VisualDensity.compact,
                    onSelected: (_) {
                      if (!a.chosen) ctrl.setCityStars(hotel.city, a.stars);
                    },
                  ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _link(BuildContext context, String label, IconData icon, String url) {
    final scheme = Theme.of(context).colorScheme;
    return InkWell(
      onTap: () => openUrl(context, url),
      borderRadius: BorderRadius.circular(999),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5),
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(999),
          border: Border.all(color: scheme.outline),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 12, color: scheme.primary),
            const SizedBox(width: 4),
            Text(label, style: const TextStyle(fontSize: 11)),
          ],
        ),
      ),
    );
  }
}
