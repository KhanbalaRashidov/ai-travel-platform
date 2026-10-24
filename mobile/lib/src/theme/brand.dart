import 'package:flutter/material.dart';
import 'app_theme.dart';

/// The brand logo mark — the same device the web app uses: a rounded square with
/// the violet→teal gradient and a white navigation arrow. Drawn with a painter so
/// it needs no raster asset and stays crisp at any size.
class LogoMark extends StatelessWidget {
  const LogoMark({super.key, this.size = 36});

  final double size;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: size,
      height: size,
      child: CustomPaint(painter: _LogoPainter()),
    );
  }
}

class _LogoPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final rect = Offset.zero & size;
    final rrect = RRect.fromRectAndRadius(
      rect,
      Radius.circular(size.width * 0.28),
    );
    final bg = Paint()
      ..shader = AppColors.brandGradient.createShader(rect);
    canvas.drawRRect(rrect, bg);

    // A paper-plane / navigation arrow pointing up-right.
    final w = size.width;
    final arrow = Path()
      ..moveTo(w * 0.28, w * 0.66)
      ..lineTo(w * 0.72, w * 0.30)
      ..lineTo(w * 0.56, w * 0.72)
      ..lineTo(w * 0.47, w * 0.55)
      ..close();
    final arrowPaint = Paint()..color = Colors.white;
    canvas.drawPath(arrow, arrowPaint);

    final tail = Path()
      ..moveTo(w * 0.47, w * 0.55)
      ..lineTo(w * 0.40, w * 0.62);
    canvas.drawPath(
      tail,
      Paint()
        ..color = Colors.white.withValues(alpha: 0.85)
        ..strokeWidth = w * 0.03
        ..style = PaintingStyle.stroke
        ..strokeCap = StrokeCap.round,
    );
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}

/// Brand wordmark: the gradient-filled app name, used in the app bar / hero.
class BrandWordmark extends StatelessWidget {
  const BrandWordmark({super.key, this.fontSize = 18});

  final double fontSize;

  @override
  Widget build(BuildContext context) {
    return ShaderMask(
      shaderCallback: (rect) => AppColors.brandGradient.createShader(rect),
      blendMode: BlendMode.srcIn,
      child: Text(
        'AI Travel',
        style: TextStyle(
          fontSize: fontSize,
          fontWeight: FontWeight.w800,
          letterSpacing: -0.4,
          color: Colors.white,
        ),
      ),
    );
  }
}
