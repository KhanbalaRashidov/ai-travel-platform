import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../api/api_client.dart';
import '../api/travel_api.dart';

/// The shared HTTP client (base URL from --dart-define=API_BASE).
final apiClientProvider = Provider<ApiClient>((ref) => ApiClient());

/// The typed gateway client used across the app. Plan/roadmap/chat state
/// controllers (added in later stages) depend on this.
final travelApiProvider =
    Provider<TravelApi>((ref) => TravelApi(ref.watch(apiClientProvider)));
