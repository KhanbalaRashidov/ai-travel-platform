import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:latlong2/latlong.dart';

import '../../../state/settings_controller.dart';
import '../../../theme/app_theme.dart';
import '../../../widgets/common.dart';
import '../plan_state.dart';
import '../roadmap_state.dart';

/// Map of the route on OpenStreetMap tiles (no API key). City stops are numbered
/// pins connected by a polyline in visiting order; roadmap POIs with coords are
/// small accent dots.
class RouteMap extends ConsumerWidget {
  const RouteMap({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final plan = ref.watch(planControllerProvider);
    final rs = ref.watch(roadmapControllerProvider);

    // Ordered city coordinates: prefer the roadmap's, else the form's.
    final order = plan.quote?.order ?? plan.destinations.map((d) => d.name).toList();
    final coordByName = <String, LatLng>{};
    for (final d in plan.destinations) {
      if (d.lat != 0 || d.lng != 0) coordByName[d.name.toLowerCase()] = LatLng(d.lat, d.lng);
    }
    for (final c in (rs.roadmap?.cities ?? const [])) {
      final lat = c.lat, lng = c.lng;
      if (lat != null && lng != null && (lat != 0 || lng != 0)) {
        coordByName[c.city.toLowerCase()] = LatLng(lat, lng);
      }
    }
    final cityPoints = <({String name, LatLng p})>[];
    for (final name in order) {
      final p = coordByName[name.toLowerCase()];
      if (p != null) cityPoints.add((name: name, p: p));
    }

    // POI dots from the roadmap.
    final poiPoints = <LatLng>[];
    for (final day in (rs.roadmap?.days ?? const [])) {
      for (final poi in day.pois) {
        if (poi.lat != null && poi.lng != null && (poi.lat != 0 || poi.lng != 0)) {
          poiPoints.add(LatLng(poi.lat!, poi.lng!));
        }
      }
    }

    if (cityPoints.length < 2) return const SizedBox.shrink();

    final linePts = cityPoints.map((e) => e.p).toList();

    return SectionCard(
      title: t('res.map').toUpperCase(),
      padding: const EdgeInsets.all(12),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(12),
        child: SizedBox(
          height: 280,
          child: FlutterMap(
            options: MapOptions(
              initialCameraFit: CameraFit.coordinates(
                coordinates: linePts,
                padding: const EdgeInsets.all(40),
              ),
              interactionOptions: const InteractionOptions(
                flags: InteractiveFlag.all & ~InteractiveFlag.rotate,
              ),
            ),
            children: [
              TileLayer(
                urlTemplate: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                userAgentPackageName: 'al.kofe.ai_travel',
                maxZoom: 19,
              ),
              PolylineLayer(polylines: [
                Polyline(
                  points: linePts,
                  strokeWidth: 3,
                  color: AppColors.violet,
                ),
              ]),
              MarkerLayer(markers: [
                for (final p in poiPoints)
                  Marker(
                    point: p,
                    width: 10,
                    height: 10,
                    child: const DecoratedBox(
                      decoration: BoxDecoration(
                        color: AppColors.teal,
                        shape: BoxShape.circle,
                      ),
                    ),
                  ),
              ]),
              MarkerLayer(markers: [
                for (var i = 0; i < cityPoints.length; i++)
                  Marker(
                    point: cityPoints[i].p,
                    width: 26,
                    height: 26,
                    child: Container(
                      decoration: BoxDecoration(
                        gradient: AppColors.brandGradient,
                        shape: BoxShape.circle,
                        border: Border.all(color: Colors.white, width: 2),
                      ),
                      alignment: Alignment.center,
                      child: Text(
                        '${i + 1}',
                        style: const TextStyle(
                            color: Colors.white,
                            fontSize: 12,
                            fontWeight: FontWeight.w800),
                      ),
                    ),
                  ),
              ]),
              const RichAttributionWidget(
                attributions: [
                  TextSourceAttribution('© OpenStreetMap contributors'),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
