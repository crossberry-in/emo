# emo

> Expo-like Android development framework in Go — write Go, see it live on Android, no JavaScript, no Gradle rebuilds.

`emo` is an open-source framework that brings the Expo Go live-reload experience to Android development, but instead of JavaScript/TypeScript you write **pure Go**. Your Go component functions produce a virtual view tree (vtree); the dev server pushes that tree over a WebSocket to the **emo Go** preview app on your Android device or emulator, which renders it as native Jetpack Compose UI. Save a file — your device updates in under a second.

---

## Why emo?

| Approach | Reload time | Language | Native UI? |
|---|---|---|---|
| Native Android (Kotlin + Gradle) | 5–20 s | Kotlin | ✅ |
| Expo (React Native) | <1 s | JS/TS | ❌ Bridge |
| Flutter | 1–2 s | Dart | ❌ Custom renderer |
| **emo** | **<1 s** | **Go** | ✅ Jetpack Compose |

emo combines Go's static typing and fast compile with a React-like component model and Expo-style preview app. You get the hot-reload loop you love from web development, with truly native Android UI.

---

## Architecture

```
┌─────────────────────┐         WebSocket (vtree JSON)         ┌──────────────────────┐
│  emo dev server     │ ◄────────────────────────────────────► │  emo Go preview app  │
│  (Go binary)        │   events back (click, change, ...)     │  (Android APK)       │
│                     │                                        │                      │
│  • File watcher     │                                        │  • Kotlin            │
│  • DSL → vtree      │                                        │  • Jetpack Compose   │
│  • Codegen → Kotlin │                                        │  • VTreeRenderer     │
│  • Hot function swap│                                        │  • EmoClient (WS)    │
│  • Plugin transport  │                                        │  • Plugin bridges    │
└──────────┬──────────┘                                        └──────────┬───────────┘
           │                                                              │
           │ adb install / am start                                       │ native calls
           │                                                              │ (camera, GPS, …)
           ▼                                                              ▼
   ┌────────────────┐                                            ┌────────────────┐
   │ Android device │                                            │  Device APIs   │
   │  or emulator   │                                            │  (CameraX, …)  │
   └────────────────┘                                            └────────────────┘
```

### Components

| Package | Purpose |
|---|---|
| [`dsl/`](dsl/) | React-like Go UI DSL: `Column`, `Row`, `Text`, `Button`, `UseState`, `UseEffect`. Produces a JSON-serialisable `Element` tree. |
| [`codegen/`](codegen/) | Translates an emo vtree into Kotlin Jetpack Compose source. Also diffs two vtrees for incremental hot-swap patches. |
| [`server/`](server/) | The dev server: file watcher (fsnotify), WebSocket hub, state-mutation scheduler, adb integration, plugin transport. |
| [`plugin/`](plugin/) | Plugin registry + built-in plugins: `camera`, `location`, `storage`, `vibration`. Third-party plugins register at init time. |
| [`runtime/`](runtime/) | Shared wire-protocol types (Message, VTreePayload, EventPayload, …) used by both the server and the Android preview app. |
| [`cli/`](cli/) + [`cmd/emo`](cmd/emo/) | The `emo` CLI: `init`, `start`, `build`, `go`, `plugins`. |
| [`android/`](android/) | The **emo Go** preview app — Kotlin / Jetpack Compose. Install once, use with any emo project. |
| [`examples/counter/`](examples/counter/) | Minimal counter demo. |

---

## Install

### One-line install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash
```

This downloads a pre-built binary for your platform (Linux/macOS/Windows × amd64/arm64) from the [latest release](https://github.com/crossberry-in/emo/releases) and installs it to `~/.local/bin/emo`.

If no pre-built binary is available, the script falls back to building from source (requires Go 1.22+ and git).

**Install a specific version:**

```bash
curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash -s -- v0.1.0
```

**Install to a custom directory:**

```bash
EMO_INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash
```

**Authenticated install (avoids GitHub API rate limits):**

```bash
GITHUB_TOKEN=ghp_xxx curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash
```

### Build from source

```bash
git clone https://github.com/crossberry-in/emo
cd emo
go build -o /usr/local/bin/emo ./cmd/emo
```

Requirements:
- Go 1.22+
- Android SDK + Platform Tools (adb) — for `emo go`
- Android Studio or standalone Gradle — for building the emo Go preview APK once

### Build the emo Go preview app

```bash
cd android
./gradlew :app:assembleDebug
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

You only do this once. The same preview app works for every emo project.

---

## Quickstart

```bash
emo init myapp
cd myapp
emo start
```

In another terminal, with an emulator running (`emulator -avd Pixel_7`) or a device connected via USB:

```bash
emo go --apk /path/to/emo-go.apk
```

The emo Go preview app launches on your device, auto-connects to your dev server, and renders your `App()` component. Edit `App.go` and save — your UI updates instantly.

---

## The emo DSL

emo's DSL follows the React mental model: components are pure functions that take props and return a vtree; side-effects live in hooks.

```go
package main

import (
    "fmt"
    "github.com/emo-framework/emo/dsl"
)

func App() dsl.Element {
    // useState: declare reactive state. The setter triggers a re-render.
    count, setCount := dsl.UseStateInt(0)
    name, setName := dsl.UseStateString("World")

    // useEffect: schedule side-effects. Re-runs when deps change.
    dsl.UseEffect(func() {
        fmt.Println("count is now", count)
    }, count)

    return dsl.Scaffold(
        dsl.Prop("title", "my app"),
        dsl.Children(
            dsl.Column(
                dsl.Spacing(16),
                dsl.Padding(24),
                dsl.Children(
                    dsl.Text(fmt.Sprintf("Hello, %s!", name), dsl.Font(28, "bold")),

                    // TextField with onChange handler
                    dsl.TextField("Your name", dsl.OnChange(func(v string) {
                        setName(v)
                    })),

                    // Button with onClick handler
                    dsl.Button("Increment", dsl.OnClick(func() {
                        setCount(count + 1)
                    })),

                    dsl.Text(fmt.Sprintf("Count: %d", count), dsl.Font(18, "normal")),
                ),
            ),
        ),
    )
}

func main() { emoServe(App) }
var emoServe = func(root func() dsl.Element) {}
```

### Element constructors

| Function | Maps to (Compose) | Notes |
|---|---|---|
| `dsl.Scaffold(...)` | `Scaffold` | Top-level wrapper |
| `dsl.Column(...)` | `Column` | Vertical stack |
| `dsl.Row(...)` | `Row` | Horizontal stack |
| `dsl.View(...)` | `Box` | Generic container |
| `dsl.Text(s, ...)` | `Text` | Label |
| `dsl.Button(label, ...)` | `Button { Text(...) }` | Clickable button |
| `dsl.TextField(placeholder, ...)` | `TextField` | Editable text |
| `dsl.Image(src, ...)` | `Image` | Network/asset image (via Coil) |
| `dsl.Spacer(...)` | `Spacer` | Fills available space |
| `dsl.Divider(...)` | `HorizontalDivider` | Visual separator |

### Style options

| Option | Type | Description |
|---|---|---|
| `dsl.Padding(dp)` | float64 | Padding in dp |
| `dsl.Margin(dp)` | float64 | Margin (emulated; Compose has no native margin) |
| `dsl.Bg("#FFFFFFFF")` | hex string | Background colour |
| `dsl.Fg("#FF0000FF")` | hex string | Foreground/text colour |
| `dsl.Spacing(dp)` | float64 | Gap between Column/Row children |
| `dsl.Width(v)` / `dsl.Height(v)` | float64 \| int \| "match" \| "wrap" | Explicit dimensions |
| `dsl.Font(size, weight)` | (sp, "normal"\|"bold") | Typography |
| `dsl.Prop(key, value)` | any | Arbitrary prop for custom elements |
| `dsl.Children(...)` | `...Element` | Append child elements |

### Event handlers

| Option | Triggers |
|---|---|
| `dsl.OnClick(func())` | Button click |
| `dsl.OnChange(func(string))` | TextField text change |

Handlers are registered with the runtime and referenced by opaque tokens in the vtree. When the user taps a button on Android, emo Go sends `{kind:"event", payload:{token:"el_xxx", event:"click"}}` over the WebSocket; the dev server dispatches it to the registered Go func. Your handler runs in Go, mutates state, and emo pushes a new vtree back to the device.

### Hooks

| Hook | Signature | Description |
|---|---|---|
| `dsl.UseState(initial any)` | `(any, func(any))` | Reactive state |
| `dsl.UseStateInt(initial int)` | `(int, func(int))` | Typed convenience |
| `dsl.UseStateString(initial string)` | `(string, func(string))` | Typed convenience |
| `dsl.UseEffect(fn func(), deps ...any)` | | Side-effect after commit; re-runs when any dep changes |

---

## Live reload, hot function swap, and state

emo's live-reload pipeline has three tiers:

1. **State mutation** — When `setCount(...)` is called, emo re-renders the root component, diffs the old and new vtrees, and pushes a patch to the device. This is the fastest path: **<100 ms** round-trip on a local emulator.

2. **File save** — When you save `App.go`, fsnotify fires, the dev server debounces for 150 ms, then re-renders and pushes the new vtree. Reload time: **~200–500 ms** depending on tree size.

3. **Hot function swap** *(roadmap)* — When you change Go code that affects non-DSL logic (helper functions, business logic), emo recompiles the project as a Go plugin (Linux) or fork-and-execs the new binary and migrates state. The current open-source preview re-runs the existing `RootFactory` on file save; full hot-swap requires the `emo start` subprocess to be reloaded, which is the next milestone.

State is preserved across vtree pushes because the dev server holds the `hookFrame` in memory between renders. Only a full binary restart resets state.

---

## Plugin system

Plugins extend emo with native device capabilities. Built-in plugins:

| Plugin | Methods | Kotlin counterpart |
|---|---|---|
| `camera` | `takePhoto(quality int) → base64`, `requestPermission() → bool` | CameraX |
| `location` | `getCurrentPosition(highAccuracy bool) → {lat,lng,accuracy,ts}`, `startWatch() → stream` | FusedLocationProvider |
| `storage` | `get(key) → string`, `set(key, value)`, `remove(key)` | SharedPreferences |
| `vibration` | `vibrate(ms int)` | Vibrator |

### Invoke a plugin from Go

```go
func App() dsl.Element {
    photo, setPhoto := dsl.UseStateString("")

    return dsl.Column(
        dsl.Children(
            dsl.Button("Take photo", dsl.OnClick(func() {
                plugin.Invoke("camera", "takePhoto", map[string]any{"quality": 80},
                    func(result any, err error) {
                        if err != nil { return }
                        setPhoto(result.(string))
                    })
            })),
            dsl.Image(photo),
        ),
    )
}
```

### Author a third-party plugin

```go
package myplugin

import "github.com/emo-framework/emo/plugin"

type FooPlugin struct{}
func (FooPlugin) Name() string { return "foo" }
func (FooPlugin) Methods() []plugin.Method {
    return []plugin.Method{{
        Name:   "bar",
        Params: []plugin.Param{{Name: "x", Type: "int"}},
        Return: "int",
        Invoke: func(params map[string]any, reply func(any, error)) {
            x := params["x"].(int)
            reply(x*2, nil)
        },
    }}
}

func init() { plugin.Register(FooPlugin{}) }
```

Add to `emo.toml`:
```toml
[plugins]
foo = "github.com/me/emo-plugin-foo"
```

The Kotlin counterpart must be packaged inside the emo Go preview app (or a separate installable APK). For the open-source preview, only the four built-in plugins have Kotlin counterparts shipped.

---

## `emo.toml` reference

```toml
name = "myapp"
package = "dev.emo.myapp"
version = "0.1.0"

[dev]
port = 7575                  # 0 = auto-pick
watch = "."                  # directory to watch for live reload

[build]
output = "build/app.apk"     # `emo build` output path
kotlinPackage = "dev.emo.myapp"

[plugins]
camera = true
location = true
storage = true
vibration = true
# foo = "github.com/me/emo-plugin-foo"
```

---

## CLI reference

| Command | Description |
|---|---|
| `emo init <name>` | Scaffold a new emo project (creates `App.go`, `emo.toml`, `.gitignore`) |
| `emo start [--port 7575] [--watch .] [--launch] [--apk path]` | Start the dev server with live reload. `--launch` auto-installs and launches emo Go on a connected device |
| `emo go [--apk path]` | Install (if `--apk` given) and launch emo Go preview app on a connected Android device |
| `emo build [--out app.apk] [--package dev.emo.x]` | Generate Kotlin source for a standalone APK. Drives Gradle if the template is present |
| `emo plugins` | List registered plugins |

---

## Project layout

```
myapp/
├── App.go          # your root component
├── emo.toml        # project config
├── .gitignore
├── components/     # (optional) more component files — all .go files are watched
└── .emo/           # generated; do not commit
    ├── build/
    │   └── EmoRoot.kt  # generated Kotlin for `emo build`
    └── App.so       # (Linux only) plugin for hot function swap
```

---

## How live reload actually works

1. You run `emo start`. The CLI loads your `App.go` and installs `RootFactory` on the dev server.
2. The dev server calls `RootFactory()` to produce the initial vtree.
3. The vtree is JSON-encoded and pushed over WebSocket to every connected emo Go client.
4. emo Go's `VTreeRenderer` walks the JSON and emits Compose `@Composable` calls. The UI appears on screen.
5. You tap a button. emo Go sends `{kind:"event", token:"el_…"}` back over the WebSocket.
6. The dev server calls `dsl.InvokeHandler(token, nil)`, which runs your Go `onClick` closure.
7. Your closure calls `setCount(count + 1)`. The setter mutates state and calls `ScheduleReRender()`.
8. The dev server re-invokes `RootFactory()`, diffs old vs. new vtree, and pushes only the changed ops.
9. emo Go applies the patch in place — no activity restart, no recomposition of unchanged subtrees.

For file saves, the only difference is step 1: fsnotify fires `Reload("file:App.go")` which re-runs the pipeline from step 2 onward. Because the Go binary itself isn't reloaded, all in-memory state (hooks, registries) is preserved.

---

## Roadmap

- [ ] **Full hot function swap** — Go `plugin` package on Linux; fork-and-exec with state migration on macOS/Windows
- [ ] **Patch application** — emo Go currently replaces the whole tree on every patch; switch to in-place mutation for sub-100 ms diffs
- [ ] **Multi-screen navigation** — `dsl.Navigator` with stack-based routing
- [ ] **Forms & validation** — `dsl.Form`, `dsl.Validator`
- [ ] **Animated transitions** — `dsl.AnimatedVisibility`, `dsl.AnimatedContent`
- [ ] **Codegen for non-Compose backends** — XML views (for legacy Android), Wear OS Compose
- [ ] **emo publish** — push to Play Store via Gradle `assembleRelease` + signing
- [ ] **emo cloud** — hosted dev server for sharing preview sessions (like Expo Snack)

---

## Limitations of the open-source preview

This is a working framework skeleton designed for community contribution. Known gaps:

1. **Hot function swap** currently re-renders the existing `RootFactory`; editing helper functions outside the DSL requires restarting `emo start`. Full plugin-based hot-swap is the top roadmap item.
2. **Plugin Kotlin counterparts** are stubs in the Android app — `camera.takePhoto`, `location.getCurrentPosition`, etc. need real CameraX / FusedLocation implementations to work end-to-end.
3. **`emo build`** generates Kotlin source but does not drive Gradle automatically; you wire the generated `EmoRoot.kt` into an Android Studio project manually. A Gradle template under `templates/android-standalone` is planned.
4. **Codegen style** is intentionally verbose — each `Column`, `Text`, `Button` becomes an explicit Compose call. Optimisations (inlining, hoisting) are planned.

Pull requests welcome. See `ROADMAP.md` (planned) and the issues board.

---

## License

MIT — see `LICENSE`.

---

## Acknowledgements

emo stands on the shoulders of giants:
- **Expo** for the dev-server-and-preview-app architecture
- **React** for the component + hooks mental model
- **Jetpack Compose** for the native Android UI layer
- **gomobile** for proving Go on Android is viable
