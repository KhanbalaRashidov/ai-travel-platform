import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../i18n/language_switcher.dart';
import '../../state/settings_controller.dart';
import '../../theme/brand.dart';
import '../../widgets/common.dart';
import '../../widgets/money.dart';
import 'plan_state.dart';
import 'result_section.dart';
import 'widgets/destinations_list.dart';
import 'widgets/pickers.dart';
import 'widgets/suggest_panel.dart';

/// The planner: destination input (full / minimal / country), trip parameters and
/// the computed result. Business logic is entirely server-side; this assembles
/// the request and renders what the gateway returns.
class PlanScreen extends ConsumerWidget {
  const PlanScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final state = ref.watch(planControllerProvider);
    final ctrl = ref.read(planControllerProvider.notifier);

    return Scaffold(
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => context.push('/chat'),
        icon: const Icon(Icons.auto_awesome, size: 18),
        label: Text(t('ai.title')),
      ),
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.go('/'),
        ),
        titleSpacing: 0,
        title: Row(
          mainAxisSize: MainAxisSize.min,
          children: const [
            LogoMark(size: 26),
            SizedBox(width: 8),
            BrandWordmark(fontSize: 16),
          ],
        ),
        actions: const [
          ThemeToggleButton(),
          Padding(
            padding: EdgeInsets.only(right: 10, left: 2),
            child: Center(child: LanguageSwitcher()),
          ),
        ],
      ),
      body: SafeArea(
        child: LayoutBuilder(builder: (context, c) {
          return SingleChildScrollView(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 40),
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 760),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    _ModeTabs(state: state, ctrl: ctrl),
                    const SizedBox(height: 12),
                    _OriginBudget(state: state, ctrl: ctrl),
                    const SizedBox(height: 12),
                    _DestinationsSection(state: state, ctrl: ctrl),
                    const SizedBox(height: 12),
                    _TripParams(state: state, ctrl: ctrl),
                    const SizedBox(height: 16),
                    FilledButton(
                      onPressed: state.loading ? null : () => ctrl.compute(),
                      child: Padding(
                        padding: const EdgeInsets.symmetric(vertical: 4),
                        child: state.loading
                            ? Row(
                                mainAxisAlignment: MainAxisAlignment.center,
                                children: [
                                  const SizedBox(
                                      width: 18,
                                      height: 18,
                                      child: CircularProgressIndicator(
                                          strokeWidth: 2, color: Colors.white)),
                                  const SizedBox(width: 10),
                                  Text(t('plan.calculating')),
                                ],
                              )
                            : Text(t('plan.calc'),
                                style: const TextStyle(fontSize: 15)),
                      ),
                    ),
                    if (state.error != null)
                      Padding(
                        padding: const EdgeInsets.only(top: 12),
                        child: _ErrorBox(messageKey: state.error!),
                      ),
                    const SizedBox(height: 16),
                    const ResultSection(),
                  ],
                ),
              ),
            ),
          );
        }),
      ),
    );
  }
}

class _ModeTabs extends StatelessWidget {
  const _ModeTabs({required this.state, required this.ctrl});
  final PlanState state;
  final PlanController ctrl;

  @override
  Widget build(BuildContext context) {
    final labels = {
      InputMode.full: 'Cities',
      InputMode.minimal: 'Budget',
      InputMode.country: 'Country',
    };
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: scheme.surfaceContainerHighest.withValues(alpha: 0.5),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: scheme.outline),
      ),
      child: Row(
        children: [
          for (final m in InputMode.values)
            Expanded(
              child: GestureDetector(
                onTap: () => ctrl.setInputMode(m),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 140),
                  padding: const EdgeInsets.symmetric(vertical: 10),
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: m == state.inputMode ? scheme.primary : null,
                    borderRadius: BorderRadius.circular(9),
                  ),
                  child: Text(
                    labels[m]!,
                    style: TextStyle(
                      fontWeight: FontWeight.w700,
                      fontSize: 13,
                      color: m == state.inputMode
                          ? scheme.onPrimary
                          : scheme.onSurface.withValues(alpha: 0.7),
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _OriginBudget extends ConsumerWidget {
  const _OriginBudget({required this.state, required this.ctrl});
  final PlanState state;
  final PlanController ctrl;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;
    return SectionCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(t('plan.origin'),
              style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
          const SizedBox(height: 6),
          if (state.origin.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 6),
              child: Wrap(children: [
                Chip(
                  avatar: const Icon(Icons.flight_takeoff, size: 16),
                  label: Text('${state.origin}  ·  ${t('plan.originNote')}'),
                  onDeleted: () => ctrl.setOrigin(''),
                ),
              ]),
            ),
          CityPicker(
            hint: t('plan.origin'),
            onPick: (p) => ctrl.setOrigin(p.name),
          ),
          const SizedBox(height: 12),
          Text(t('plan.budget'),
              style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
          const SizedBox(height: 6),
          TextField(
            keyboardType: TextInputType.number,
            decoration: InputDecoration(
              hintText: t('plan.budget'),
              prefixIcon: const Icon(Icons.euro, size: 18),
            ),
            onChanged: (v) =>
                ctrl.setBudget(double.tryParse(v.replaceAll(',', '.'))),
          ),
          if (state.inputMode != InputMode.full)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Text(t('country.hint'),
                  style: TextStyle(
                      fontSize: 11.5,
                      color: scheme.onSurface.withValues(alpha: 0.55))),
            ),
        ],
      ),
    );
  }
}

class _DestinationsSection extends ConsumerWidget {
  const _DestinationsSection({required this.state, required this.ctrl});
  final PlanState state;
  final PlanController ctrl;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    return SectionCard(
      title: t('plan.cities').toUpperCase(),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          switch (state.inputMode) {
            InputMode.full => CityPicker(
                onPick: (p) => ctrl.addDestination(
                    Destination(name: p.name, lat: p.lat, lng: p.lng)),
              ),
            InputMode.country => const SuggestPanel(minimal: false),
            InputMode.minimal => const SuggestPanel(minimal: true),
          },
          const SizedBox(height: 12),
          const DestinationsList(),
        ],
      ),
    );
  }
}

class _TripParams extends ConsumerWidget {
  const _TripParams({required this.state, required this.ctrl});
  final PlanState state;
  final PlanController ctrl;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;

    Future<void> pickDate(bool isReturn) async {
      final now = DateTime.now();
      final initial = (isReturn ? state.returnDate : state.departDate) ?? now;
      final picked = await showDatePicker(
        context: context,
        initialDate: initial.isBefore(now) ? now : initial,
        firstDate: now,
        lastDate: now.add(const Duration(days: 365 * 2)),
      );
      if (picked == null) return;
      if (isReturn) {
        ctrl.setReturnDate(picked);
      } else {
        ctrl.setDepartDate(picked);
      }
    }

    Widget dateField(String label, DateTime? value, bool isReturn) {
      return Expanded(
        child: InkWell(
          onTap: () => pickDate(isReturn),
          borderRadius: BorderRadius.circular(12),
          child: InputDecorator(
            decoration: InputDecoration(labelText: label),
            child: Text(
              value == null ? '—' : fmtDateDisplay(value),
              style: TextStyle(
                color: value == null
                    ? scheme.onSurface.withValues(alpha: 0.5)
                    : scheme.onSurface,
              ),
            ),
          ),
        ),
      );
    }

    return Column(
      children: [
        SectionCard(
          title: '${t('plan.depart')} / ${t('plan.return')}'.toUpperCase(),
          child: Row(
            children: [
              dateField(t('plan.depart'), state.departDate, false),
              const SizedBox(width: 10),
              dateField(t('plan.return'), state.returnDate, true),
              if (state.returnDate != null)
                IconButton(
                  icon: const Icon(Icons.clear, size: 18),
                  onPressed: () => ctrl.setReturnDate(null),
                ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        SectionCard(
          title: t('plan.passengers').toUpperCase(),
          child: Column(
            children: [
              _paxRow(t('plan.adults'), state.adults, ctrl.setAdults, min: 1),
              const SizedBox(height: 10),
              _paxRow(t('plan.children'), state.children, ctrl.setChildren),
              const SizedBox(height: 10),
              _paxRow(t('plan.infants'), state.infants, ctrl.setInfants),
            ],
          ),
        ),
        const SizedBox(height: 12),
        SectionCard(
          title: t('plan.comfort').toUpperCase(),
          child: SegmentedChips<String>(
            options: const ['budget', 'standard', 'vip'],
            value: state.tier,
            labelOf: (v) => t('tier.$v'),
            onChanged: (v) {
              ctrl.setTier(v);
              if (state.quote != null) ctrl.reprice();
            },
          ),
        ),
        const SizedBox(height: 12),
        SectionCard(
          title: t('plan.transport').toUpperCase(),
          child: SegmentedChips<String>(
            options: const ['cheapest', 'flight', 'train', 'bus', 'ferry', 'car'],
            value: state.transport,
            iconOf: modeIcon,
            labelOf: (v) => t('mode.$v'),
            onChanged: (v) {
              ctrl.setTransport(v);
              if (state.quote != null) ctrl.reprice();
            },
          ),
        ),
        const SizedBox(height: 12),
        SectionCard(
          title: t('plan.currency').toUpperCase(),
          child: SegmentedChips<String>(
            options: const ['EUR', 'USD', 'AZN', 'TRY'],
            value: state.currency,
            labelOf: (v) => v,
            onChanged: (v) {
              ctrl.setCurrency(v);
              if (state.quote != null) ctrl.reprice();
            },
          ),
        ),
      ],
    );
  }

  Widget _paxRow(String label, int value, ValueChanged<int> onChanged,
      {int min = 0}) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text(label, style: const TextStyle(fontWeight: FontWeight.w500)),
        NumberStepper(value: value, min: min, max: 9, onChanged: onChanged),
      ],
    );
  }
}

class _ErrorBox extends ConsumerWidget {
  const _ErrorBox({required this.messageKey});
  final String messageKey;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: scheme.error.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: scheme.error.withValues(alpha: 0.4)),
      ),
      child: Row(
        children: [
          Icon(Icons.error_outline, color: scheme.error, size: 20),
          const SizedBox(width: 10),
          Expanded(child: Text(t(messageKey))),
          TextButton(
            onPressed: () => ref.read(planControllerProvider.notifier).compute(),
            child: Text(t('plan.retry')),
          ),
        ],
      ),
    );
  }
}
