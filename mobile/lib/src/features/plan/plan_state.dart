import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/api_client.dart';
import '../../api/models.dart';
import '../../api/requests.dart';
import '../../state/providers.dart';

/// The three ways to pick destinations, mirroring the web form.
enum InputMode { full, minimal, country }

/// A chosen destination with its coordinates and nights.
@immutable
class Destination {
  const Destination({
    required this.name,
    this.lat = 0,
    this.lng = 0,
    this.nights = 2,
  });
  final String name;
  final double lat;
  final double lng;
  final int nights;

  Destination copyWith({int? nights}) => Destination(
        name: name,
        lat: lat,
        lng: lng,
        nights: nights ?? this.nights,
      );
}

@immutable
class PlanState {
  const PlanState({
    this.inputMode = InputMode.full,
    this.origin = '',
    this.budgetEur,
    this.destinations = const [],
    this.departDate,
    this.returnDate,
    this.adults = 1,
    this.children = 0,
    this.infants = 0,
    this.tier = 'standard',
    this.transport = 'cheapest',
    this.currency = 'EUR',
    this.loading = false,
    this.error,
    this.quote,
    this.legModes = const {},
    this.cityStars = const {},
  });

  final InputMode inputMode;
  final String origin;
  final double? budgetEur;
  final List<Destination> destinations;
  final DateTime? departDate;
  final DateTime? returnDate;
  final int adults;
  final int children;
  final int infants;
  final String tier; // budget | standard | vip
  final String transport; // cheapest | flight | train | bus | ferry | car
  final String currency;

  // Result + interaction overrides
  final bool loading;
  final String? error;
  final PlanQuote? quote;
  final Map<String, String> legModes; // "from|to" -> mode
  final Map<String, int> cityStars; // city -> stars

  int get travelers => adults + children + infants;

  PlanState copyWith({
    InputMode? inputMode,
    String? origin,
    Object? budgetEur = _sentinel,
    List<Destination>? destinations,
    Object? departDate = _sentinel,
    Object? returnDate = _sentinel,
    int? adults,
    int? children,
    int? infants,
    String? tier,
    String? transport,
    String? currency,
    bool? loading,
    Object? error = _sentinel,
    Object? quote = _sentinel,
    Map<String, String>? legModes,
    Map<String, int>? cityStars,
  }) {
    return PlanState(
      inputMode: inputMode ?? this.inputMode,
      origin: origin ?? this.origin,
      budgetEur: budgetEur == _sentinel ? this.budgetEur : budgetEur as double?,
      destinations: destinations ?? this.destinations,
      departDate:
          departDate == _sentinel ? this.departDate : departDate as DateTime?,
      returnDate:
          returnDate == _sentinel ? this.returnDate : returnDate as DateTime?,
      adults: adults ?? this.adults,
      children: children ?? this.children,
      infants: infants ?? this.infants,
      tier: tier ?? this.tier,
      transport: transport ?? this.transport,
      currency: currency ?? this.currency,
      loading: loading ?? this.loading,
      error: error == _sentinel ? this.error : error as String?,
      quote: quote == _sentinel ? this.quote : quote as PlanQuote?,
      legModes: legModes ?? this.legModes,
      cityStars: cityStars ?? this.cityStars,
    );
  }

  static const _sentinel = Object();
}

String _fmtDate(DateTime? d) {
  if (d == null) return '';
  final m = d.month.toString().padLeft(2, '0');
  final day = d.day.toString().padLeft(2, '0');
  return '${d.year}-$m-$day';
}

/// Holds the form, computes the plan and re-prices on interaction. All pricing
/// is server-side; this only assembles the request and stores the response.
class PlanController extends StateNotifier<PlanState> {
  PlanController(this._ref) : super(const PlanState());

  final Ref _ref;

  // --- form mutations --------------------------------------------------------

  void setInputMode(InputMode m) => state = state.copyWith(inputMode: m);
  void setOrigin(String v) => state = state.copyWith(origin: v);
  void setBudget(double? v) => state = state.copyWith(budgetEur: v);
  void setDepartDate(DateTime? d) => state = state.copyWith(departDate: d);
  void setReturnDate(DateTime? d) => state = state.copyWith(returnDate: d);
  void setAdults(int v) => state = state.copyWith(adults: v.clamp(1, 9));
  void setChildren(int v) => state = state.copyWith(children: v.clamp(0, 9));
  void setInfants(int v) => state = state.copyWith(infants: v.clamp(0, 9));
  void setTier(String v) => state = state.copyWith(tier: v);
  void setTransport(String v) => state = state.copyWith(transport: v);
  void setCurrency(String v) => state = state.copyWith(currency: v);

  void addDestination(Destination d) {
    if (state.destinations.any((x) => x.name == d.name)) return;
    state = state.copyWith(destinations: [...state.destinations, d]);
  }

  void removeDestination(String name) {
    state = state.copyWith(
        destinations:
            state.destinations.where((d) => d.name != name).toList());
  }

  void setNights(String name, int nights) {
    state = state.copyWith(
      destinations: state.destinations
          .map((d) => d.name == name ? d.copyWith(nights: nights.clamp(1, 30)) : d)
          .toList(),
    );
  }

  void reorder(int oldIndex, int newIndex) {
    final list = [...state.destinations];
    if (newIndex > oldIndex) newIndex -= 1;
    final item = list.removeAt(oldIndex);
    list.insert(newIndex, item);
    state = state.copyWith(destinations: list);
  }

  void setCityStars(String city, int stars) {
    final next = {...state.cityStars, city: stars};
    state = state.copyWith(cityStars: next);
    compute(); // re-price with the new tier for that city
  }

  // --- compute ---------------------------------------------------------------

  PlanQuoteReq _buildReq({List<String>? order, String? variant}) {
    final d = state.destinations;
    return PlanQuoteReq(
      cities: d.map((e) => e.name).toList(),
      places: d
          .where((e) => e.lat != 0 || e.lng != 0)
          .map((e) => PlaceRef(name: e.name, lat: e.lat, lng: e.lng))
          .toList(),
      origin: state.origin.trim().isEmpty ? null : state.origin.trim(),
      budgetEur: state.budgetEur,
      start: d.isNotEmpty ? d.first.name : '',
      mode: state.transport,
      tier: state.tier,
      nights: 2,
      currency: state.currency,
      order: order,
      variant: variant,
      legModes: state.legModes.isEmpty ? null : state.legModes,
      startDate: _fmtDate(state.departDate),
      returnDate: _fmtDate(state.returnDate),
      adults: state.adults,
      children: state.children,
      infants: state.infants,
      cityNights: {for (final e in d) e.name: e.nights},
      cityStars: state.cityStars.isEmpty ? null : state.cityStars,
    );
  }

  Future<void> compute({List<String>? order, String? variant}) async {
    if (state.destinations.length < 2) {
      state = state.copyWith(error: 'err.minCities');
      return;
    }
    state = state.copyWith(loading: true, error: null);
    try {
      final api = _ref.read(travelApiProvider);
      final quote = await api.planQuote(_buildReq(order: order, variant: variant));
      state = state.copyWith(loading: false, quote: quote, error: null);
    } on ApiException catch (e) {
      state = state.copyWith(
        loading: false,
        error: e.isTimeout ? 'err.timeout' : 'err.plan',
      );
      if (kDebugMode) debugPrint('plan error: $e');
    } catch (e) {
      state = state.copyWith(loading: false, error: 'err.plan');
      if (kDebugMode) debugPrint('plan error: $e');
    }
  }

  /// Re-price keeping the already-chosen order (used by cost/timeline toggles).
  Future<void> reprice() async {
    final q = state.quote;
    await compute(order: q?.order, variant: q?.variant);
  }

  /// Override one leg's transport mode and re-price.
  Future<void> setLegMode(String from, String to, String mode) async {
    final key = '$from|$to';
    final next = {...state.legModes};
    if (mode.isEmpty || mode == 'cheapest') {
      next.remove(key);
    } else {
      next[key] = mode;
    }
    state = state.copyWith(legModes: next);
    await reprice();
  }
}

final planControllerProvider =
    StateNotifierProvider<PlanController, PlanState>((ref) => PlanController(ref));
