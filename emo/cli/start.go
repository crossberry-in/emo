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
func evalComponent(f *eml.File, c eml.Component) dsl.Element {
        if c.Render == nil {
                return dsl.View()
        }
        return evalJSXElement(f, *c.Render)
}

// evalJSXElement converts a JSXElement AST node to a dsl.Element.
func evalJSXElement(f *eml.File, el eml.JSXElement) dsl.Element {
        // Apply className from CSS first.
        var opts []dsl.Option
        for _, a := range el.Attrs {
                if a.Name == "className" && a.Value.Kind == "string" {
                        opts = append(opts, cssPropsToOptions(f, a.Value.String)...)
                }
        }
        // Apply explicit attrs.
        for _, a := range el.Attrs {
                if a.Name == "className" {
                        continue
                }
                opt := jsxAttrToOption(a)
                if opt != nil {
                        opts = append(opts, opt)
                }
        }
        // Build children.
        var children []dsl.Element
        for _, c := range el.Children {
                switch c.Kind {
                case "text":
                        children = append(children, dsl.Text(c.Text))
                case "expr":
                        children = append(children, dsl.Text(fmt.Sprintf("%v", c.Expr)))
                case "element":
                        children = append(children, evalJSXElement(f, *c.Element))
                }
        }

        // For Text and Button, the first arg is the text content.
        textContent := buildTextContent(el.Children)

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

func ensureGo() error {
        if _, err := exec.LookPath("go"); err != nil {
                return fmt.Errorf("go not found in PATH: %w", err)
        }
        return nil
}
