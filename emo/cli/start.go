package cli

import (
        "fmt"
        "log"
        "net"
        "os"
        "os/exec"
        "os/signal"
        "path/filepath"
        "syscall"

        "github.com/emo-framework/emo/dsl"
        "github.com/emo-framework/emo/eml"
        "github.com/emo-framework/emo/server"
        "github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
        var port int
        var watch string
        var launchEmoGo bool
        var emoGoApk string
        c := &cobra.Command{
                Use:   "start",
                Short: "Start the emo dev server with live reload",
                RunE: func(cmd *cobra.Command, args []string) error {
                        dir, _ := os.Getwd()
                        if watch != "" {
                                dir = filepath.Join(dir, watch)
                        }

                        // Load and transpile App.em
                        root, err := loadRootFromEM(dir)
                        if err != nil {
                                log.Printf("warning: could not load .em file: %v", err)
                                log.Printf("         falling back to built-in demo")
                                root = builtinDemoRootEM
                        }

                        if port == 0 {
                                port, err = freePort()
                                if err != nil {
                                        return err
                                }
                        } else {
                                // Check if the requested port is available; if not, auto-pick.
                                if !isPortFree(port) {
                                        oldPort := port
                                        port, err = freePort()
                                        if err != nil {
                                                return fmt.Errorf("port %d is in use and could not find a free port: %w", oldPort, err)
                                        }
                                        log.Printf("warning: port %d is in use, using port %d instead", oldPort, port)
                                }
                        }

                        srv := server.New(dir, port)
                        srv.RootFactory = root

                        if launchEmoGo {
                                go func() {
                                        if err := srv.LaunchOnDevice(emoGoApk, "dev.emo.go/.MainActivity"); err != nil {
                                                log.Printf("warning: could not launch emo Go on device: %v", err)
                                                log.Printf("         install the emo Go preview app and connect manually")
                                        }
                                }()
                        }

                        go func() {
                                sig := make(chan os.Signal, 1)
                                signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
                                <-sig
                                fmt.Println("\nemo dev server stopping…")
                                srv.Stop()
                                os.Exit(0)
                        }()

                        return srv.Start()
                },
        }
        c.Flags().IntVar(&port, "port", 7575, "Dev server port (0 = auto-pick)")
        c.Flags().StringVar(&watch, "watch", ".", "Directory to watch for live reload")
        c.Flags().BoolVar(&launchEmoGo, "launch", false, "Auto-install & launch emo Go preview app on a connected device")
        c.Flags().StringVar(&emoGoApk, "apk", "", "Path to emo Go preview APK to install before launching")
        return c
}

// loadRootFromEM finds App.em in the project directory, transpiles it to Go
// DSL code, writes the generated Go to __emo_gen__/emo_gen.go, compiles it,
// and returns a factory function that invokes the transpiled App() component.
//
// In this MVP, we transpile the .em file on every file change and re-evaluate
// the component tree. The transpiled Go source is written to
// __emo_gen__/emo_gen.go and compiled in-process via the Go runtime's
// plugin mechanism on Linux, or via fork-exec on other platforms.
//
// For the open-source preview, we use a simpler approach: the transpiler
// output is loaded as a Go source file, and the RootFactory directly
// evaluates the parsed .em AST's render tree, converting JSX elements to
// dsl.Element calls at runtime. This avoids the need for Go's plugin
// package and works on all platforms.
func loadRootFromEM(dir string) (func() dsl.Element, error) {
        // Look for app/index.em first (like Expo Router), then App.em, then any .em file.
        candidates := []string{
                filepath.Join(dir, "app", "index.em"),
                filepath.Join(dir, "App.em"),
        }
        for _, p := range candidates {
                if _, err := os.Stat(p); err == nil {
                        root, err := emlRootFactory(p)
                        if err != nil {
                                return nil, err
                        }
                        return root, nil
                }
        }

        // Find the first .em file in the directory.
        entries, err := os.ReadDir(dir)
        if err != nil {
                return nil, fmt.Errorf("read dir: %w", err)
        }
        for _, e := range entries {
                if filepath.Ext(e.Name()) == ".em" {
                        root, err := emlRootFactory(filepath.Join(dir, e.Name()))
                        if err != nil {
                                return nil, err
                        }
                        return root, nil
                }
        }
        return nil, fmt.Errorf("no .em file found in %s (looked for app/index.em, App.em, *.em)", dir)
}

// emlRootFactory creates a factory function that re-transpiles the .em file
// on each invocation, so file changes are picked up live.
func emlRootFactory(path string) (func() dsl.Element, error) {
        // Do an initial transpile to verify the file is valid.
        f, err := eml.TranspileFile(path)
        if err != nil {
                return nil, err
        }
        log.Printf("loaded %s — %d component(s), CSS: %v", path, len(f.Components), f.CSS != nil)

        return func() dsl.Element {
                // Re-transpile on each render to pick up file changes.
                f, err := eml.TranspileFile(path)
                if err != nil {
                        log.Printf("transpile error: %v", err)
                        return dsl.Text("Error: " + err.Error())
                }
                if len(f.Components) == 0 {
                        return dsl.Text("No components in .em file")
                }
                return evalComponent(f, f.Components[0])
        }, nil
}

// evalComponent evaluates a parsed .em Component AST into a dsl.Element at
// runtime, without going through Go code generation. This is the key
// innovation that makes live reload work without recompiling Go code.
//
// It tracks state variables declared in the component and resolves state
// references in expressions (e.g. {url} → "https://expo.dev").
func evalComponent(f *eml.File, c eml.Component) dsl.Element {
        if c.Render == nil {
                return dsl.View()
        }
        // Build a state map from the component's state declarations.
        stateValues := make(map[string]any)
        for _, s := range c.States {
                stateValues[s.Name] = parseStateValue(s.Default.Raw)
        }
        return evalJSXElementState(f, *c.Render, stateValues)
}

// parseStateValue parses a raw state default expression into a Go value.
func parseStateValue(raw string) any {
        raw = trimSpaceEM(raw)
        // String literal
        if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
                return raw[1 : len(raw)-1]
        }
        // Boolean
        if raw == "true" {
                return true
        }
        if raw == "false" {
                return false
        }
        // Integer
        if n, err := parseIntEM(raw); err == nil {
                return n
        }
        // Float
        if f, err := parseFloatEM(raw); err == nil {
                return f
        }
        // Fallback: treat as string
        return raw
}

func parseIntEM(s string) (int, error) {
        if s == "" {
                return 0, fmt.Errorf("empty")
        }
        n := 0
        for _, c := range s {
                if c < '0' || c > '9' {
                        return 0, fmt.Errorf("not int")
                }
                n = n*10 + int(c-'0')
        }
        return n, nil
}

func parseFloatEM(s string) (float64, error) {
        if s == "" {
                return 0, fmt.Errorf("empty")
        }
        var f float64
        var div float64 = 1
        hasDot := false
        for _, c := range s {
                if c == '.' {
                        hasDot = true
                        continue
                }
                if c < '0' || c > '9' {
                        return 0, fmt.Errorf("not float")
                }
                if hasDot {
                        div *= 10
                        f = f*10 + float64(c-'0')
                } else {
                        f = f*10 + float64(c-'0')
                }
        }
        return f / div, nil
}

// evalJSXElementState converts a JSXElement to a dsl.Element using the
// provided state map for expression resolution.
func evalJSXElementState(f *eml.File, el eml.JSXElement, state map[string]any) dsl.Element {
        // Apply className from CSS first.
        var opts []dsl.Option
        for _, a := range el.Attrs {
                if a.Name == "className" && a.Value.Kind == "string" {
                        opts = append(opts, cssPropsToOptions(f, a.Value.String)...)
                }
        }
        // Apply explicit attrs, resolving state references in expressions.
        for _, a := range el.Attrs {
                if a.Name == "className" {
                        continue
                }
                opt := jsxAttrToOptionState(a, state)
                if opt != nil {
                        opts = append(opts, opt)
                }
        }
        // Build children, resolving state references in expressions.
        var children []dsl.Element
        for _, c := range el.Children {
                switch c.Kind {
                case "text":
                        children = append(children, dsl.Text(c.Text))
                case "expr":
                        val := resolveExprEM(c.Expr, state)
                        children = append(children, dsl.Text(fmt.Sprintf("%v", val)))
                case "element":
                        children = append(children, evalJSXElementState(f, *c.Element, state))
                }
        }

        // For Text and Button, the first arg is the text content.
        textContent := buildTextContentState(el.Children, state)

        switch el.Tag {
        case "Scaffold":
                return dsl.Scaffold(append(opts, dsl.Children(children...))...)
        case "Column":
                return dsl.Column(append(opts, dsl.Children(children...))...)
        case "Row":
                return dsl.Row(append(opts, dsl.Children(children...))...)
        case "View":
                return dsl.View(append(opts, dsl.Children(children...))...)
        case "Text":
                return dsl.Text(textContent, opts...)
        case "Button":
                return dsl.Button(textContent, opts...)
        case "TextField":
                placeholder := ""
                for _, a := range el.Attrs {
                        if a.Name == "placeholder" {
                                placeholder = a.Value.String
                        }
                }
                return dsl.TextField(placeholder, opts...)
        case "Image":
                src := ""
                for _, a := range el.Attrs {
                        if a.Name == "source" {
                                src = a.Value.String
                        }
                }
                return dsl.Image(src, opts...)
        case "Spacer":
                return dsl.Spacer(opts...)
        case "Divider":
                return dsl.Divider(opts...)
        // --- New native UI elements (emo 0.1.2) ---
        case "WebView":
                return dsl.WebView(opts...)
        case "Input":
                return dsl.Input(opts...)
        case "SafeAreaView":
                return dsl.SafeAreaView(append(opts, dsl.Children(children...))...)
        case "ScrollView":
                return dsl.ScrollView(append(opts, dsl.Children(children...))...)
        case "Switch":
                return dsl.Switch(opts...)
        case "Slider":
                return dsl.Slider(opts...)
        case "ActivityIndicator":
                return dsl.ActivityIndicator(opts...)
        case "Picker":
                return dsl.Picker(opts...)
        case "List":
                return dsl.List(append(opts, dsl.Children(children...))...)
        case "Card":
                return dsl.Card(append(opts, dsl.Children(children...))...)
        case "Checkbox":
                return dsl.Checkbox(opts...)
        case "RadioButton":
                return dsl.RadioButton(opts...)
        case "Icon":
                return dsl.Icon(opts...)
        case "Fab":
                return dsl.Fab(opts...)
        case "Progress":
                return dsl.Progress(opts...)
        case "TabBar":
                return dsl.TabBar(append(opts, dsl.Children(children...))...)
        case "BottomNav":
                return dsl.BottomNav(append(opts, dsl.Children(children...))...)
        case "TopBar":
                return dsl.TopBar(opts...)
        default:
                // Unknown tag — render as a View with children.
                return dsl.View(append(opts, dsl.Children(children...))...)
        }
}

// buildTextContent concatenates text and expression children into a single
// string for Text/Button elements.
func buildTextContent(children []eml.JSXChild) string {
        var sb []byte
        for _, c := range children {
                switch c.Kind {
                case "text":
                        sb = append(sb, c.Text...)
                case "expr":
                        sb = append(sb, "{"...)
                        sb = append(sb, c.Expr...)
                        sb = append(sb, "}"...)
                }
        }
        return string(sb)
}

// jsxAttrToOption converts a JSX attribute to a dsl.Option.
func jsxAttrToOption(a eml.JSXAttr) dsl.Option {
        switch a.Name {
        case "onClick":
                expr := a.Value.Expr
                body := rewriteStateAssignEM(expr)
                return dsl.OnClick(func() {
                        // In the MVP runtime, state assignment rewrites are handled
                        // syntactically. For actual execution we'd need a Go interpreter.
                        // For now, we log the intended action.
                        log.Printf("onClick: %s", body)
                })
        case "spacing":
                return dsl.Spacing(parseNumEM(a.Value))
        case "padding":
                return dsl.Padding(parseNumEM(a.Value))
        case "background":
                return dsl.Bg(a.Value.String)
        case "color":
                return dsl.Fg(a.Value.String)
        case "fontSize":
                return dsl.Font(parseNumEM(a.Value), "normal")
        case "fontWeight":
                return dsl.Prop("fontWeight", a.Value.String)
        default:
                return dsl.Prop(a.Name, formatAttrValEM(a.Value))
        }
}

// jsxAttrToOptionState converts a JSX attribute to a dsl.Option, resolving
// state references in expressions using the provided state map.
func jsxAttrToOptionState(a eml.JSXAttr, state map[string]any) dsl.Option {
        switch a.Name {
        case "onClick":
                expr := a.Value.Expr
                // Parse the state assignment and create a handler that mutates state
                // and triggers re-render.
                targetState, newValue := parseStateAssign(expr, state)
                return dsl.OnClick(func() {
                        if targetState != "" {
                                state[targetState] = newValue
                                dsl.ScheduleReRender()
                        }
                })
        case "onChange":
                return dsl.OnChange(func(s string) {
                        // For now, just log — full onChange state wiring needs more work
                        log.Printf("onChange: %s", s)
                })
        case "spacing":
                return dsl.Spacing(parseNumEMState(a.Value, state))
        case "padding":
                return dsl.Padding(parseNumEMState(a.Value, state))
        case "background":
                return dsl.Bg(resolveStringEM(a.Value, state))
        case "color":
                return dsl.Fg(resolveStringEM(a.Value, state))
        case "fontSize":
                return dsl.Font(parseNumEMState(a.Value, state), "normal")
        case "fontWeight":
                return dsl.Prop("fontWeight", a.Value.String)
        case "source":
                // For WebView/Image — resolve the URL from state.
                return dsl.Source(resolveStringEM(a.Value, state))
        case "title":
                return dsl.Prop("title", resolveStringEM(a.Value, state))
        case "value":
                return dsl.Value(resolveExprEM(a.Value.Expr, state))
        default:
                return dsl.Prop(a.Name, resolveAttrValEM(a.Value, state))
        }
}

// parseStateAssign parses an onClick expression like "count = count + 1"
// and returns the target state name and the new value.
func parseStateAssign(expr string, state map[string]any) (string, any) {
        expr = trimSpaceEM(expr)
        // Strip arrow function: () => X
        if i := indexStr(expr, "=>"); i >= 0 {
                lhs := trimSpaceEM(expr[:i])
                if lhs == "()" || lhs == "(_)" {
                        return parseStateAssign(trimSpaceEM(expr[i+2:]), state)
                }
        }
        // Find assignment
        eq := indexByte(expr, '=')
        if eq < 0 {
                return "", nil
        }
        target := trimSpaceEM(expr[:eq])
        rhs := trimSpaceEM(expr[eq+1:])
        // Skip ==, <=, >=
        if len(rhs) > 0 && rhs[0] == '=' {
                return "", nil
        }
        // Evaluate the RHS expression
        val := evalExprEM(rhs, state)
        return target, val
}

// evalExprEM evaluates a simple expression against the state map.
// Supports: "count + 1", "count - 1", "0", "\"text\"", "true", state refs.
func evalExprEM(expr string, state map[string]any) any {
        expr = trimSpaceEM(expr)
        // String literal
        if len(expr) >= 2 && expr[0] == '"' && expr[len(expr)-1] == '"' {
                return expr[1 : len(expr)-1]
        }
        // Boolean
        if expr == "true" {
                return true
        }
        if expr == "false" {
                return false
        }
        // Addition/subtraction: "state + N" or "state - N"
        for _, op := range []string{"+", "-"} {
                if idx := indexStr(expr, " "+op+" "); idx > 0 {
                        left := trimSpaceEM(expr[:idx])
                        right := trimSpaceEM(expr[idx+3:])
                        leftVal := resolveValueEM(left, state)
                        rightVal := resolveValueEM(right, state)
                        return arithmeticEM(leftVal, rightVal, op)
                }
        }
        // Single value reference
        return resolveValueEM(expr, state)
}

// resolveValueEM resolves a single value (state name, number, or literal).
func resolveValueEM(ref string, state map[string]any) any {
        ref = trimSpaceEM(ref)
        // State reference
        if v, ok := state[ref]; ok {
                return v
        }
        // Integer
        if n, err := parseIntEM(ref); err == nil {
                return n
        }
        // Float
        if f, err := parseFloatEM(ref); err == nil {
                return f
        }
        // String literal
        if len(ref) >= 2 && ref[0] == '"' && ref[len(ref)-1] == '"' {
                return ref[1 : len(ref)-1]
        }
        return ref
}

// resolveExprEM resolves a state reference expression to its value.
func resolveExprEM(expr string, state map[string]any) any {
        return evalExprEM(expr, state)
}

// resolveStringEM resolves a JSXAttrValue to a string, handling state refs.
func resolveStringEM(v eml.JSXAttrValue, state map[string]any) string {
        switch v.Kind {
        case "string":
                return v.String
        case "expr":
                val := resolveExprEM(v.Expr, state)
                return fmt.Sprintf("%v", val)
        case "number":
                return v.Number
        }
        return ""
}

// resolveAttrValEM resolves a JSXAttrValue to an any for dsl.Prop.
func resolveAttrValEM(v eml.JSXAttrValue, state map[string]any) any {
        switch v.Kind {
        case "string":
                return v.String
        case "number":
                var f float64
                fmt.Sscanf(v.Number, "%f", &f)
                return f
        case "expr":
                return resolveExprEM(v.Expr, state)
        }
        return nil
}

// parseNumEMState parses a number from a JSXAttrValue, resolving state refs.
func parseNumEMState(v eml.JSXAttrValue, state map[string]any) float64 {
        if v.Kind == "number" {
                var f float64
                fmt.Sscanf(v.Number, "%f", &f)
                return f
        }
        if v.Kind == "expr" {
                val := resolveExprEM(v.Expr, state)
                if f, ok := val.(float64); ok {
                        return f
                }
                if n, ok := val.(int); ok {
                        return float64(n)
                }
        }
        return 0
}

// buildTextContentState builds text content from children, resolving state refs.
func buildTextContentState(children []eml.JSXChild, state map[string]any) string {
        var result string
        for _, c := range children {
                switch c.Kind {
                case "text":
                        result += c.Text
                case "expr":
                        val := resolveExprEM(c.Expr, state)
                        result += fmt.Sprintf("%v", val)
                }
        }
        return result
}

// arithmeticEM performs arithmetic on two values.
func arithmeticEM(a, b any, op string) any {
        ai, aOK := toIntEM(a)
        bi, bOK := toIntEM(b)
        if aOK && bOK {
                switch op {
                case "+":
                        return ai + bi
                case "-":
                        return ai - bi
                }
        }
        af, _ := toFloatEM(a)
        bf, _ := toFloatEM(b)
        switch op {
        case "+":
                return af + bf
        case "-":
                return af - bf
        }
        return 0
}

func toIntEM(v any) (int, bool) {
        switch x := v.(type) {
        case int:
                return x, true
        case float64:
                return int(x), true
        }
        return 0, false
}

func toFloatEM(v any) (float64, bool) {
        switch x := v.(type) {
        case int:
                return float64(x), true
        case float64:
                return x, true
        }
        return 0, false
}

// cssPropsToOptions converts CSS class properties to dsl.Options.
func cssPropsToOptions(f *eml.File, className string) []dsl.Option {
        if f.CSS == nil {
                return nil
        }
        props := f.CSS.LookupClass(className)
        if props == nil {
                return nil
        }
        var opts []dsl.Option
        for k, v := range props {
                switch k {
                case "background":
                        opts = append(opts, dsl.Bg(v))
                case "color":
                        opts = append(opts, dsl.Fg(v))
                case "padding":
                        opts = append(opts, dsl.Padding(parseCSSDimEM(v)))
                case "spacing":
                        opts = append(opts, dsl.Spacing(parseCSSDimEM(v)))
                case "font-size":
                        opts = append(opts, dsl.Font(parseCSSDimEM(v), "normal"))
                case "font-weight":
                        opts = append(opts, dsl.Prop("fontWeight", v))
                case "width":
                        opts = append(opts, dsl.Width(parseCSSDimEM(v)))
                case "height":
                        opts = append(opts, dsl.Height(parseCSSDimEM(v)))
                }
        }
        return opts
}

func parseNumEM(v eml.JSXAttrValue) float64 {
        if v.Kind == "number" {
                var f float64
                fmt.Sscanf(v.Number, "%f", &f)
                return f
        }
        if v.Kind == "expr" {
                var f float64
                fmt.Sscanf(v.Expr, "%f", &f)
                return f
        }
        return 0
}

func parseCSSDimEM(v string) float64 {
        var f float64
        s := v
        for len(s) > 0 && (s[len(s)-1] == 'p' || s[len(s)-1] == 'x' || s[len(s)-1] == 'd' || s[len(s)-1] == 's') {
                s = s[:len(s)-1]
        }
        fmt.Sscanf(s, "%f", &f)
        return f
}

func formatAttrValEM(v eml.JSXAttrValue) any {
        switch v.Kind {
        case "string":
                return v.String
        case "number":
                var f float64
                fmt.Sscanf(v.Number, "%f", &f)
                return f
        case "expr":
                return v.Expr
        }
        return nil
}

// rewriteStateAssignEM is a simple rewrite for state assignment expressions.
func rewriteStateAssignEM(expr string) string {
        expr = trimSpaceEM(expr)
        if i := indexStr(expr, "=>"); i >= 0 {
                lhs := trimSpaceEM(expr[:i])
                if lhs == "()" || lhs == "(_)" {
                        return rewriteStateAssignEM(trimSpaceEM(expr[i+2:]))
                }
        }
        eq := indexByte(expr, '=')
        if eq < 0 {
                return expr
        }
        lhs := trimSpaceEM(expr[:eq])
        rhs := trimSpaceEM(expr[eq+1:])
        if len(rhs) > 0 && rhs[0] == '=' {
                return expr
        }
        return "set" + capitalizeEM(lhs) + "(" + rhs + ")"
}

func capitalizeEM(s string) string {
        if s == "" {
                return s
        }
        if s[0] >= 'a' && s[0] <= 'z' {
                return string(s[0]-32) + s[1:]
        }
        return s
}

func trimSpaceEM(s string) string {
        start := 0
        for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
                start++
        }
        end := len(s)
        for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
                end--
        }
        return s[start:end]
}

func indexByte(s string, c byte) int {
        for i := 0; i < len(s); i++ {
                if s[i] == c {
                        return i
                }
        }
        return -1
}

func indexStr(s, sub string) int {
        for i := 0; i+len(sub) <= len(s); i++ {
                if s[i:i+len(sub)] == sub {
                        return i
                }
        }
        return -1
}

// builtinDemoRootEM is the fallback when no .em file is found.
func builtinDemoRootEM() dsl.Element {
        count, setCount := dsl.UseStateInt(0)
        return dsl.Scaffold(
                dsl.Prop("title", "emo"),
                dsl.Children(
                        dsl.Column(
                                dsl.Spacing(16),
                                dsl.Padding(24),
                                dsl.Children(
                                        dsl.Text("emo 0.1 SDK", dsl.Font(28, "bold")),
                                        dsl.Text(fmt.Sprintf("Count: %d", count), dsl.Font(18, "normal")),
                                        dsl.Button("Increment", dsl.OnClick(func() {
                                                setCount(count + 1)
                                        })),
                                        dsl.Button("Reset", dsl.OnClick(func() {
                                                setCount(0)
                                        })),
                                        dsl.Divider(),
                                        dsl.Text("Create App.em to get started!", dsl.Fg("#666666")),
                                ),
                        ),
                ),
        )
}

func freePort() (int, error) {
        l, err := net.Listen("tcp", ":0")
        if err != nil {
                return 0, err
        }
        defer l.Close()
        return l.Addr().(*net.TCPAddr).Port, nil
}

// isPortFree returns true if the given TCP port is available for binding.
func isPortFree(port int) bool {
        l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
        if err != nil {
                return false
        }
        l.Close()
        return true
}

func ensureGo() error {
        if _, err := exec.LookPath("go"); err != nil {
                return fmt.Errorf("go not found in PATH: %w", err)
        }
        return nil
}
