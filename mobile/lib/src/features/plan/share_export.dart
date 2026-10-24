import 'dart:typed_data';

import 'package:pdf/pdf.dart';
import 'package:pdf/widgets.dart' as pw;
import 'package:printing/printing.dart';
import 'package:share_plus/share_plus.dart';

import '../../api/models.dart';
import '../../widgets/money.dart';

/// Plain-text trip summary for the native share sheet (route + totals + roadmap).
String buildShareText(
  PlanQuote q,
  Roadmap? roadmap,
  String Function(String, [Map<String, Object>?]) t,
) {
  final cur = q.currency;
  final b = StringBuffer();
  b.writeln(t('app.title'));
  b.writeln(t('res.route'));
  b.writeln(q.order.join('  →  '));
  b.writeln();
  for (final l in q.legs) {
    final price = l.price > 0 ? money(l.price, cur) : t('res.transfer');
    b.writeln('• ${l.from} → ${l.to} (${t('mode.${l.mode}')}): $price');
  }
  b.writeln();
  b.writeln('${t('res.transport')}: ${money(q.transport, cur)}');
  b.writeln('${t('res.hotels')}: ${money(q.hotels, cur)}');
  b.writeln('${t('res.activities')}: ${money(q.activities, cur)}');
  final food = (roadmap?.food ?? [])
      .fold<double>(0, (s, f) => s + (f.total ?? f.perDay * f.nights));
  if (food > 0) b.writeln('${t('res.food')}: ${money(food, cur)}');
  b.writeln('${t('res.total')}: ${money(q.total + food, cur)}');
  if (roadmap != null && roadmap.days.isNotEmpty) {
    b.writeln();
    b.writeln(t('ai.itin.title'));
    for (final d in roadmap.days) {
      b.writeln('${t('res.day', {'n': d.day})} · ${d.city}');
      for (final p in d.pois) {
        b.writeln('  - ${p.name}');
      }
    }
  }
  b.writeln();
  b.writeln(t('app.badge'));
  return b.toString();
}

/// Share the trip summary via the OS share sheet.
Future<void> sharePlan(
  PlanQuote q,
  Roadmap? roadmap,
  String Function(String, [Map<String, Object>?]) t,
) async {
  await Share.share(buildShareText(q, roadmap, t), subject: t('res.planTitle'));
}

/// Build a structured PDF of the plan INCLUDING the AI day-by-day roadmap, then
/// open the print / save-as-PDF / share dialog (D15 parity).
Future<void> printPlan(
  PlanQuote q,
  Roadmap? roadmap,
  String Function(String, [Map<String, Object>?]) t,
) async {
  final bytes = await _buildPdf(q, roadmap, t);
  await Printing.layoutPdf(onLayout: (_) async => bytes, name: t('print.title'));
}

Future<void> sharePlanPdf(
  PlanQuote q,
  Roadmap? roadmap,
  String Function(String, [Map<String, Object>?]) t,
) async {
  final bytes = await _buildPdf(q, roadmap, t);
  await Printing.sharePdf(bytes: bytes, filename: 'ai-travel-plan.pdf');
}

Future<Uint8List> _buildPdf(
  PlanQuote q,
  Roadmap? roadmap,
  String Function(String, [Map<String, Object>?]) t,
) async {
  final doc = pw.Document();
  final cur = q.currency;
  final food = (roadmap?.food ?? [])
      .fold<double>(0, (s, f) => s + (f.total ?? f.perDay * f.nights));

  pw.Widget kv(String k, String v, {bool bold = false}) => pw.Padding(
        padding: const pw.EdgeInsets.symmetric(vertical: 2),
        child: pw.Row(
          mainAxisAlignment: pw.MainAxisAlignment.spaceBetween,
          children: [
            pw.Text(k,
                style: pw.TextStyle(
                    fontWeight: bold ? pw.FontWeight.bold : pw.FontWeight.normal)),
            pw.Text(v,
                style: pw.TextStyle(
                    fontWeight: bold ? pw.FontWeight.bold : pw.FontWeight.normal)),
          ],
        ),
      );

  doc.addPage(
    pw.MultiPage(
      pageFormat: PdfPageFormat.a4,
      margin: const pw.EdgeInsets.all(28),
      build: (ctx) => [
        pw.Header(
          level: 0,
          child: pw.Text(t('print.title'),
              style: pw.TextStyle(fontSize: 20, fontWeight: pw.FontWeight.bold)),
        ),
        pw.Text(q.order.join('  >  '),
            style: const pw.TextStyle(fontSize: 12, color: PdfColors.grey700)),
        pw.SizedBox(height: 14),

        pw.Text(t('res.route'),
            style: pw.TextStyle(fontSize: 14, fontWeight: pw.FontWeight.bold)),
        pw.SizedBox(height: 6),
        pw.TableHelper.fromTextArray(
          headerStyle: pw.TextStyle(fontWeight: pw.FontWeight.bold),
          cellAlignment: pw.Alignment.centerLeft,
          headers: ['', t('plan.transport'), ''],
          data: [
            for (final l in q.legs)
              [
                '${l.from} > ${l.to}',
                t('mode.${l.mode}'),
                l.price > 0 ? money(l.price, cur) : t('res.transfer'),
              ],
          ],
        ),
        pw.SizedBox(height: 14),

        pw.Text(t('res.liveCost'),
            style: pw.TextStyle(fontSize: 14, fontWeight: pw.FontWeight.bold)),
        pw.SizedBox(height: 6),
        kv(t('res.transport'), money(q.transport, cur)),
        kv(t('res.hotels'), money(q.hotels, cur)),
        kv(t('res.activities'), money(q.activities, cur)),
        if (food > 0) kv(t('res.food'), money(food, cur)),
        pw.Divider(),
        kv(t('res.total'), money(q.total + food, cur), bold: true),
        if (q.perPerson != null && (q.travelers ?? 1) > 1)
          pw.Text(
              '${money((q.total + food) / (q.travelers ?? 1), cur)} · ${t('res.perPerson')}',
              style: const pw.TextStyle(fontSize: 10, color: PdfColors.grey700)),
        pw.SizedBox(height: 14),

        if (roadmap != null && roadmap.days.isNotEmpty) ...[
          pw.Text(t('ai.itin.title'),
              style:
                  pw.TextStyle(fontSize: 14, fontWeight: pw.FontWeight.bold)),
          pw.SizedBox(height: 6),
          for (final city in roadmap.cities) ...[
            pw.Text(
                '${city.city}  ·  ${t('res.nights', {'n': city.nights})}'
                '${(city.bestSeason ?? '').isNotEmpty ? '  ·  ${t('res.season')}: ${city.bestSeason}' : ''}',
                style: pw.TextStyle(fontWeight: pw.FontWeight.bold)),
            if ((city.summary ?? '').isNotEmpty)
              pw.Text(city.summary!,
                  style:
                      const pw.TextStyle(fontSize: 10, color: PdfColors.grey700)),
            for (final d in roadmap.days
                .where((x) => x.city.toLowerCase() == city.city.toLowerCase()))
              pw.Padding(
                padding: const pw.EdgeInsets.only(left: 8, top: 4, bottom: 4),
                child: pw.Column(
                  crossAxisAlignment: pw.CrossAxisAlignment.start,
                  children: [
                    pw.Text(
                        d.title?.isNotEmpty == true
                            ? '${t('res.day', {'n': d.day})} · ${d.title}'
                            : t('res.day', {'n': d.day}),
                        style: pw.TextStyle(
                            fontWeight: pw.FontWeight.bold, fontSize: 11)),
                    for (final p in d.pois)
                      pw.Bullet(
                        text:
                            '${p.name}${p.free ? ' (${t('res.free')})' : (p.price ?? '').isNotEmpty ? ' (${p.price})' : ''}',
                        style: const pw.TextStyle(fontSize: 10),
                      ),
                  ],
                ),
              ),
            pw.SizedBox(height: 8),
          ],
        ],

        pw.SizedBox(height: 10),
        pw.Text(t('res.disclaimer'),
            style: const pw.TextStyle(fontSize: 8, color: PdfColors.grey600)),
        pw.Text(t('app.badge'),
            style: const pw.TextStyle(fontSize: 8, color: PdfColors.grey600)),
      ],
    ),
  );

  return doc.save();
}
