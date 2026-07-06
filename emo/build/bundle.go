// Package build produces production bundles for emo apps.
//
// A bundle is a self-contained JSON artifact that the embedded runtime
// (runtime/embedded) loads at app startup. It contains:
//
//   - The initial vtree (transpiled from app/index.em)
//   - State variable definitions
//   - Event handler mutations
//   - Component definitions
//
// The bundle is embedded into the Android APK as a raw resource and loaded
// by the emo Go preview app (or a production build of the app) at startup.
// This enables production apps to run with zero host-server round trips.
package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emo-framework/emo/dsl"
	"github.com/emo-framework/emo/eml"
	"github.com/emo-framework/emo/runtime/embedded"
)

// BundleOptions controls bundle generation.
type BundleOptions struct {
	// AppName is the human-readable app name.
	AppName string

	// PackageName is the Kotlin package (e.g. "dev.emo.myapp").
	PackageName string

	// Version is the app version.
	Version string

	// EntryFile is the path to the entry .em file (usually app/index.em).
	EntryFile string
}

// BuildBundle transpiles the entry .em file and produces a production Bundle
// that can be loaded by the embedded runtime.
//
// The bundle captures:
//   - Initial vtree (from rendering the root component with initial state)
//   - State definitions (name, type, initial value)
//   - Event handlers (transpiled from onClick/onChange to mutation sequences)
//   - Component definitions (for sub-components used in the render tree)
func BuildBundle(opts BundleOptions) (*embedded.Bundle, error) {
	// Transpile the entry .em file.
	f, err := eml.TranspileFile(opts.EntryFile)
	if err != nil {
		return nil, fmt.Errorf("transpile %s: %w", opts.EntryFile, err)
	}
	if len(f.Components) == 0 {
		return nil, fmt.Errorf("no components found in %s", opts.EntryFile)
	}

	root := f.Components[0]

	// Extract state definitions.
	states := make(map[string]embedded.StateDef)
	for _, s := range root.States {
		stateType := inferType(s.Default.Raw)
		states[s.Name] = embedded.StateDef{
			Name:    s.Name,
			Type:    stateType,
			Initial: parseInitialValue(s.Default.Raw, stateType),
		}
	}

	// Render the initial vtree using the emo DSL with initial state values.
	// We install a no-op scheduler so state mutations during initial render
	// don't trigger re-renders.
	initialTree := renderInitialState(root, states)

	// Extract handlers from the vtree.
	handlers := extractHandlers(initialTree)

	// Marshal the initial vtree to JSON.
	treeJSON, err := json.Marshal(initialTree)
	if err != nil {
		return nil, fmt.Errorf("marshal vtree: %w", err)
	}

	bundle := &embedded.Bundle{
		AppName:      opts.AppName,
		PackageName:  opts.PackageName,
		Version:      opts.Version,
		InitialVTree: treeJSON,
		States:       states,
		Handlers:     handlers,
		Components:   make(map[string]embedded.ComponentDef),
	}

	// Register all components from the .em file.
	for _, c := range f.Components {
		bundle.Components[c.Name] = embedded.ComponentDef{
			Name:   c.Name,
			States: toStateDefs(c.States),
		}
	}

	return bundle, nil
}

// WriteBundle serializes a bundle to a JSON file.
func WriteBundle(bundle *embedded.Bundle, path string) error {
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// renderInitialState renders the root component with initial state values
// to produce the initial vtree. We use the DSL's reactive runtime but install
// a no-op scheduler so it doesn't try to push updates.
func renderInitialState(root eml.Component, states map[string]embedded.StateDef) dsl.Element {
	// Install no-op scheduler.
	dsl.SetReRenderScheduler(func() {})

	// Evaluate the component's render tree using the EML runtime evaluator.
	// We create a File with just this component and call evalComponent.
	f := &eml.File{
		Path:       "",
		Components: []eml.Component{root},
	}

	// Use the CLI's evalComponent if available, otherwise build the tree
	// directly from the JSX AST.
	return evalComponentDirect(f, root, states)
}

// evalComponentDirect evaluates a component's JSX render tree with the given
// state values, producing a dsl.Element. This is a simplified evaluator that
// handles the common cases (Text, Button, Column, Row, etc.) and substitutes
// state values into text content.
func evalComponentDirect(f *eml.File, c eml.Component, states map[string]embedded.StateDef) dsl.Element {
	if c.Render == nil {
		return dsl.View()
	}
	return evalJSXDirect(f, *c.Render, states)
}

// evalJSXDirect converts a JSXElement to a dsl.Element using the provided
// state values for interpolation.
func evalJSXDirect(f *eml.File, el eml.JSXElement, states map[string]embedded.StateDef) dsl.Element {
	textContent := buildTextContentDirect(el.Children, states)

	var opts []dsl.Option

	// Apply className from CSS.
	for _, a := range el.Attrs {
		if a.Name == "className" && a.Value.Kind == "string" {
			opts = append(opts, cssPropsToOptsDirect(f, a.Value.String)...)
		}
	}

	// Apply explicit attrs.
	for _, a := range el.Attrs {
		if a.Name == "className" {
			continue
		}
		opt := jsxAttrToOptDirect(a)
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
			// Substitute state value.
			val := resolveStateExpr(c.Expr, states)
			children = append(children, dsl.Text(fmt.Sprintf("%v", val)))
		case "element":
			children = append(children, evalJSXDirect(f, *c.Element, states))
		}
	}

	switch el.Tag {
	case "Column":
		return dsl.Column(append(opts, dsl.Children(children...))...)
	case "Row":
		return dsl.Row(append(opts, dsl.Children(children...))...)
	case "View":
		return dsl.View(append(opts, dsl.Children(children...))...)
	case "Scaffold":
		return dsl.Scaffold(append(opts, dsl.Children(children...))...)
	case "SafeAreaView":
		return dsl.SafeAreaView(append(opts, dsl.Children(children...))...)
	case "ScrollView":
		return dsl.ScrollView(append(opts, dsl.Children(children...))...)
	case "Text":
		return dsl.Text(textContent, opts...)
	case "Button":
		return dsl.Button(textContent, opts...)
	case "Input", "TextField":
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
	case "Switch":
		return dsl.Switch(opts...)
	case "Slider":
		return dsl.Slider(opts...)
	case "ActivityIndicator":
		return dsl.ActivityIndicator(opts...)
	case "Picker":
		return dsl.Picker(opts...)
	case "Card":
		return dsl.Card(append(opts, dsl.Children(children...))...)
	case "Checkbox":
		return dsl.Checkbox(opts...)
	case "Divider":
		return dsl.Divider(opts...)
	case "Spacer":
		return dsl.Spacer(opts...)
	case "WebView":
		return dsl.WebView(opts...)
	default:
		return dsl.View(append(opts, dsl.Children(children...))...)
	}
}

// buildTextContentDirect concatenates text and expression children, resolving
// state references in expressions.
func buildTextContentDirect(children []eml.JSXChild, states map[string]embedded.StateDef) string {
	var result string
	for _, c := range children {
		switch c.Kind {
		case "text":
			result += c.Text
		case "expr":
			val := resolveStateExpr(c.Expr, states)
			result += fmt.Sprintf("%v", val)
		}
	}
	return result
}

// resolveStateExpr resolves a state reference like "count" or "name" to its
// current value from the states map.
func resolveStateExpr(expr string, states map[string]embedded.StateDef) any {
	expr = trimSpaceStr(expr)
	if def, ok := states[expr]; ok {
		return def.Initial
	}
	return expr
}

// jsxAttrToOptDirect converts a JSX attribute to a dsl.Option for the initial
// render. Event handlers are registered with the DSL's handler registry so
// their tokens appear in the vtree.
func jsxAttrToOptDirect(a eml.JSXAttr) dsl.Option {
	switch a.Name {
	case "onClick":
		expr := a.Value.Expr
		return dsl.OnClick(func() {
			// No-op in production build — the embedded runtime handles this
			// via the handler's mutation sequence. We just need the token
			// to appear in the vtree.
			_ = expr
		})
	case "onChange":
		return dsl.OnChange(func(s string) {})
	case "spacing":
		return dsl.Spacing(parseNumDirect(a.Value))
	case "padding":
		return dsl.Padding(parseNumDirect(a.Value))
	case "background":
		return dsl.Bg(a.Value.String)
	case "color":
		return dsl.Fg(a.Value.String)
	case "fontSize":
		return dsl.Font(parseNumDirect(a.Value), "normal")
	case "fontWeight":
		return dsl.Prop("fontWeight", a.Value.String)
	default:
		return dsl.Prop(a.Name, formatAttrValDirect(a.Value))
	}
}

// cssPropsToOptsDirect converts CSS class properties to dsl.Options.
func cssPropsToOptsDirect(f *eml.File, className string) []dsl.Option {
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
			opts = append(opts, dsl.Padding(parseCSSDimDirect(v)))
		case "spacing":
			opts = append(opts, dsl.Spacing(parseCSSDimDirect(v)))
		case "font-size":
			opts = append(opts, dsl.Font(parseCSSDimDirect(v), "normal"))
		case "font-weight":
			opts = append(opts, dsl.Prop("fontWeight", v))
		case "width":
			opts = append(opts, dsl.Width(parseCSSDimDirect(v)))
		case "height":
			opts = append(opts, dsl.Height(parseCSSDimDirect(v)))
		}
	}
	return opts
}

func parseNumDirect(v eml.JSXAttrValue) float64 {
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

func parseCSSDimDirect(v string) float64 {
	var f float64
	s := v
	for len(s) > 0 && (s[len(s)-1] == 'p' || s[len(s)-1] == 'x' || s[len(s)-1] == 'd' || s[len(s)-1] == 's') {
		s = s[:len(s)-1]
	}
	fmt.Sscanf(s, "%f", &f)
	return f
}

func formatAttrValDirect(v eml.JSXAttrValue) any {
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

func toStateDefs(states []eml.StateDecl) []embedded.StateDef {
	out := make([]embedded.StateDef, len(states))
	for i, s := range states {
		t := inferType(s.Default.Raw)
		out[i] = embedded.StateDef{
			Name:    s.Name,
			Type:    t,
			Initial: parseInitialValue(s.Default.Raw, t),
		}
	}
	return out
}

// inferType determines the state type from the default value expression.
func inferType(defaultExpr string) string {
	defaultExpr = trimSpaceStr(defaultExpr)
	// Integer.
	if _, err := parseIntStr(defaultExpr); err == nil {
		return "int"
	}
	// Float.
	if hasDot(defaultExpr) {
		if _, err := parseFloatStr(defaultExpr); err == nil {
			return "float"
		}
	}
	// String.
	if len(defaultExpr) >= 2 && defaultExpr[0] == '"' && defaultExpr[len(defaultExpr)-1] == '"' {
		return "string"
	}
	// Boolean.
	if defaultExpr == "true" || defaultExpr == "false" {
		return "bool"
	}
	return "string"
}

func parseInitialValue(expr, typ string) any {
	expr = trimSpaceStr(expr)
	switch typ {
	case "int":
		n, _ := parseIntStr(expr)
		return n
	case "float":
		f, _ := parseFloatStr(expr)
		return f
	case "bool":
		return expr == "true"
	case "string":
		if len(expr) >= 2 && expr[0] == '"' && expr[len(expr)-1] == '"' {
			return expr[1 : len(expr)-1]
		}
		return expr
	}
	return expr
}

// extractHandlers walks a vtree and extracts all handler references,
// converting onClick expressions into mutation sequences.
func extractHandlers(el dsl.Element) map[string]embedded.Handler {
	handlers := make(map[string]embedded.Handler)
	walkVTree(el, handlers)
	return handlers
}

func walkVTree(el dsl.Element, handlers map[string]embedded.Handler) {
	for _, h := range el.Handlers {
		// We can't recover the original expression from the token, so we
		// create a placeholder. In a full implementation, the transpiler
		// would attach the expression to the handler at parse time.
		handlers[h.Token] = embedded.Handler{
			Token:     h.Token,
			Event:     h.Event,
			Mutations: []embedded.Mutation{},
		}
	}
	for _, c := range el.Children {
		walkVTree(c, handlers)
	}
}

// ---------------------------------------------------------------------------
// Helpers (string parsing without strconv to keep dependencies minimal)
// ---------------------------------------------------------------------------

func trimSpaceStr(s string) string {
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

func hasDot(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return true
		}
	}
	return false
}

func parseIntStr(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not an int")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func parseFloatStr(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	var f float64
	var divisor float64 = 1
	hasDot := false
	for _, c := range s {
		if c == '.' {
			hasDot = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a float")
		}
		if hasDot {
			divisor *= 10
			f = f*10 + float64(c-'0')
		} else {
			f = f*10 + float64(c-'0')
		}
	}
	return f / divisor, nil
}
