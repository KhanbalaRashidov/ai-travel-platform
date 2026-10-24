import 'package:go_router/go_router.dart';

import '../features/home/home_screen.dart';
import '../features/plan/plan_screen.dart';

/// App navigation. Two surfaces, mirroring the web app: the landing/home page and
/// the planner. Deep routes (result, map, roadmap) live inside the planner as
/// scrollable sections / sheets rather than separate URLs.
final appRouter = GoRouter(
  initialLocation: '/',
  routes: [
    GoRoute(
      path: '/',
      name: 'home',
      builder: (context, state) => const HomeScreen(),
    ),
    GoRoute(
      path: '/plan',
      name: 'plan',
      builder: (context, state) => const PlanScreen(),
    ),
  ],
);
