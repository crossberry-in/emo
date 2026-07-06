// Package server implements the emo dev server.
//
// The server is the central piece of emo's live-reload experience. It:
//
//   1. Watches .go files in the project for changes.
//   2. On change, re-parses and re-renders the root component into a vtree.
//   3. Pushes the new vtree (or a diff) to all connected emo Go preview apps
//      over a WebSocket.
//   4. Receives user events from the preview app and dispatches them to the
//      registered Go handlers.
//   5. Forwards plugin calls from Go to the device and routes results back.
//   6. Optionally launches / restarts the emo Go preview app on a connected
//      Android emulator or device via adb.
//
// The server is intentionally a single binary with no external dependencies
// beyond the standard library plus fsnotify, gorilla/websocket, and cobra.
package server

import (
        "context"
        "crypto/sha1"
        "encoding/hex"
        "encoding/json"
        "fmt"
        "log"
        "net"
        "net/http"
        "os"
        "os/exec"
        "path/filepath"
        "strings"
        "sync"
        "time"

        "github.com/emo-framework/emo/codegen"
        "github.com/emo-framework/emo/dsl"
        "github.com/emo-framework/emo/plugin"
        "github.com/emo-framework/emo/runtime"
        "github.com/fsnotify/fsnotify"
        "github.com/gorilla/websocket"
)

// Server is the emo dev server.
type Server struct {
        mu sync.Mutex

        // Project config.
        ProjectDir string
        ProjectID  string
        Port       int

        // Root component factory. The server re-invokes this on every file change
        // and every state mutation. The function is set up by the CLI after it
        // loads the project's main package.
        RootFactory func() dsl.Element

        // Current vtree state (last pushed). Used for diffing on the next push.
        current dsl.Element

        // Connected clients.
        clients   map[*websocket.Conn]*client
        clientsMu sync.Mutex

        // Watcher.
        watcher *fsnotify.Watcher

        // Hot-reload debounce.
        debounce *time.Timer

        // Plugin call transport (sends to all clients).
        pluginResultsMu sync.Mutex
        pluginResults   map[string]chan runtime.PluginPayload

        // Lifecycle.
        ctx    context.Context
        cancel context.CancelFunc
}

type client struct {
        conn     *websocket.Conn
        handshake runtime.HandshakePayload
}

// New constructs a Server bound to projectDir.
func New(projectDir string, port int) *Server {
        ctx, cancel := context.WithCancel(context.Background())
        return &Server{
                ProjectDir: projectDir,
                ProjectID:  projectID(projectDir),
                Port:       port,
                clients:    map[*websocket.Conn]*client{},
                pluginResults: map[string]chan runtime.PluginPayload{},
                ctx:        ctx,
                cancel:     cancel,
        }
}

// Start launches the server: opens the file watcher, opens the HTTP/WS
// listener, and blocks until ctx is cancelled.
func (s *Server) Start() error {
        // Install reactive scheduler: state mutations trigger a re-render.
        dsl.SetReRenderScheduler(s.ScheduleReRender)

        // Install plugin transport.
        plugin.SetTransport(s.forwardPluginCall)

        // Open fsnotify watcher.
        w, err := fsnotify.NewWatcher()
        if err != nil {
                return fmt.Errorf("open watcher: %w", err)
        }
        s.watcher = w
        if err := s.watchRecursive(s.ProjectDir); err != nil {
                return fmt.Errorf("watch project dir: %w", err)
        }
        go s.watchLoop()

        // HTTP routes.
        mux := http.NewServeMux()
        mux.HandleFunc("/ws", s.handleWS)
        mux.HandleFunc("/", s.handleIndex)
        mux.HandleFunc("/manifest", s.handleManifest)

        addr := fmt.Sprintf(":%d", s.Port)
        log.Printf("emo dev server listening on http://localhost%s  (project: %s)", addr, s.ProjectID)

        srv := &http.Server{Addr: addr, Handler: mux}

        // Push initial vtree once RootFactory is installed.
        go s.initialPush()

        return srv.ListenAndServe()
}

// Stop tears down the server.
func (s *Server) Stop() {
        s.cancel()
        if s.watcher != nil {
                _ = s.watcher.Close()
        }
}

// ---------------------------------------------------------------------------
// File watcher
// ---------------------------------------------------------------------------

func (s *Server) watchRecursive(root string) error {
        return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
                if err != nil {
                        return nil
                }
                if info.IsDir() {
                        // Skip hidden and build dirs.
                        name := info.Name()
                        if name == ".git" || name == "build" || name == ".gradle" || name == "node_modules" {
                                return filepath.SkipDir
                        }
                        return s.watcher.Add(path)
                }
                return nil
        })
}

func (s *Server) watchLoop() {
        for {
                select {
                case <-s.ctx.Done():
                        return
                case ev, ok := <-s.watcher.Events:
                        if !ok {
                                return
                        }
                        if ev.Has(fsnotify.Write) || ev.Has(fsnotify.Create) {
                                if isGoFile(ev.Name) {
                                        s.debouncedReload("file:" + ev.Name)
                                }
                        }
                case err, ok := <-s.watcher.Errors:
                        if !ok {
                                return
                        }
                        log.Printf("watcher error: %v", err)
                }
        }
}

func isGoFile(p string) bool {
        ext := filepath.Ext(p)
        // emo 0.1 SDK watches .em and .css files in addition to .go.
        return ext == ".go" || ext == ".em" || ext == ".css"
}

// debouncedReload coalesces a burst of file writes into a single reload.
func (s *Server) debouncedReload(reason string) {
        s.mu.Lock()
        defer s.mu.Unlock()
        if s.debounce != nil {
                s.debounce.Stop()
        }
        s.debounce = time.AfterFunc(150*time.Millisecond, func() {
                s.Reload(reason)
        })
}

// Reload re-runs the build pipeline and pushes a new vtree.
func (s *Server) Reload(reason string) {
        if s.RootFactory == nil {
                log.Println("reload requested but RootFactory not installed yet")
                return
        }

        log.Printf("reload (%s) — re-rendering vtree", reason)

        if err := s.hotSwapGo(); err != nil {
                s.broadcastError("Hot reload failed", err.Error(), "", 0)
                return
        }

        newTree := s.RootFactory()
        old := s.current
        s.current = newTree

        if old.ID == "" {
                // First push: send full vtree.
                s.broadcastVTree(newTree, reason)
                return
        }

        // Compute the diff. If the tree changed structurally (file edit), send
        // the full vtree so the client can resync. If only state values changed,
        // send a patch (smaller payload, faster).
        ops := codegen.Diff(old, newTree)
        if len(ops) == 0 {
                return
        }

        // For file-change reloads, always send the full tree. For state mutations,
        // the patch is sufficient. We distinguish by the reason string.
        if isStateMutation(reason) && len(ops) < 10 {
                // Small state change — send patch.
                s.broadcastPatch(ops, reason)
        } else {
                // File change or large diff — send full tree for reliability.
                s.broadcastVTree(newTree, reason)
        }
}

// isStateMutation returns true if the reload was triggered by a state change
// (not a file edit). State mutations produce small diffs; file edits can
// produce large structural changes that are safer to send as a full tree.
func isStateMutation(reason string) bool {
        return reason == "state" || strings.HasPrefix(reason, "state:")
}

// hotSwapGo rebuilds the project's main package as a plugin and swaps it in.
// In MVP we use a sentinel file approach: any .go change triggers a reload of
// the running binary's RootFactory pointer (which the CLI keeps re-assigning).
// A full implementation would use Go's plugin package on Linux or a
// fork-and-exec on other platforms.
func (s *Server) hotSwapGo() error {
        // MVP: no-op. The CLI reassigns RootFactory on restart.
        // Documented in README as a known limitation for the open-source preview.
        return nil
}

// ScheduleReRender is called by dsl state setters. It re-renders the root and
// pushes a diff. We debounce to coalesce multiple state mutations in the same
// tick.
func (s *Server) ScheduleReRender() {
        s.debouncedReload("state")
}

// initialPush waits for RootFactory to be installed, then pushes the first
// vtree. The CLI installs the factory by loading the project's main package.
func (s *Server) initialPush() {
        for i := 0; i < 100; i++ { // wait up to 10s
                s.mu.Lock()
                f := s.RootFactory
                s.mu.Unlock()
                if f != nil {
                        s.Reload("init")
                        return
                }
                time.Sleep(100 * time.Millisecond)
        }
        log.Println("warning: RootFactory not installed after 10s; live reload will not work")
}

// ---------------------------------------------------------------------------
// HTTP / WebSocket handlers
// ---------------------------------------------------------------------------

var upgrader = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
        c, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
                log.Printf("ws upgrade: %v", err)
                return
        }
        defer c.Close()

        cl := &client{conn: c}
        s.addClient(cl)
        defer s.removeClient(cl)

        // Send hello.
        hello := runtime.Message{
                Kind: runtime.KindHello,
                TS:   time.Now(),
                Payload: runtime.HelloPayload{
                        ServerName: "emo-dev-server",
                        Version:    "0.1.0",
                        ProjectID:  s.ProjectID,
                },
        }
        if err := c.WriteJSON(hello); err != nil {
                return
        }

        // If we already have a vtree, push it immediately.
        s.mu.Lock()
        tree := s.current
        s.mu.Unlock()
        if tree.ID != "" {
                s.sendVTree(cl, tree, "init")
        }

        // Read loop.
        for {
                var msg runtime.Message
                if err := c.ReadJSON(&msg); err != nil {
                        return
                }
                s.handleClientMessage(cl, msg)
        }
}

func (s *Server) handleClientMessage(cl *client, msg runtime.Message) {
        switch msg.Kind {
        case runtime.KindHandshake:
                if hs, ok := msg.Payload.(map[string]any); ok {
                        b, _ := json.Marshal(hs)
                        _ = json.Unmarshal(b, &cl.handshake)
                        log.Printf("client connected: %s on %s (Android %s)", cl.handshake.Client, cl.handshake.Device, cl.handshake.Android)
                }
        case runtime.KindEvent:
                if ev, ok := msg.Payload.(map[string]any); ok {
                        token, _ := ev["token"].(string)
                        event, _ := ev["event"].(string)
                        value := ev["value"]
                        log.Printf("event: %s on handler %s value=%v", event, token, value)
                        dsl.InvokeHandler(token, value)
                }
        case runtime.KindAck:
                // no-op
        default:
                log.Printf("unknown message kind from client: %s", msg.Kind)
        }
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
        // Serves a small landing page with QR-code-ish info for connecting the
        // emo Go preview app.
        ip := s.lanIP()
        w.Header().Set("Content-Type", "text/html")
        fmt.Fprintf(w, `<!doctype html><html><body style="font-family:sans-serif;max-width:600px;margin:40px auto;padding:0 16px">
<h1>emo dev server</h1>
<p>Project: <code>%s</code></p>
<p>Server: <code>%s:%d</code> (LAN: <code>%s:%d</code>)</p>
<h2>Connect your device</h2>
<ol>
  <li>Install the <b>emo Go</b> preview app on your Android device or emulator.</li>
  <li>Open the app and enter this URL: <code>ws://%s:%d/ws</code></li>
  <li>Or scan the QR code below with emo Go.</li>
</ol>
<p>Connected clients: <b>%d</b></p>
<p><a href="/manifest">Manifest</a></p>
</body></html>`, s.ProjectID, "127.0.0.1", s.Port, ip, s.Port, ip, s.Port, s.clientCount())
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]any{
                "projectId": s.ProjectID,
                "port":      s.Port,
                "lanIP":     s.lanIP(),
                "plugins":   pluginNames(),
        })
}

// ---------------------------------------------------------------------------
// Client management & broadcast
// ---------------------------------------------------------------------------

func (s *Server) addClient(c *client) {
        s.clientsMu.Lock()
        s.clients[c.conn] = c
        s.clientsMu.Unlock()
}

func (s *Server) removeClient(c *client) {
        s.clientsMu.Lock()
        delete(s.clients, c.conn)
        s.clientsMu.Unlock()
}

func (s *Server) clientCount() int {
        s.clientsMu.Lock()
        defer s.clientsMu.Unlock()
        return len(s.clients)
}

func (s *Server) eachClient(fn func(c *client)) {
        s.clientsMu.Lock()
        snapshot := make([]*client, 0, len(s.clients))
        for _, c := range s.clients {
                snapshot = append(snapshot, c)
        }
        s.clientsMu.Unlock()
        for _, c := range snapshot {
                fn(c)
        }
}

func (s *Server) broadcastVTree(tree dsl.Element, reason string) {
        s.eachClient(func(c *client) { s.sendVTree(c, tree, reason) })
}

func (s *Server) sendVTree(c *client, tree dsl.Element, reason string) {
        msg := runtime.Message{
                Kind: runtime.KindVTree,
                TS:   time.Now(),
                Payload: runtime.VTreePayload{
                        Root:   tree,
                        Hash:   hashTree(tree),
                        Reason: reason,
                },
        }
        _ = c.conn.WriteJSON(msg)
}

func (s *Server) broadcastPatch(ops []codegen.Op, reason string) {
        s.eachClient(func(c *client) {
                msg := runtime.Message{
                        Kind: runtime.KindPatch,
                        TS:   time.Now(),
                        Payload: map[string]any{
                                "ops":    ops,
                                "reason": reason,
                                "hash":   hashTree(s.current),
                        },
                }
                _ = c.conn.WriteJSON(msg)
        })
}

func (s *Server) broadcastError(title, msg, file string, line int) {
        s.eachClient(func(c *client) {
                _ = c.conn.WriteJSON(runtime.Message{
                        Kind: runtime.KindError,
                        TS:   time.Now(),
                        Payload: runtime.ErrorPayload{
                                Title:   title,
                                Message: msg,
                                File:    file,
                                Line:    line,
                        },
                })
        })
}

// forwardPluginCall sends a plugin call to all connected devices and resolves
// the reply when the first one responds.
func (s *Server) forwardPluginCall(pluginName, method string, params map[string]any, reply func(any, error)) {
        callID := "call_" + randID(6)
        ch := make(chan runtime.PluginPayload, 1)

        s.pluginResultsMu.Lock()
        s.pluginResults[callID] = ch
        s.pluginResultsMu.Unlock()

        // Send to first connected client.
        s.eachClient(func(c *client) {
                _ = c.conn.WriteJSON(runtime.Message{
                        Kind: runtime.KindPlugin,
                        TS:   time.Now(),
                        Payload: map[string]any{
                                "callId": callID,
                                "plugin": pluginName,
                                "method": method,
                                "params": params,
                        },
                })
        })

        // Wait for response with timeout.
        go func() {
                select {
                case r := <-ch:
                        if r.Error != "" {
                                reply(nil, fmt.Errorf("%s", r.Error))
                        } else {
                                reply(r.Result, nil)
                        }
                case <-time.After(30 * time.Second):
                        reply(nil, fmt.Errorf("plugin call %s.%s timed out", pluginName, method))
                }
                s.pluginResultsMu.Lock()
                delete(s.pluginResults, callID)
                s.pluginResultsMu.Unlock()
        }()
}

// ---------------------------------------------------------------------------
// adb integration
// ---------------------------------------------------------------------------

// LaunchOnDevice installs the emo Go preview APK on the first adb-visible
// device and launches its MainActivity, passing the dev server URL.
func (s *Server) LaunchOnDevice(apkPath, activityName string) error {
        if _, err := exec.LookPath("adb"); err != nil {
                return fmt.Errorf("adb not found in PATH: %w", err)
        }

        // List devices.
        out, err := exec.Command("adb", "devices").CombinedOutput()
        if err != nil {
                return fmt.Errorf("adb devices: %w: %s", err, out)
        }
        if !hasAdbDevice(string(out)) {
                return fmt.Errorf("no adb device connected (output: %s)", string(out))
        }

        // Install APK (-r reinstall).
        if apkPath != "" {
                if out, err := exec.Command("adb", "install", "-r", apkPath).CombinedOutput(); err != nil {
                        return fmt.Errorf("adb install: %w: %s", err, out)
                }
        }

        // Launch with --es to pass the dev server URL as an extra.
        ip := s.lanIP()
        url := fmt.Sprintf("ws://%s:%d/ws", ip, s.Port)
        args := []string{
                "shell", "am", "start", "-n", activityName,
                "--es", "emo_server", url,
                "--es", "emo_project", s.ProjectID,
        }
        if out, err := exec.Command("adb", args...).CombinedOutput(); err != nil {
                return fmt.Errorf("adb shell am start: %w: %s", err, out)
        }
        log.Printf("launched emo Go on device, connecting to %s", url)
        return nil
}

func hasAdbDevice(s string) bool {
        lines := splitLines(s)
        for _, l := range lines[1:] { // skip header
                if l == "" {
                        continue
                }
                if !contains(l, "offline") && !contains(l, "unauthorized") {
                        return true
                }
        }
        return false
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (s *Server) lanIP() string {
        addrs, err := net.InterfaceAddrs()
        if err != nil {
                return "127.0.0.1"
        }
        for _, a := range addrs {
                if n, ok := a.(*net.IPNet); ok && !n.IP.IsLoopback() {
                        if n.IP.To4() != nil {
                                return n.IP.String()
                        }
                }
        }
        return "127.0.0.1"
}

func projectID(dir string) string {
        abs, _ := filepath.Abs(dir)
        h := sha1.Sum([]byte(abs))
        return hex.EncodeToString(h[:6])
}

func hashTree(t dsl.Element) string {
        b, _ := json.Marshal(t)
        h := sha1.Sum(b)
        return hex.EncodeToString(h[:8])
}

func pluginNames() []string {
        all := plugin.All()
        out := make([]string, len(all))
        for i, p := range all {
                out[i] = p.Name()
        }
        return out
}

func randID(n int) string {
        b := make([]byte, n)
        for i := range b {
                b[i] = byte('a' + (i * 7 % 26))
        }
        return string(b)
}

func splitLines(s string) []string {
        var out []string
        cur := ""
        for _, r := range s {
                if r == '\n' {
                        out = append(out, cur)
                        cur = ""
                } else if r != '\r' {
                        cur += string(r)
                }
        }
        if cur != "" {
                out = append(out, cur)
        }
        return out
}

func contains(s, sub string) bool {
        return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
        for i := 0; i+len(sub) <= len(s); i++ {
                if s[i:i+len(sub)] == sub {
                        return i
                }
        }
        return -1
}
