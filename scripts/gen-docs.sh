#!/bin/bash
SIDEBAR='    <h3>Getting Started</h3>
    <a href="../index.html">Introduction</a>
    <a href="install.html">Installation</a>
    <a href="quickstart.html">Quick Start</a>
    <a href="project-structure.html">Project Structure</a>
    <h3>The .em Language</h3>
    <a href="em-syntax.html">Syntax Reference</a>
    <a href="css.html">CSS Styling</a>
    <a href="state.html">State &amp; Hooks</a>
    <a href="components.html">Components</a>
    <h3>UI Elements</h3>
    <a href="elements.html">All Elements</a>
    <a href="webview.html">WebView</a>
    <a href="navigation.html">Navigation</a>
    <h3>Features</h3>
    <a href="live-reload.html">Live Reload</a>
    <a href="android-sync.html">Android Sync</a>
    <a href="plugins.html">Plugins</a>
    <a href="haptics.html">Haptics</a>
    <a href="securestore.html">SecureStore</a>
    <a href="linking.html">Linking</a>
    <h3>Deploy</h3>
    <a href="build.html">Building APKs</a>
    <a href="troubleshooting.html">Troubleshooting</a>
    <a href="cli.html">CLI Reference</a>'

generate() {
  local slug="$1"
  local title="$2"
  local body="$3"
  local sidebar=$(echo "$SIDEBAR" | sed "s|href=\"$slug\"|href=\"$slug\" class=\"active\"|")
  cat > "/home/z/my-project/emo/docs/guides/$slug" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>$title — emo docs</title>
<link rel="stylesheet" href="../assets/style.css">
<link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><rect width='100' height='100' rx='22' fill='%236C47FF'/><text x='50' y='72' font-size='64' font-weight='bold' fill='white' text-anchor='middle'>e</text></svg>">
</head>
<body>
<div class="topbar">
  <a href="../index.html" class="logo"><div class="e">e</div>emo</a>
  <div class="nav">
    <a href="https://github.com/crossberry-in/emo">GitHub</a>
    <a href="https://github.com/crossberry-in/emo/releases">Releases</a>
    <a href="https://github.com/crossberry-in/emo-templates">Templates</a>
  </div>
</div>
<div class="layout">
  <nav class="sidebar">
$sidebar
  </nav>
  <main class="content">
$body
  </main>
</div>
<div class="footer">Built with <span class="heart">♥</span> by <a href="https://crossberry.vercel.app">crossberry</a> · MIT License</div>
<script src="../assets/script.js"></script>
</body>
</html>
EOF
  echo "✓ $slug"
}

generate "quickstart.html" "Quick Start" '<h1>Quick Start</h1>
<p>Create your first emo app and see it live on Android in under 5 minutes.</p>
<h2>1. Install emo</h2>
<div class="install-cmd"><span class="prompt">$</span><span class="cmd">curl -fsSL https://raw.githubusercontent.com/crossberry-in/emo/main/install.sh | bash</span><button class="copy">Copy</button></div>
<h2>2. Create your app</h2>
<pre><code>emo create my-app
cd my-app</code></pre>
<h2>3. Start the dev server</h2>
<pre><code>emo start</code></pre>
<h2>4. Launch on Android</h2>
<pre><code>emo go</code></pre>
<h2>5. Edit and see changes instantly</h2>
<pre><code>// app/index.em
component App {
  state count = 0
  render {
    &lt;Column spacing={16} padding={24}&gt;
      &lt;Text fontSize={28} fontWeight="bold"&gt;Hello, emo!&lt;/Text&gt;
      &lt;Text fontSize={48}&gt;{count}&lt;/Text&gt;
      &lt;Button onClick={() =&gt; count = count + 1}&gt;Increment&lt;/Button&gt;
    &lt;/Column&gt;
  }
}
style "./index.css"</code></pre>
<div class="callout"><p><strong>🎉 That'\''s it!</strong> Save the file — your device updates instantly.</p></div>'

generate "project-structure.html" "Project Structure" '<h1>Project Structure</h1>
<p>When you run <code>emo create my-app</code>, you get a full project:</p>
<pre><code>my-app/
├── app/                         # File-based routing
│   ├── (tabs)/                  # Tab navigation group
│   │   ├── _layout.em           # Tab bar layout
│   │   ├── index.em             # Home screen
│   │   └── explore.em           # Explore screen
│   ├── _layout.em               # Root layout
│   ├── index.em                 # Entry point
│   └── +not-found.em            # 404 screen
├── components/                  # Reusable components
├── hooks/                       # Custom hooks
├── android/                     # Native Android project
│   └── app/src/main/java/dev/emo/app/
│       └── MainActivity.kt
├── emo.json                     # App config
└── emo.toml                     # Project config</code></pre>
<h2>Entry point</h2>
<p><code>emo start</code> looks for <code>app/index.em</code> → <code>App.em</code> → first <code>.em</code> file.</p>'

generate "em-syntax.html" "Syntax Reference" '<h1>The .em Language</h1>
<p><code>.em</code> is emo'\''s custom language — Svelte/Vue-style single-file components with JSX-like syntax.</p>
<h2>Basic component</h2>
<pre><code>component Counter {
  state count = 0
  render {
    &lt;Column spacing={16} padding={24}&gt;
      &lt;Text fontSize={28} fontWeight="bold"&gt;Counter&lt;/Text&gt;
      &lt;Text&gt;Count: {count}&lt;/Text&gt;
      &lt;Button onClick={() =&gt; count = count + 1}&gt;Increment&lt;/Button&gt;
    &lt;/Column&gt;
  }
}
style "./Counter.css"</code></pre>
<h2>Syntax overview</h2>
<table>
<tr><th>Feature</th><th>Syntax</th></tr>
<tr><td>Component</td><td><code>component Name { ... }</code></td></tr>
<tr><td>State</td><td><code>state count = 0</code></td></tr>
<tr><td>Typed state</td><td><code>state name: string = "World"</code></td></tr>
<tr><td>Render block</td><td><code>render { &lt;JSX&gt; }</code></td></tr>
<tr><td>Element</td><td><code>&lt;Tag attrs&gt;children&lt;/Tag&gt;</code></td></tr>
<tr><td>Self-closing</td><td><code>&lt;Tag attrs /&gt;</code></td></tr>
<tr><td>Expression</td><td><code>{expression}</code></td></tr>
<tr><td>CSS ref</td><td><code>style "./Component.css"</code></td></tr>
<tr><td>Import</td><td><code>import { Header } from "./Header.em"</code></td></tr>
</table>'

generate "css.html" "CSS Styling" '<h1>CSS Styling</h1>
<p>emo uses standard CSS syntax with <code>dp</code>/<code>sp</code> units.</p>
<h2>Example</h2>
<pre><code>.container {
  background: #FFFFFF;
  padding: 24dp;
  spacing: 16dp;
}
.title {
  font-size: 28sp;
  font-weight: bold;
  color: #333333;
}</code></pre>
<h2>Using CSS in .em</h2>
<pre><code>&lt;Column className="container"&gt;
  &lt;Text className="title"&gt;Hello&lt;/Text&gt;
&lt;/Column&gt;</code></pre>
<h2>Supported properties</h2>
<table>
<tr><th>CSS property</th><th>DSL prop</th><th>Type</th></tr>
<tr><td>background</td><td>Bg()</td><td>hex color</td></tr>
<tr><td>color</td><td>Fg()</td><td>hex color</td></tr>
<tr><td>padding</td><td>Padding()</td><td>dp</td></tr>
<tr><td>spacing</td><td>Spacing()</td><td>dp</td></tr>
<tr><td>font-size</td><td>Font()</td><td>sp</td></tr>
<tr><td>font-weight</td><td>Prop("fontWeight")</td><td>"normal"/"bold"</td></tr>
<tr><td>width</td><td>Width()</td><td>dp/"match"/"wrap"</td></tr>
<tr><td>height</td><td>Height()</td><td>dp/"match"/"wrap"</td></tr>
</table>'

generate "state.html" "State & Hooks" '<h1>State &amp; Hooks</h1>
<p>emo uses React-like hooks for state management.</p>
<h2>state declaration (in .em)</h2>
<pre><code>component App {
  state count = 0
  state name = "World"
  state enabled: bool = true

  render {
    &lt;Text&gt;{count}&lt;/Text&gt;
    &lt;Text&gt;{name}&lt;/Text&gt;
  }
}</code></pre>
<h2>State mutation</h2>
<pre><code>// This:
&lt;Button onClick={() =&gt; count = count + 1}&gt;+&lt;/Button&gt;

// Becomes:
dsl.OnClick(func() { setCount(count + 1) })</code></pre>
<h2>Hook reference</h2>
<table>
<tr><th>Hook</th><th>Signature</th><th>Description</th></tr>
<tr><td>UseState</td><td>(any, func(any))</td><td>Untyped state</td></tr>
<tr><td>UseStateInt</td><td>(int, func(int))</td><td>Typed: int</td></tr>
<tr><td>UseStateString</td><td>(string, func(string))</td><td>Typed: string</td></tr>
<tr><td>UseEffect</td><td>func(), deps...</td><td>Side-effects</td></tr>
</table>'

generate "components.html" "Components" '<h1>Components</h1>
<p>Components are reusable UI blocks defined in <code>.em</code> files.</p>
<h2>Defining a component</h2>
<pre><code>// components/Card.em
component Card {
  state title = ""
  render {
    &lt;Column className="card"&gt;
      &lt;Text fontWeight="bold"&gt;{title}&lt;/Text&gt;
    &lt;/Column&gt;
  }
}
style "./Card.css"</code></pre>
<h2>Importing and using</h2>
<pre><code>import { Card } from "./components/Card.em"

component App {
  render {
    &lt;Card title="Hello" /&gt;
  }
}</code></pre>
<h2>Installing components</h2>
<pre><code>emo install Card
emo install WebView
emo install Modal</code></pre>
<p>Run <code>emo components</code> to see all 22 installable components.</p>'

generate "elements.html" "All Elements" '<h1>Native UI Elements</h1>
<p>emo supports 27 native UI elements that render as Jetpack Compose components.</p>
<h2>Layout</h2>
<table>
<tr><th>Element</th><th>Description</th></tr>
<tr><td>&lt;Column&gt;</td><td>Vertical stack</td></tr>
<tr><td>&lt;Row&gt;</td><td>Horizontal stack</td></tr>
<tr><td>&lt;View&gt;</td><td>Generic container</td></tr>
<tr><td>&lt;Scaffold&gt;</td><td>Top-level layout</td></tr>
<tr><td>&lt;SafeAreaView&gt;</td><td>Respects status bar insets</td></tr>
<tr><td>&lt;ScrollView&gt;</td><td>Scrollable container</td></tr>
<tr><td>&lt;Card&gt;</td><td>Material card</td></tr>
<tr><td>&lt;Spacer&gt;</td><td>Fills space</td></tr>
<tr><td>&lt;Divider&gt;</td><td>Horizontal line</td></tr>
</table>
<h2>Text & Input</h2>
<table>
<tr><th>Element</th><th>Description</th></tr>
<tr><td>&lt;Text&gt;</td><td>Label</td></tr>
<tr><td>&lt;Button&gt;</td><td>Clickable button</td></tr>
<tr><td>&lt;Input&gt;</td><td>Single-line text input</td></tr>
<tr><td>&lt;TextField&gt;</td><td>Same as Input</td></tr>
</table>
<h2>Interactive</h2>
<table>
<tr><th>Element</th><th>Description</th></tr>
<tr><td>&lt;Switch&gt;</td><td>Toggle on/off</td></tr>
<tr><td>&lt;Slider&gt;</td><td>Range input (0..1)</td></tr>
<tr><td>&lt;Picker&gt;</td><td>Dropdown</td></tr>
<tr><td>&lt;Checkbox&gt;</td><td>Checkbox</td></tr>
<tr><td>&lt;RadioButton&gt;</td><td>Radio button</td></tr>
</table>
<h2>Media</h2>
<table>
<tr><th>Element</th><th>Description</th></tr>
<tr><td>&lt;Image&gt;</td><td>Image (URL or asset)</td></tr>
<tr><td>&lt;WebView&gt;</td><td>Embed web pages</td></tr>
<tr><td>&lt;Icon&gt;</td><td>Material icon</td></tr>
</table>
<h2>Loading & Navigation</h2>
<table>
<tr><th>Element</th><th>Description</th></tr>
<tr><td>&lt;ActivityIndicator&gt;</td><td>Circular spinner</td></tr>
<tr><td>&lt;Progress&gt;</td><td>Linear progress bar</td></tr>
<tr><td>&lt;TopBar&gt;</td><td>Top app bar</td></tr>
<tr><td>&lt;TabBar&gt;</td><td>Tab row</td></tr>
<tr><td>&lt;BottomNav&gt;</td><td>Bottom navigation</td></tr>
<tr><td>&lt;Fab&gt;</td><td>Floating action button</td></tr>
</table>'

generate "webview.html" "WebView" '<h1>WebView</h1>
<p>emo'\''s WebView has full feature parity with <a href="https://github.com/react-native-webview/react-native-webview">react-native-webview</a>.</p>
<h2>Basic usage</h2>
<pre><code>&lt;WebView source="https://expo.dev" /&gt;</code></pre>
<h2>Inline HTML</h2>
<pre><code>&lt;WebView html="&lt;h1&gt;Hello!&lt;/h1&gt;&lt;p&gt;Inline HTML&lt;/p&gt;" /&gt;</code></pre>
<h2>JavaScript injection</h2>
<pre><code>&lt;WebView
  source="https://example.com"
  injectedJavaScript="document.title"
  onMessage={(msg) =&gt; console.log(msg)"
/&gt;</code></pre>
<h2>JavaScript bridge</h2>
<p>From inside the WebView, call:</p>
<pre><code>window.EmoGo.postMessage("Hello from web page!");</code></pre>
<p>This triggers the <code>onMessage</code> handler in your .em file.</p>
<h2>All options</h2>
<table>
<tr><th>Option</th><th>Type</th><th>Description</th></tr>
<tr><td>source</td><td>string</td><td>URL to load</td></tr>
<tr><td>html</td><td>string</td><td>Inline HTML</td></tr>
<tr><td>injectedJavaScript</td><td>string</td><td>JS injected after load</td></tr>
<tr><td>userAgent</td><td>string</td><td>Custom User-Agent</td></tr>
<tr><td>javaScriptEnabled</td><td>bool</td><td>Enable JS (default: true)</td></tr>
<tr><td>domStorageEnabled</td><td>bool</td><td>DOM storage</td></tr>
<tr><td>cacheEnabled</td><td>bool</td><td>Cache (default: true)</td></tr>
<tr><td>scalesPageToFit</td><td>bool</td><td>Auto-fit page</td></tr>
<tr><td>textZoom</td><td>int</td><td>Text zoom %</td></tr>
<tr><td>method</td><td>string</td><td>HTTP method (GET/POST)</td></tr>
</table>
<h2>Event handlers</h2>
<table>
<tr><th>Handler</th><th>Callback</th><th>Fires when</th></tr>
<tr><td>onMessage</td><td>func(string)</td><td>window.EmoGo.postMessage()</td></tr>
<tr><td>onLoadStart</td><td>func(string)</td><td>Page starts loading</td></tr>
<tr><td>onLoadEnd</td><td>func(string)</td><td>Page finishes loading</td></tr>
<tr><td>onLoadProgress</td><td>func(float64)</td><td>Load progress (0-1)</td></tr>
<tr><td>onNavigationStateChange</td><td>func(map)</td><td>URL changes</td></tr>
<tr><td>onError</td><td>func(string)</td><td>Load error</td></tr>
<tr><td>onHttpError</td><td>func(map)</td><td>HTTP 4xx/5xx</td></tr>
</table>'

generate "navigation.html" "Navigation" '<h1>Navigation</h1>
<p>emo'\''s navigation API is inspired by <a href="https://github.com/expo/expo/tree/main/packages/expo-router">expo-router</a>.</p>
<h2>Programmatic navigation</h2>
<pre><code>import "github.com/emo-framework/emo/nav"

nav.Navigate("/profile")
nav.Navigate("/user/42")
nav.Back()
nav.Replace("/login")
nav.Reset("/")</code></pre>
<h2>Route params</h2>
<pre><code>// Register route
nav.RegisterRoute("user/[id]", "user/:id")

// Navigate
nav.Navigate("/user/42")

// Get param
id := nav.Param("id")  // → "42"</code></pre>
<h2>File-based routing</h2>
<table>
<tr><th>File</th><th>Route</th></tr>
<tr><td>app/index.em</td><td>/</td></tr>
<tr><td>app/profile.em</td><td>/profile</td></tr>
<tr><td>app/user/[id].em</td><td>/user/:id</td></tr>
<tr><td>app/(tabs)/home.em</td><td>/home (group ignored)</td></tr>
</table>
<h2>Navigation state</h2>
<pre><code>nav.Current()      // current Route
nav.CanGoBack()    // true if stack > 1
nav.Stack()        // full navigation stack
nav.OnNavigate(func(r Route) { ... })  // listener</code></pre>'

generate "live-reload.html" "Live Reload" '<h1>Live Reload</h1>
<p>emo'\''s live reload works in three tiers:</p>
<h2>1. State mutation (&lt; 100ms)</h2>
<p>When <code>setCount(...)</code> is called, emo re-renders and pushes a patch to the device.</p>
<h2>2. File save (~200-500ms)</h2>
<ol>
<li><code>fsnotify</code> detects the change</li>
<li>Dev server debounces for 150ms</li>
<li>Re-transpiles the .em file</li>
<li>Pushes new vtree to all connected devices</li>
</ol>
<h2>3. Hot function swap (roadmap)</h2>
<p>Editing helper functions outside the DSL requires restarting <code>emo start</code>.</p>
<h2>How it works</h2>
<pre><code>Save .em file
    ↓
fsnotify fires
    ↓
Dev server re-transpiles .em → vtree
    ↓
WebSocket push to device
    ↓
emo Go app applies vtree as Jetpack Compose
    ↓
UI updates (no app restart)</code></pre>'

generate "android-sync.html" "Android Sync" '<h1>Android Sync</h1>
<p>emo auto-generates native Kotlin and XML in your <code>android/</code> folder on every file save.</p>
<h2>How it works</h2>
<p>When you run <code>emo start</code>, the syncer generates:</p>
<pre><code>android/app/src/main/java/dev/emo/app/GeneratedComponents.kt  ← Kotlin @Composable
android/app/src/main/res/values/emo_styles.xml                ← CSS → XML</code></pre>
<h2>Example</h2>
<p>Edit <code>app/index.em</code>:</p>
<pre><code>component App {
  state count = 0
  render {
    &lt;Column spacing={16} padding={24}&gt;
      &lt;Text fontSize={28} fontWeight="bold"&gt;Counter&lt;/Text&gt;
      &lt;Button onClick={() =&gt; count = count + 1}&gt;+&lt;/Button&gt;
    &lt;/Column&gt;
  }
}</code></pre>
<p>Generated Kotlin:</p>
<pre><code>@Composable
fun EmoRoot() { App() }

@Composable
fun App() {
    var count by remember { mutableIntStateOf(0) }
    Column(modifier = Modifier.fillMaxWidth(), verticalArrangement = ...) {
        Text("Counter", fontWeight = FontWeight.Bold)
        Button(onClick = { count = count + 1 }) { Text("+") }
    }
}</code></pre>
<h2>Usage</h2>
<pre><code>emo start                       # sync-android is ON by default
emo start --sync-android=false  # disable
emo start --package com.example.myapp  # custom package</code></pre>'

generate "plugins.html" "Plugins" '<h1>Plugin System</h1>
<p>emo has built-in plugins for native device capabilities.</p>
<h2>Available plugins</h2>
<table>
<tr><th>Plugin</th><th>Methods</th></tr>
<tr><td>camera</td><td>takePhoto(quality), requestPermission()</td></tr>
<tr><td>location</td><td>getCurrentPosition(highAccuracy), startWatch()</td></tr>
<tr><td>storage</td><td>get(key), set(key, value), remove(key)</td></tr>
<tr><td>vibration</td><td>vibrate(ms)</td></tr>
<tr><td>haptics</td><td>impact(style), notification(type), selection()</td></tr>
<tr><td>linking</td><td>openURL(url), openSettings()</td></tr>
</table>
<h2>Using plugins</h2>
<pre><code>import "github.com/emo-framework/emo/plugin"

plugin.Invoke("camera", "takePhoto", map[string]any{"quality": 80},
    func(result any, err error) {
        if err != nil { return }
        setPhoto(result.(string))
    })</code></pre>
<h2>Authoring plugins</h2>
<pre><code>type FooPlugin struct{}
func (FooPlugin) Name() string { return "foo" }
func (FooPlugin) Methods() []plugin.Method { ... }
func init() { plugin.Register(FooPlugin{}) }</code></pre>'

generate "haptics.html" "Haptics" '<h1>Haptics</h1>
<p>Haptic feedback, inspired by <a href="https://github.com/expo/expo/tree/main/packages/expo-haptics">expo-haptics</a>.</p>
<h2>Impact</h2>
<pre><code>import "github.com/emo-framework/emo/haptics"

haptics.Impact(haptics.ImpactLight)
haptics.Impact(haptics.ImpactMedium)
haptics.Impact(haptics.ImpactHeavy)</code></pre>
<h2>Notification</h2>
<pre><code>haptics.Notification(haptics.Success)
haptics.Notification(haptics.Warning)
haptics.Notification(haptics.Error)</code></pre>
<h2>Selection</h2>
<pre><code>haptics.Selection()  // light click for list/picker</code></pre>
<h2>Custom vibration</h2>
<pre><code>haptics.Vibrate(200)  // 200ms</code></pre>'

generate "securestore.html" "SecureStore" '<h1>SecureStore</h1>
<p>Encrypted key-value storage, inspired by <a href="https://github.com/expo/expo/tree/main/packages/expo-secure-store">expo-secure-store</a>.</p>
<h2>Usage</h2>
<pre><code>import "github.com/emo-framework/emo/securestore"

// Initialize (call once on startup)
securestore.Init(projectDir)

// Set
securestore.Set("authToken", "eyJhbGc...")

// Get
token, ok := securestore.Get("authToken")

// Delete
securestore.Delete("authToken")

// Check
if securestore.Has("authToken") { ... }

// All keys
keys := securestore.Keys()

// Clear all
securestore.Clear()</code></pre>
<p>Values are stored in <code>.emo/securestore.json</code> with <code>0600</code> permissions (owner-only).</p>'

generate "linking.html" "Linking" '<h1>Linking</h1>
<p>Deep linking and URL handling, inspired by <a href="https://github.com/expo/expo/tree/main/packages/expo-linking">expo-linking</a>.</p>
<h2>Open external URLs</h2>
<pre><code>import "github.com/emo-framework/emo/linking"

linking.OpenURL("https://example.com")
linking.OpenURL("mailto:support@example.com")
linking.OpenSettings()</code></pre>
<h2>Create deep links</h2>
<pre><code>linking.SetScheme("myapp")
url := linking.CreateURL("/profile/42")
// → "myapp:///profile/42"</code></pre>
<h2>Parse URLs</h2>
<pre><code>p := linking.Parse("myapp://profile/42?ref=notif")
// p.Scheme = "myapp"
// p.Host = "profile"
// p.Path = "/42"
// p.Query = "ref=notif"</code></pre>
<h2>Listen for deep links</h2>
<pre><code>linking.AddListener(func(p linking.ParsedURL) {
    fmt.Println("Received:", p.Path)
})</code></pre>'

generate "build.html" "Building APKs" '<h1>Building APKs</h1>
<h2>Development (live reload)</h2>
<pre><code>emo start     # dev server
emo go        # launch on device</code></pre>
<h2>Production build</h2>
<pre><code>emo build</code></pre>
<p>This generates:</p>
<ul>
<li><code>__emo_gen__/emo_gen.go</code> — transpiled Go source</li>
<li><code>.emo/build/emo-bundle.json</code> — production bundle (embedded runtime)</li>
<li><code>.emo/build/EmoRoot.kt</code> — Kotlin Compose source</li>
</ul>
<h2>Assemble the APK</h2>
<ol>
<li>Copy <code>EmoRoot.kt</code> into an Android Studio project</li>
<li>Run <code>./gradlew assembleDebug</code></li>
<li>APK at <code>app/build/outputs/apk/debug/app-debug.apk</code></li>
</ol>
<h2>Production mode (embedded runtime)</h2>
<p>The <code>emo-bundle.json</code> is loaded by the embedded runtime at app startup. The app runs entirely on-device with no dev server connection needed.</p>'

generate "troubleshooting.html" "Troubleshooting" '<h1>Troubleshooting</h1>
<h2>adb not found on PATH</h2>
<pre><code># Debian/Ubuntu:
apt-get install -y android-tools-adb

# macOS:
brew install android-platform-tools</code></pre>
<h2>address already in use</h2>
<p>emo start automatically finds a free port if 7575 is busy. Check the log for the actual port.</p>
<h2>TLS certificate error</h2>
<pre><code># Install CA certificates:
apt-get install -y ca-certificates && update-ca-certificates

# Or as a workaround:
export EMO_INSECURE=1</code></pre>
<h2>emo: command not found</h2>
<pre><code>echo '\''export PATH="$HOME/.local/bin:$PATH"'\'' >> ~/.bashrc
source ~/.bashrc</code></pre>
<h2>No device connected</h2>
<pre><code>adb devices    # list devices
emulator -avd Pixel_7  # start emulator</code></pre>
<h2>WebView shows ERR_NAME_NOT_RESOLVED</h2>
<p>Make sure <code>source={url}</code> uses <code>{url}</code> (expression) not <code>source="url"</code> (literal string). Update to the latest emo version — this was a bug fixed in v0.2.1.</p>'

generate "cli.html" "CLI Reference" '<h1>CLI Reference</h1>
<h2>emo create [name]</h2>
<p>Interactive project creation (like create-expo-app).</p>
<pre><code>emo create my-app
emo create my-app --template counter</code></pre>
<h2>emo init [name]</h2>
<p>Create a project from a template.</p>
<pre><code>emo init myapp                          # uses "default" template
emo init myapp --template counter
emo init myapp --package com.example.myapp</code></pre>
<h2>emo start</h2>
<p>Start the dev server with live reload.</p>
<pre><code>emo start                    # default port 7575
emo start --port 8080        # specific port
emo start --launch           # auto-launch on device
emo start --sync-android=false  # disable native codegen</code></pre>
<h2>emo go</h2>
<p>Launch emo Go preview app on a connected device.</p>
<pre><code>emo go
emo go --apk /path/to/emo-go.apk</code></pre>
<h2>emo build</h2>
<p>Generate production bundle + Kotlin source.</p>
<pre><code>emo build
emo build --out app.apk --package dev.emo.myapp</code></pre>
<h2>emo templates</h2>
<p>List available project templates from GitHub.</p>
<h2>emo components</h2>
<p>List installable components.</p>
<h2>emo install &lt;name&gt;</h2>
<p>Install a component from the registry.</p>
<pre><code>emo install WebView
emo install Card</code></pre>
<h2>emo plugins</h2>
<p>List registered plugins.</p>'

echo "=== All guide pages generated ==="
ls -la /home/z/my-project/emo/docs/guides/
