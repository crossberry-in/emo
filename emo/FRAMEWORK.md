# emo Framework — Complete Documentation

> Expo-like Android development framework in Go. Write `.em` files, see them live on Android. No JavaScript, no Gradle rebuilds.

**Version**: 0.1.2
**License**: MIT
**GitHub**: https://github.com/crossberry-in/emo
**Templates**: https://github.com/crossberry-in/emo-templates

---

## Table of Contents

1. [Quick Start](#1-quick-start)
2. [Installation](#2-installation)
3. [Project Structure](#3-project-structure)
4. [The `.em` Language](#4-the-em-language)
5. [CSS Styling](#5-css-styling)
6. [Native UI Elements](#6-native-ui-elements)
7. [State and Hooks](#7-state-and-hooks)
8. [File-Based Routing](#8-file-based-routing)
9. [Live Reload](#9-live-reload)
10. [Components and Templates](#10-components-and-templates)
11. [Plugin System](#11-plugin-system)
12. [Android Integration](#12-android-integration)
13. [CLI Reference](#13-cli-reference)
14. [Building APKs](#14-building-apks)
15. [Configuration](#15-configuration)
16. [Troubleshooting](#16-troubleshooting)
17. [Architecture](#17-architecture)

---

## 1. Quick Start

### Install emo

```bash
curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash
```

### Create your first app

```bash
emo create my-app
cd my-app
emo start
```

### Run on Android

In another terminal, with an Android emulator running or a device connected via `adb`:

```bash
emo go
```

Edit any `.em` file and save — your device updates instantly.

---

## 2. Installation

### One-line install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash
```

Installs to `~/.local/bin/emo`. Works on Linux, macOS, and Windows (amd64/arm64).

### Install a specific version

```bash
curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash -s -- v0.1.2
```

### Install to a custom directory

```bash
EMO_INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash
```

### Build from source

```bash
git clone https://github.com/crossberry-in/emo
cd emo
go build -o /usr/local/bin/emo ./cmd/emo
```

Requires Go 1.22+.

### Verify installation

```bash
emo --help
emo templates
emo components
```

---

## 3. Project Structure

When you run `emo create my-app` or `emo init my-app`, you get a full project:

```
my-app/
├── app/                         # File-based routing (like Expo Router)
│   ├── (tabs)/                  # Tab navigation group
│   │   ├── _layout.em           # Tab bar layout
│   │   ├── index.em             # Home screen
│   │   └── explore.em           # Explore screen
│   ├── _layout.em               # Root layout
│   ├── index.em                 # Entry point (emo start loads this)
│   ├── modal.em                 # Modal screen
│   └── +not-found.em            # 404 screen
├── components/                  # Reusable components
│   ├── themed-text.em           # Themed text (title/body/link)
│   ├── themed-view.em           # Themed container
│   ├── hello-wave.em            # Wave greeting
│   ├── external-link.em         # External link
│   └── ui/
│       ├── collapsible.em       # Collapsible section
│       └── icon-symbol.em       # Icon
├── hooks/                       # Custom hooks
│   ├── use-color-scheme.em
│   └── use-theme-color.em
├── scripts/
│   └── reset-project.em
├── assets/
│   └── images/                  # App icons and images
├── android/                     # Per-project native Android project
│   ├── app/
│   │   ├── build.gradle.kts     # Personalized with app name
│   │   └── src/main/
│   │       ├── AndroidManifest.xml
│   │       └── java/dev/emo/app/
│   │           └── MainActivity.kt   # Connects to emo dev server
│   ├── build.gradle.kts
│   ├── settings.gradle.kts
│   └── gradle.properties
├── emo.json                     # App config (like Expo's app.json)
├── emo.toml                     # Project config
├── README.md, CLAUDE.md, AGENTS.md
└── .gitignore
```

### Entry point

`emo start` looks for files in this order:

1. `app/index.em` (preferred — like Expo Router)
2. `App.em` (legacy)
3. The first `.em` file in the project root

---

## 4. The `.em` Language

`.em` is emo's custom language — a Svelte/Vue-style single-file component format with JSX-like syntax.

### Basic component

```em
// Counter.em
component Counter {
  state count = 0

  render {
    <Column spacing={16} padding={24}>
      <Text fontSize={28} fontWeight="bold">Counter</Text>
      <Text fontSize={18}>Count: {count}</Text>
      <Button onClick={() => count = count + 1}>Increment</Button>
      <Button onClick={() => count = 0}>Reset</Button>
    </Column>
  }
}

style "./Counter.css"
```

### Syntax overview

| Feature | Syntax |
|---|---|
| Component declaration | `component Name { ... }` |
| State declaration | `state count = 0` |
| State with type | `state name: string = "World"` |
| Render block | `render { <JSX> }` |
| Element | `<Tag attrs>children</Tag>` |
| Self-closing | `<Tag attrs />` |
| Expression | `{expression}` |
| String attribute | `title="Hello"` |
| Number attribute | `spacing={16}` |
| Expression attribute | `onClick={() => count + 1}` |
| CSS reference | `style "./Component.css"` |
| Import | `import { Header } from "./Header.em"` |

### Imports

```em
import { Header } from "./components/Header.em"
import { Card, Modal } from "./components/index.em"

component App {
  render {
    <Column>
      <Header />
      <Card title="Hello">
        <Text>Content</Text>
      </Card>
    </Column>
  }
}
```

### Comments

```em
// Line comment

/* Block comment */
```

---

## 5. CSS Styling

emo uses standard CSS syntax with `dp`/`sp` units for dimensions.

### Example CSS

```css
/* App.css */
.container {
  background: #FFFFFF;
  padding: 24dp;
  spacing: 16dp;
}

.title {
  font-size: 28sp;
  font-weight: bold;
  color: #333333;
}

.button-row {
  spacing: 8dp;
}
```

### Using CSS in `.em`

```em
<Column className="container">
  <Text className="title">Hello</Text>
  <Row className="button-row">
    <Button>OK</Button>
    <Button>Cancel</Button>
  </Row>
</Column>
```

### Supported CSS properties

| CSS property | emo DSL prop | Type |
|---|---|---|
| `background` | `dsl.Bg()` | hex color |
| `color` | `dsl.Fg()` | hex color |
| `padding` | `dsl.Padding()` | dp |
| `margin` | `dsl.Margin()` | dp |
| `spacing` | `dsl.Spacing()` | dp (Column/Row only) |
| `font-size` | `dsl.Font(size, weight)` | sp |
| `font-weight` | `dsl.Prop("fontWeight", ...)` | "normal" / "bold" |
| `width` | `dsl.Width()` | dp / "match" / "wrap" |
| `height` | `dsl.Height()` | dp / "match" / "wrap" |

### Units

- `dp` — density-independent pixels (use for padding, spacing, dimensions)
- `sp` — scale-independent pixels (use for font sizes)
- `px` — treated as dp
- Plain numbers (e.g. `24`) — treated as dp

---

## 6. Native UI Elements

emo supports 27 native UI elements that render as real Jetpack Compose components on Android.

### Layout elements

| Element | Description | Example |
|---|---|---|
| `<Column>` | Vertical stack | `<Column spacing={16}><Text>A</Text><Text>B</Text></Column>` |
| `<Row>` | Horizontal stack | `<Row spacing={8}><Text>A</Text><Text>B</Text></Row>` |
| `<View>` | Generic container | `<View><Text>Content</Text></View>` |
| `<Scaffold>` | Top-level layout | `<Scaffold><Column>...</Column></Scaffold>` |
| `<SafeAreaView>` | Respects status bar insets | `<SafeAreaView><Column>...</Column></SafeAreaView>` |
| `<ScrollView>` | Scrollable container | `<ScrollView><Column>...</Column></ScrollView>` |
| `<Card>` | Material card | `<Card><Text>Content</Text></Card>` |
| `<Spacer>` | Fills space | `<Spacer />` |
| `<Divider>` | Horizontal line | `<Divider />` |

### Text elements

| Element | Description | Example |
|---|---|---|
| `<Text>` | Label | `<Text fontSize={18}>Hello</Text>` |
| `<Button>` | Clickable button | `<Button onClick={() => ...}>Click</Button>` |
| `<Input>` | Single-line text input | `<Input placeholder="Type..." />` |
| `<TextField>` | Same as Input | `<TextField placeholder="Name" />` |

### Interactive elements

| Element | Description | Example |
|---|---|---|
| `<Switch>` | Toggle on/off | `<Switch value={true} />` |
| `<Slider>` | Range input (0..1) | `<Slider value={0.5} />` |
| `<Picker>` | Dropdown | `<Picker options={["A", "B", "C"]} />` |
| `<Checkbox>` | Checkbox | `<Checkbox value={true} />` |
| `<RadioButton>` | Radio button | `<RadioButton value={true} />` |

### Media elements

| Element | Description | Example |
|---|---|---|
| `<Image>` | Image (URL or asset) | `<Image source="https://..." />` |
| `<WebView>` | Embed web pages | `<WebView source="https://expo.dev" />` |
| `<Icon>` | Material icon | `<Icon name="home" />` |

### Loading indicators

| Element | Description | Example |
|---|---|---|
| `<ActivityIndicator>` | Circular spinner | `<ActivityIndicator />` |
| `<Progress>` | Linear progress bar | `<Progress value={0.7} />` |

### Navigation elements

| Element | Description | Example |
|---|---|---|
| `<TopBar>` | Top app bar | `<TopBar title="My App" />` |
| `<TabBar>` | Tab row with content | `<TabBar><Column>...</Column></TabBar>` |
| `<BottomNav>` | Bottom navigation | `<BottomNav>...</BottomNav>` |
| `<Fab>` | Floating action button | `<Fab onClick={() => ...} />` |

### Common attributes

| Attribute | Type | Description |
|---|---|---|
| `spacing` | number | Gap between children (Column/Row) |
| `padding` | number | Inner padding in dp |
| `background` | hex string | Background color |
| `color` | hex string | Text/foreground color |
| `fontSize` | number | Font size in sp |
| `fontWeight` | "normal" / "bold" | Font weight |
| `width` / `height` | number / "match" / "wrap" | Dimensions |
| `className` | string | CSS class name |
| `source` | string | URL (Image/WebView) |
| `value` | any | Current value (Switch/Slider/Progress) |
| `placeholder` | string | Placeholder text (Input) |
| `options` | string[] | Picker options |
| `name` | string | Icon name |
| `title` | string | TopBar title |

### Event handlers

| Event | Elements | Syntax |
|---|---|---|
| `onClick` | Button, Fab, RadioButton | `onClick={() => count + 1}` |
| `onChange` | Input, TextField, Picker | `onChange={v => name = v}` |
| `onToggle` | Switch | `onToggle={v => enabled = v}` |
| `onSlide` | Slider | `onSlide={v => value = v}` |

---

## 7. State and Hooks

emo uses React-like hooks for state management.

### `state` declaration (in `.em` files)

```em
component App {
  state count = 0                    // int (inferred)
  state name = "World"               // string (inferred)
  state items: int = 42              // explicit type
  state enabled: bool = true         // explicit type
  state price: float = 9.99          // explicit type

  render {
    <Column>
      <Text>{count}</Text>
      <Text>{name}</Text>
    </Column>
  }
}
```

### State mutation

State is mutated by calling the setter. In `.em` files, assignment is automatically rewritten:

```em
// This:
<Button onClick={() => count = count + 1}>+</Button>

// Is rewritten to:
dsl.OnClick(func() { setCount(count + 1) })
```

### Hooks (in Go DSL code)

```go
func App() dsl.Element {
    // UseStateInt — typed state
    count, setCount := dsl.UseStateInt(0)
    name, setName := dsl.UseStateString("World")

    // UseState — untyped state
    data, setData := dsl.UseState(nil)

    // UseEffect — side-effects after render
    dsl.UseEffect(func() {
        log.Println("count changed:", count)
    }, count)

    return dsl.Column(
        dsl.Children(
            dsl.Text(fmt.Sprintf("Count: %d", count)),
            dsl.Button("Increment", dsl.OnClick(func() {
                setCount(count + 1)
            })),
        ),
    )
}
```

---

## 8. File-Based Routing

emo supports Expo Router-style file-based routing via the `app/` directory.

### Route conventions

| File | Route | Description |
|---|---|---|
| `app/index.em` | `/` | Home screen (entry point) |
| `app/(tabs)/_layout.em` | — | Tab navigation layout |
| `app/(tabs)/index.em` | `/` (inside tabs) | Tab home screen |
| `app/(tabs)/explore.em` | `/explore` | Explore screen |
| `app/modal.em` | `/modal` | Modal screen |
| `app/+not-found.em` | `*` | 404 screen |
| `app/_layout.em` | — | Root layout |

### Example: `app/index.em`

```em
import { TabsLayout } from "./(tabs)/_layout.em"

component App {
  render {
    <TabsLayout />
  }
}
```

### Example: `app/(tabs)/_layout.em`

```em
component TabsLayout {
  state activeTab = "home"
  render {
    <Column>
      <Column className="content">
        <Text>{activeTab}</Text>
      </Column>
      <Row className="tabBar">
        <Button onClick={() => activeTab = "home"}>Home</Button>
        <Button onClick={() => activeTab = "explore"}>Explore</Button>
        <Button onClick={() => activeTab = "profile"}>Profile</Button>
      </Row>
    </Column>
  }
}
```

---

## 9. Live Reload

emo's live reload works in three tiers:

### 1. State mutation (< 100ms)

When `setCount(...)` is called, emo re-renders the root component, diffs the old and new vtrees, and pushes a patch to the device.

### 2. File save (~200-500ms)

When you save a `.em` or `.css` file:
1. `fsnotify` detects the change
2. Dev server debounces for 150ms
3. Re-transpiles the `.em` file
4. Pushes the new vtree to all connected devices

### 3. Hot function swap (roadmap)

Editing helper functions outside the DSL requires restarting `emo start`. Full hot-swap is the top roadmap item.

### How it works

```
Save .em file
    ↓
fsnotify fires
    ↓
Dev server re-transpiles .em → vtree
    ↓
WebSocket push to device
    ↓
emo Go app applies vtree as Jetpack Compose
    ↓
UI updates (no app restart, no recomposition of unchanged parts)
```

---

## 10. Components and Templates

### List available templates

```bash
emo templates
```

### Create from template

```bash
emo init myapp --template counter
emo create myapp              # uses "default" template
```

### Available templates

| Template | Description |
|---|---|
| `default` | Full project with routing, components, hooks, android/ |
| `blank` | Minimal starting point |
| `counter` | Counter app with state |
| `todo` | Todo list with TextField |
| `navigation` | Multi-screen app with bottom nav |

### Install components

```bash
emo components              # list all 22 components
emo install WebView         # install a component
emo install Card
emo install SafeAreaView
```

Components are installed to `components/<name>/` in your project.

### Using installed components

```em
import { WebView } from "./components/webview/WebView.em"

component App {
  render {
    <WebView source="https://expo.dev" />
  }
}
```

---

## 11. Plugin System

emo has built-in plugins for native device capabilities.

### Available plugins

| Plugin | Methods |
|---|---|
| `camera` | `takePhoto(quality)`, `requestPermission()` |
| `location` | `getCurrentPosition(highAccuracy)`, `startWatch()` |
| `storage` | `get(key)`, `set(key, value)`, `remove(key)` |
| `vibration` | `vibrate(ms)` |

### Using plugins (in Go DSL code)

```go
import "github.com/emo-framework/emo/plugin"

func App() dsl.Element {
    photo, setPhoto := dsl.UseStateString("")

    return dsl.Column(
        dsl.Children(
            dsl.Button("Take photo", dsl.OnClick(func() {
                plugin.Invoke("camera", "takePhoto",
                    map[string]any{"quality": 80},
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

---

## 12. Android Integration

### Per-project `android/` folder

Every emo project includes an `android/` folder with a native Android project:

```
android/
├── app/
│   ├── build.gradle.kts              # Personalized with your app's package name
│   └── src/main/
│       ├── AndroidManifest.xml
│       └── java/dev/emo/app/
│           └── MainActivity.kt       # Connects to emo dev server
├── build.gradle.kts
├── settings.gradle.kts
└── gradle.properties
```

### How the Android app works

1. `emo go` launches the app via `adb am start` with the dev server URL as an Intent extra
2. `MainActivity.kt` opens a WebSocket to the dev server
3. Receives vtree JSON messages
4. Renders vtree as Jetpack Compose UI via `VTreeRenderer.kt`
5. Sends user events (clicks, changes) back to the dev server
6. Dev server dispatches events to Go handlers
7. State mutations trigger a re-render → new vtree pushed to device

### emo Go preview app

The emo Go preview app is a generic vtree renderer — it knows nothing about your specific project. Install it once and use with any emo project.

Download: https://github.com/crossberry-in/emo/releases

---

## 13. CLI Reference

### `emo create [name]`

Create a new emo project with interactive prompts (like `create-expo-app`).

```bash
emo create my-app
emo create my-app --template counter
```

### `emo init [name]`

Create a new emo project from a template.

```bash
emo init myapp                          # uses "default" template
emo init myapp --template counter       # specific template
emo init myapp --package com.example.myapp
```

### `emo start`

Start the dev server with live reload.

```bash
emo start                    # default port 7575 (auto-finds free port if busy)
emo start --port 8080        # specific port
emo start --launch           # auto-launch on connected device
emo start --apk path.apk     # install APK before launching
```

### `emo go`

Install and launch the emo Go preview app on a connected Android device.

```bash
emo go
emo go --apk /path/to/emo-go-preview.apk
```

### `emo build`

Generate Kotlin source for a standalone APK.

```bash
emo build
emo build --out app.apk --package dev.emo.myapp
```

### `emo templates`

List available project templates from GitHub.

### `emo components`

List installable components from GitHub.

### `emo install <name>`

Install a component from the registry.

```bash
emo install WebView
emo install Card
```

### `emo plugins`

List registered plugins.

---

## 14. Building APKs

### Development (live reload)

```bash
emo start     # dev server with hot reload
emo go        # launch on device
```

### Standalone APK

```bash
emo build
```

This generates Kotlin source from your `.em` files in `__emo_gen__/emo_gen.go` and `EmoRoot.kt`. To assemble the APK:

1. Copy `EmoRoot.kt` into an Android Studio project
2. Run `./gradlew assembleDebug`
3. APK is at `app/build/outputs/apk/debug/app-debug.apk`

### Building the emo Go preview APK

```bash
cd android/
./gradlew :app:assembleDebug
# APK at app/build/outputs/apk/debug/app-debug.apk
```

---

## 15. Configuration

### `emo.json` (like Expo's `app.json`)

```json
{
  "emo": {
    "name": "my-app",
    "slug": "my-app",
    "version": "1.0.0",
    "orientation": "portrait",
    "sdkVersion": "0.1",
    "android": {
      "package": "dev.emo.my_app",
      "adaptiveIcon": {
        "backgroundColor": "#E6F4FE"
      },
      "edgeToEdgeEnabled": true
    },
    "plugins": [
      "emo-router",
      "emo-splash-screen"
    ]
  }
}
```

### `emo.toml`

```toml
name = "my-app"
package = "dev.emo.my_app"
version = "0.1.0"
sdkVersion = "0.1"

[dev]
port = 7575
watch = "."

[build]
output = "build/app.apk"
kotlinPackage = "dev.emo.my_app"

[plugins]
camera = true
location = true
storage = true
vibration = true
```

### Environment variables

| Variable | Description |
|---|---|
| `EMO_INSTALL_DIR` | Custom install directory for `emo` binary |
| `GITHUB_TOKEN` | Auth token for GitHub API (avoids rate limits) |
| `EMO_INSECURE` | Set to `1` to skip TLS certificate verification |

---

## 16. Troubleshooting

### `adb not found on PATH`

Install Android Platform Tools:

```bash
# Debian/Ubuntu:
apt-get install -y android-tools-adb

# macOS:
brew install android-platform-tools

# Or download:
# https://developer.android.com/studio/releases/platform-tools
```

### `address already in use`

emo start automatically finds a free port if 7575 is busy. Check the log output for the actual port used.

### `tls: failed to verify certificate: x509: certificate signed by unknown authority`

Install CA certificates:

```bash
# Debian/Ubuntu:
apt-get install -y ca-certificates && update-ca-certificates

# Alpine:
apk add ca-certificates

# CentOS/RHEL:
yum install -y ca-certificates && update-ca-trust
```

Or as a workaround:

```bash
export EMO_INSECURE=1
```

### `emo: command not found` after install

Add `~/.local/bin` to your PATH:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### No device connected

```bash
adb devices    # list connected devices
```

Start an emulator:

```bash
emulator -avd Pixel_7
```

---

## 17. Architecture

```
┌─────────────────────┐         WebSocket (vtree JSON)         ┌──────────────────────┐
│  emo dev server     │ ◄────────────────────────────────────► │  emo Go preview app  │
│  (Go binary)        │   events back (click, change, ...)     │  (Android APK)       │
│                     │                                        │                      │
│  • File watcher     │                                        │  • Kotlin            │
│  • .em → vtree      │                                        │  • Jetpack Compose   │
│  • CSS parsing      │                                        │  • VTreeRenderer     │
│  • Hot reload       │                                        │  • EmoClient (WS)    │
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

### Packages

| Package | Purpose |
|---|---|
| `dsl/` | React-like Go UI DSL: elements, hooks, handlers |
| `eml/` | `.em` language: lexer, parser, AST, codegen |
| `codegen/` | Go vtree → Kotlin Jetpack Compose source + diff |
| `server/` | Dev server: file watcher, WebSocket, ADB, plugins |
| `plugin/` | Plugin registry + built-in plugins |
| `runtime/` | Wire protocol types |
| `cli/` + `cmd/emo/` | The `emo` CLI |
| `android/` | emo Go preview app (Kotlin/Compose) |

### Live reload flow

1. You save a `.em` file
2. `fsnotify` fires → dev server debounces 150ms
3. Re-transpiles `.em` → vtree (in-memory AST evaluation)
4. Pushes vtree JSON over WebSocket to all connected devices
5. emo Go app's `VTreeRenderer` walks the JSON → emits Compose `@Composable` calls
6. UI updates — no activity restart, no recomposition of unchanged subtrees

### Event flow

1. User taps a button on Android
2. emo Go sends `{kind:"event", token:"el_…", event:"click"}` over WebSocket
3. Dev server calls `dsl.InvokeHandler(token, nil)`
4. Your Go `onClick` closure runs
5. Closure calls `setCount(count + 1)` → state mutates
6. `ScheduleReRender()` fires → dev server re-renders root
7. New vtree pushed to device → UI updates

---

## Roadmap

- [ ] Full hot function swap (Go plugin package)
- [ ] Incremental vtree patches (sub-100ms diffs)
- [ ] Multi-screen navigation API
- [ ] Animated transitions
- [ ] Codegen for Wear OS
- [ ] `emo publish` to Play Store
- [ ] `emo cloud` hosted dev sessions

---

## License

MIT — see [LICENSE](LICENSE).

---

## Links

- **Framework**: https://github.com/crossberry-in/emo
- **Templates**: https://github.com/crossberry-in/emo-templates
- **Latest release**: https://github.com/crossberry-in/emo/releases
- **Install**: `curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash`
