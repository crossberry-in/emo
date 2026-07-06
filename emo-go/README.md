# emo Go

> The Expo Go equivalent for the [emo](https://github.com/crossberry-in/emo) framework.

emo Go is a generic preview app for Android that connects to any emo dev server and renders the live vtree as native Jetpack Compose UI. Install it once and use it with any emo project.

## Features

- **Auto-detect dev servers** — scans your LAN for emo dev servers on port 7575
- **Carousel of recent projects** — tap a card to connect
- **Live preview** — renders vtree from the dev server as native Compose UI
- **In-app updates** — notification when a new version is available, tap to update
- **Full UI element support** — WebView, Text, Button, Input, Switch, Slider, Card, and 20+ more
- **Built with ♥️ by crossberry**

## Install

### Download the APK

Download the latest APK from the [releases page](https://github.com/crossberry-in/emo-go/releases).

### Build from source

```bash
git clone https://github.com/crossberry-in/emo-go
cd emo-go
./gradlew assembleDebug
# APK at app/build/outputs/apk/debug/app-debug.apk
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

## Usage

1. Start the emo dev server on your computer:
   ```bash
   emo start
   ```

2. Open **emo Go** on your Android device.

3. The app auto-detects dev servers on your WiFi network. Tap a project card to connect.

4. Edit your `.em` files and save — the device updates instantly.

## Architecture

```
┌─────────────────────┐  WebSocket (vtree JSON)  ┌──────────────────────┐
│  emo dev server     │ ◄──────────────────────► │  emo Go app          │
│  (on your computer) │   events (click, etc.)   │  (on Android device) │
└─────────────────────┘                          └──────────────────────┘
                                                          │
                                                          │ Jetpack Compose
                                                          ▼
                                                   ┌──────────────┐
                                                   │  Native UI   │
                                                   └──────────────┘
```

## Built with

- Kotlin + Jetpack Compose
- OkHttp (WebSocket client)
- Coil (image loading)
- Material 3 components

## License

MIT

## Built with ♥️ by [crossberry](https://crossberry.vercel.app)
