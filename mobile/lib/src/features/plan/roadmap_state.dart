import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/models.dart';
import '../../api/requests.dart';
import '../../state/providers.dart';
import '../../state/settings_controller.dart';
import 'plan_state.dart';

@immutable
class RoadmapState {
  const RoadmapState({
    this.loading = false,
    this.error,
    this.roadmap,
    this.citiesKey = '',
    this.nightsSig = '',
    this.staleNights = false,
  });

  final bool loading;
  final String? error;
  final Roadmap? roadmap;
  final String citiesKey; // the city set the current roadmap was built for
  final String nightsSig; // nights signature at generation time
  final bool staleNights; // nights changed since generation (D9 regen hint)

  RoadmapState copyWith({
    bool? loading,
    Object? error = _s,
    Object? roadmap = _s,
    String? citiesKey,
    String? nightsSig,
    bool? staleNights,
  }) =>
      RoadmapState(
        loading: loading ?? this.loading,
        error: error == _s ? this.error : error as String?,
        roadmap: roadmap == _s ? this.roadmap : roadmap as Roadmap?,
        citiesKey: citiesKey ?? this.citiesKey,
        nightsSig: nightsSig ?? this.nightsSig,
        staleNights: staleNights ?? this.staleNights,
      );

  static const _s = Object();
}

/// Generates and edits the AI day-by-day roadmap. Auto-fires once per city set
/// after a plan is computed (E5); nights changes flag a "regenerate" hint (D9);
/// POI remove/add are applied client-side (D18). "Every day full" and proximity
/// grouping are the server's job (fillRoadmapDays / RebalanceDays).
class RoadmapController extends StateNotifier<RoadmapState> {
  RoadmapController(this._ref) : super(const RoadmapState()) {
    // Auto-generate on the first plan for a new city set (E5).
    _ref.listen<PlanQuote?>(
      planControllerProvider.select((s) => s.quote),
      (_, next) => _onQuote(next),
      fireImmediately: true,
    );
  }

  final Ref _ref;

  List<String> _cities() {
    final plan = _ref.read(planControllerProvider);
    final q = plan.quote;
    if (q != null && q.order.isNotEmpty) return q.order;
    return plan.destinations.map((d) => d.name).toList();
  }

  List<int> _nights(List<String> cities) {
    final plan = _ref.read(planControllerProvider);
    final q = plan.quote;
    if (q?.nightsByCity != null && q!.nightsByCity!.length == cities.length) {
      return q.nightsByCity!;
    }
    // Fall back to the per-city nights chosen in the form.
    return cities.map((c) {
      final d = plan.destinations.where((x) => x.name == c);
      return d.isEmpty ? 2 : d.first.nights;
    }).toList();
  }

  List<PlaceRef> _places(List<String> cities) {
    final plan = _ref.read(planControllerProvider);
    final out = <PlaceRef>[];
    for (final c in cities) {
      final d = plan.destinations.where((x) => x.name == c);
      if (d.isNotEmpty && (d.first.lat != 0 || d.first.lng != 0)) {
        out.add(PlaceRef(name: c, lat: d.first.lat, lng: d.first.lng));
      }
    }
    return out;
  }

  void _onQuote(PlanQuote? q) {
    if (q == null) return;
    final cities = _cities();
    if (cities.length < 2) return;
    final ck = cities.join('|').toLowerCase();
    final ns = _nights(cities).join(',');
    if (ck != state.citiesKey) {
      generate(); // new city set -> auto-generate
    } else if (ns != state.nightsSig && state.roadmap != null) {
      state = state.copyWith(staleNights: true); // D9 hint
    }
  }

  Future<void> generate() async {
    final cities = _cities();
    if (cities.length < 2) return;
    final nights = _nights(cities);
    final places = _places(cities);
    final locale = _ref.read(localeProvider);
    final currency = _ref.read(planControllerProvider).currency;
    state = state.copyWith(loading: true, error: null);
    try {
      final api = _ref.read(travelApiProvider);
      final r =
          await api.roadmap(cities, nights, places, locale: locale, currency: currency);
      state = RoadmapState(
        loading: false,
        roadmap: r,
        citiesKey: cities.join('|').toLowerCase(),
        nightsSig: nights.join(','),
        staleNights: false,
      );
    } catch (e) {
      state = state.copyWith(loading: false, error: 'ai.error');
      if (kDebugMode) debugPrint('roadmap error: $e');
    }
  }

  // --- client-side edits (D18) ----------------------------------------------

  void removePoi(int dayIndex, int poiIndex) {
    final r = state.roadmap;
    if (r == null || dayIndex >= r.days.length) return;
    final day = r.days[dayIndex];
    if (poiIndex >= day.pois.length) return;
    final pois = [...day.pois]..removeAt(poiIndex);
    _replaceDay(dayIndex, day.copyWith(pois: pois));
  }

  void addPoi(int dayIndex, RoadmapPoi poi) {
    final r = state.roadmap;
    if (r == null || dayIndex >= r.days.length) return;
    final day = r.days[dayIndex];
    if (day.pois.any((p) => p.name == poi.name)) return;
    _replaceDay(dayIndex, day.copyWith(pois: [...day.pois, poi]));
  }

  void _replaceDay(int index, RoadmapDay day) {
    final r = state.roadmap!;
    final days = [...r.days];
    days[index] = day;
    state = state.copyWith(roadmap: r.copyWith(days: days));
  }
}

final roadmapControllerProvider =
    StateNotifierProvider<RoadmapController, RoadmapState>(
        (ref) => RoadmapController(ref));
