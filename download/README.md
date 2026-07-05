# emo — Deliverables

## Files in this directory

| File | Size | Description |
|---|---|---|
| `emo-go-preview.apk` | 8.5 MB | The emo Go preview app — install on Android to see live-reload in action |
| `emo-framework.zip` | 90 KB | Full emo framework source (Go DSL, dev server, codegen, plugins, CLI, Android preview app source, examples, README) |

---

## Install the emo Go preview APK

### Option A — adb (recommended)

```bash
adb install -r emo-go-preview.apk
```

The app will appear in your launcher as **emo Go**.

### Option B — Manual

Copy `emo-go-preview.apk` to your Android device (USB, cloud drive, etc.) and tap to install. You may need to enable "Install unknown apps" for your file manager.

### Requirements

- Android 8.0 (API 26) or newer
- Internet permission is auto-granted on first launch

---

## Use the emo Go preview app

### Method 1 — Connect manually

1. Start the emo dev server on your computer:
   ```bash
   unzip emo-framework.zip
   cd emo
   go build -o /usr/local/bin/emo ./cmd/emo
   emo init myapp && cd myapp
   emo start --port 7575
   ```
2. Find your computer's LAN IP (e.g. `192.168.1.10`).
3. Open **emo Go** on your device.
4. Enter `ws://192.168.1.10:7575/ws` and tap **Connect**.

### Method 2 — Auto-launch via `emo go`

```bash
emo go --apk emo-go-preview.apk
```

This installs the APK (if not already installed) and launches it with the dev server URL pre-filled via `am start --es emo_server ws://...`.

---

## Build the emo CLI from source

```bash
unzip emo-framework.zip
cd emo
go build -o /usr/local/bin/emo ./cmd/emo
emo --help
```

Requires Go 1.22+.

---

## Rebuild the APK from source

```bash
unzip emo-framework.zip
cd emo/android
# Requires Android SDK + Java 17+
./gradlew :app:assembleDebug
# Output: app/build/outputs/apk/debug/app-debug.apk
```

---

## What's inside the framework ZIP

```
emo/
├── README.md             Full documentation
├── LICENSE               MIT
├── go.mod / go.sum       Go module
├── cmd/emo/              CLI entry point
├── cli/                  emo init / start / build / go / plugins
├── dsl/                  React-like Go UI DSL + hooks
├── codegen/              Go vtree → Kotlin Jetpack Compose + diff
├── server/               Dev server (WebSocket, file watcher, adb)
├── plugin/               Built-in plugins (camera, location, storage, vibration)
├── runtime/              Wire protocol types
├── android/              emo Go preview app source (Kotlin + Compose)
└── examples/counter/     Counter demo
```

---

## Troubleshooting

**APK won't install** — Make sure "Install unknown apps" is enabled for your file manager in Settings → Apps → Special access.

**emo Go can't connect to dev server** — Verify both device and computer are on the same Wi-Fi. Check that your computer's firewall allows inbound TCP on port 7575 (or whichever port `emo start` reports). On the computer, run `curl http://localhost:7575/manifest` to confirm the server is up.

**`emo go` says "adb not found"** — Install Android Platform Tools: `brew install android-platform-tools` (macOS) or download from developer.android.com/studio/releases/platform-tools.

**`emo start` says "go not found"** — Install Go 1.22+ from go.dev/dl.

---

Enjoy building Android apps in Go with live reload! Edit `App.go`, save, and watch your device update instantly.
