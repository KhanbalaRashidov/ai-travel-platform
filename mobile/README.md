# AI Travel — Mobile (Flutter)

Flutter client for the [AI Travel Platform](../README.md). It is a **thin client**:
all routing, pricing, roadmap and AI logic stay on the Go gateway — the app only
calls `/api/v1/*`. **No secrets ship in the app**; the only build-time input is the
gateway URL via `--dart-define=API_BASE=...`.

- **Stack:** Flutter (Material 3) · Riverpod · Dio · go_router · freezed/json_serializable
- **Default language:** EN (also AZ / RU / TR, same keys as the web app)
- **Platforms:** Android + iOS
- **Design:** same brand as web — violet `#8B7CF6` → teal `#2DD4BF` gradient, deep-navy dark surface, light/dark mode

## One-time bootstrap

The repo tracks the Dart source, assets and configuration. Platform folders
(`android/`, `ios/`) and the gradle wrapper are generated locally so they are not
committed. From `mobile/`:

```bash
# 1) materialize the platform projects (does not overwrite lib/ or pubspec.yaml)
flutter create . --org al.kofe --project-name ai_travel --platforms=android,ios

# 2) fetch packages
flutter pub get

# 3) generate model code (freezed / json_serializable)
dart run build_runner build --delete-conflicting-outputs

# 4) brand launcher icon + splash (the 1024x1024 assets/icon/app_icon.png and its
#    svg source are already in the repo)
dart run flutter_launcher_icons
dart run flutter_native_splash:create
```

## Run

```bash
# Android emulator (uses host 10.0.2.2:8080 by default — no flag needed)
flutter run

# Real phone / tablet on the same Wi-Fi — point it at the host's LAN IP
flutter run --dart-define=API_BASE=http://192.168.1.10:8080
```

The gateway must be reachable at that address (`docker compose -f ../deploy/docker-compose.yml up -d`).
`API_BASE` accepts a bare host (`http://host:8080`) or a full `http://host:8080/api/v1`.

## Build

```bash
# Debug APK
flutter build apk --debug --dart-define=API_BASE=http://192.168.1.10:8080

# Release APK
flutter build apk --release --dart-define=API_BASE=https://api.example.com

# iOS (needs macOS + Xcode)
flutter build ios --dart-define=API_BASE=https://api.example.com
```

## Verify

```bash
flutter analyze
flutter test
```

## Project layout

```
lib/
  main.dart                     app entry (ProviderScope)
  src/
    app.dart                    MaterialApp.router + themes
    config.dart                 API_BASE resolution (no secrets)
    theme/                      brand colors, light/dark themes, logo mark
    i18n/                       EN/AZ/RU/TR strings (web-parity keys), switcher
    state/                      Riverpod controllers (settings, plan, …)
    api/                        Dio client + models + endpoint methods
    router/                     go_router config
    features/                   home, plan (form, result, roadmap, map, chat)
    widgets/                    shared UI
assets/icon/                    brand icon source (svg) + generated png
test/                           unit/widget tests
```

Business logic lives on the server; the app shows only what the API returns.
Prices/ratings follow the same guardrail as the web app — nothing is invented.
