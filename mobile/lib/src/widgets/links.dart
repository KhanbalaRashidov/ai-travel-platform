import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import 'common.dart';

/// Open an external URL (booking source, map, provider site) in the browser.
/// Advisory-only: we never book — we hand off to the provider the API linked to.
Future<void> openUrl(BuildContext context, String? url) async {
  if (url == null || url.trim().isEmpty) return;
  final uri = Uri.tryParse(url.trim());
  if (uri == null) return;
  final ok = await launchUrl(uri, mode: LaunchMode.externalApplication);
  if (!ok && context.mounted) {
    showToast(context, url);
  }
}

/// Google Maps link for a place (by coords when known, else by name search).
String mapsUrl({String? name, double? lat, double? lng}) {
  if (lat != null && lng != null && (lat != 0 || lng != 0)) {
    return 'https://www.google.com/maps/search/?api=1&query=$lat,$lng';
  }
  final q = Uri.encodeComponent(name ?? '');
  return 'https://www.google.com/maps/search/?api=1&query=$q';
}

/// Prefilled hotel-search links for a city + star tier (Dart port of the web
/// hotelSites helper). These are search deep-links, not bookings.
List<({String name, String url})> hotelSites(String city, int stars) {
  final c = Uri.encodeComponent(city);
  return [
    (
      name: 'Booking',
      url:
          'https://www.booking.com/searchresults.html?ss=$c&nflt=class%3D$stars'
    ),
    (name: 'Hotels', url: 'https://www.hotels.com/search.do?q-destination=$c'),
    (
      name: 'Google',
      url: 'https://www.google.com/travel/hotels/${Uri.encodeComponent(city)}'
    ),
  ];
}
