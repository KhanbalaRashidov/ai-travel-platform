import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../i18n/language_switcher.dart';
import '../../state/settings_controller.dart';
import '../../theme/app_theme.dart';
import '../../theme/brand.dart';

/// Landing page — brand hero, advisory-only badge, CTA into the planner, and the
/// three feature cards. Parity with the web home page; responsive down to phone.
class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = ref.watch(tProvider);
    final scheme = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: AppBar(
        titleSpacing: 16,
        title: Row(
          mainAxisSize: MainAxisSize.min,
          children: const [
            LogoMark(size: 30),
            SizedBox(width: 10),
            BrandWordmark(fontSize: 18),
          ],
        ),
        actions: const [
          ThemeToggleButton(),
          Padding(
            padding: EdgeInsets.only(right: 12, left: 4),
            child: Center(child: LanguageSwitcher()),
          ),
        ],
      ),
      body: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final wide = constraints.maxWidth >= 720;
            return SingleChildScrollView(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 900),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const SizedBox(height: 8),
                      _Badge(text: t('app.badge')),
                      const SizedBox(height: 18),
                      ShaderMask(
                        shaderCallback: (r) =>
                            AppColors.brandGradient.createShader(r),
                        blendMode: BlendMode.srcIn,
                        child: Text(
                          t('app.tagline'),
                          style: const TextStyle(
                            fontSize: 30,
                            height: 1.15,
                            fontWeight: FontWeight.w800,
                            letterSpacing: -0.6,
                            color: Colors.white,
                          ),
                        ),
                      ),
                      const SizedBox(height: 12),
                      Text(
                        t('app.heroLead'),
                        style: TextStyle(
                          fontSize: 16,
                          color: scheme.onSurface.withValues(alpha: 0.7),
                        ),
                      ),
                      const SizedBox(height: 22),
                      FilledButton(
                        onPressed: () => context.go('/plan'),
                        child: Text(t('app.cta')),
                      ),
                      const SizedBox(height: 32),
                      Flex(
                        direction: wide ? Axis.horizontal : Axis.vertical,
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          _featureCard(context, Icons.route_outlined,
                              t('feature.route.t'), t('feature.route.d'), wide),
                          SizedBox(width: wide ? 14 : 0, height: wide ? 0 : 14),
                          _featureCard(context, Icons.payments_outlined,
                              t('feature.cost.t'), t('feature.cost.d'), wide),
                          SizedBox(width: wide ? 14 : 0, height: wide ? 0 : 14),
                          _featureCard(context, Icons.auto_graph_outlined,
                              t('feature.smart.t'), t('feature.smart.d'), wide),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
            );
          },
        ),
      ),
    );
  }

  Widget _featureCard(BuildContext context, IconData icon, String title,
      String desc, bool wide) {
    final scheme = Theme.of(context).colorScheme;
    final card = Card(
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                gradient: AppColors.brandGradient,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Icon(icon, color: Colors.white, size: 22),
            ),
            const SizedBox(height: 14),
            Text(title,
                style: const TextStyle(
                    fontSize: 16, fontWeight: FontWeight.w700)),
            const SizedBox(height: 6),
            Text(desc,
                style: TextStyle(
                    fontSize: 13.5,
                    height: 1.4,
                    color: scheme.onSurface.withValues(alpha: 0.7))),
          ],
        ),
      ),
    );
    return wide ? Expanded(child: card) : card;
  }
}

class _Badge extends StatelessWidget {
  const _Badge({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
      decoration: BoxDecoration(
        color: AppColors.teal.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: AppColors.teal.withValues(alpha: 0.4)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.verified_outlined, size: 15, color: AppColors.teal),
          const SizedBox(width: 6),
          Flexible(
            child: Text(
              text,
              style: TextStyle(
                fontSize: 12.5,
                fontWeight: FontWeight.w600,
                color: scheme.onSurface.withValues(alpha: 0.85),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
