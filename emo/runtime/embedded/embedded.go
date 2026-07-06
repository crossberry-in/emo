// Package embedded implements the emo production runtime — a self-contained
// state engine and vtree evaluator that runs inside the Android app with no
// dev server connection required.
//
// In Development Mode, the Go dev server transpiles .em files and pushes
// vtree JSON over WebSocket. State lives in the Go dev server.
//
// In Production Mode (emo build), the .em files are transpiled at build time
// into a Bundle: a JSON snapshot of the initial vtree plus a registry of
// state variables and event handlers. The embedded runtime loads this bundle
// at app startup and runs entirely on-device.
//
// The Android app talks to the embedded runtime via a thin JNI bridge
// (gomobile) or via a localhost HTTP server inside the app. Either way, the
// runtime evaluates state mutations and produces new vtrees that the
// VTreeRenderer renders as Jetpack Compose.
//
// This package is the heart of emo 0.2.0's production mode.
package embedded

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
)

// Bundle is the serialized production artifact produced by `emo build`.
// It contains everything the embedded runtime needs to run the app on-device.
type Bundle struct {
	// AppName is the human-readable app name.
	AppName string `json:"appName"`

	// PackageName is the Kotlin package (e.g. "dev.emo.myapp").
	PackageName string `json:"packageName"`

	// Version is the app version.
	Version string `json:"version"`

	// InitialVTree is the JSON-encoded dsl.Element tree for the app's
	// initial render. The embedded runtime uses this as the starting point
	// and applies state mutations to produce subsequent vtrees.
	InitialVTree json.RawMessage `json:"initialVTree"`

	// States is the registry of initial state values, keyed by state name.
	// The runtime tracks these in memory and applies mutations from event
	// handlers.
	States map[string]StateDef `json:"states"`

	// Handlers is the registry of event handlers. Each handler is a
	// sequence of state mutations expressed as a mini-DSL (assign, increment,
	// call) that the runtime can evaluate without a Go runtime.
	Handlers map[string]Handler `json:"handlers"`

	// Components is the registry of user-defined components (from .em files),
	// keyed by component name. Each component has its own state and render
	// function expressed as a vtree factory.
	Components map[string]ComponentDef `json:"components"`
}

// StateDef defines a single state variable.
type StateDef struct {
	Name    string `json:"name"`
	Type    string `json:"type"`    // "int", "string", "bool", "float"
	Initial any    `json:"initial"` // initial value
}

// Handler defines an event handler as a sequence of mutations.
type Handler struct {
	Token     string      `json:"token"`     // opaque handler ID (matches vtree)
	Event     string      `json:"event"`     // "click", "change", etc.
	Mutations []Mutation  `json:"mutations"` // state mutations to apply
}

// Mutation is a single state change. The runtime evaluates Expr against the
// current state and assigns the result to State.
type Mutation struct {
	State string `json:"state"` // target state variable name
	Op    string `json:"op"`    // "assign", "increment", "decrement", "toggle"
	Expr  string `json:"expr"`  // expression to evaluate (e.g. "count + 1", "0")
}

// ComponentDef defines a user component for the embedded runtime.
type ComponentDef struct {
	Name   string          `json:"name"`
	States []StateDef      `json:"states"`
	Render json.RawMessage `json:"render"` // vtree template with {state} placeholders
}

// ---------------------------------------------------------------------------
// Runtime
// ---------------------------------------------------------------------------

// Runtime is the on-device state engine. It loads a Bundle, tracks state,
// evaluates event handlers, and produces new vtrees for the renderer.
//
// The runtime is thread-safe; multiple event handlers may fire concurrently
// from the Android UI thread pool.
type Runtime struct {
	mu     sync.RWMutex
	bundle *Bundle

	// state holds the current values of all state variables, keyed by name.
	state map[string]any

	// version is incremented on every state change. The Android renderer
	// can poll this to detect changes efficiently.
	version uint64

	// currentTree is the last-rendered vtree (for diffing).
	currentTree []byte
}

// NewRuntime creates a new embedded runtime from a Bundle.
func NewRuntime(bundle *Bundle) *Runtime {
	r := &Runtime{
		bundle: bundle,
		state:  make(map[string]any, len(bundle.States)),
	}
	for name, def := range bundle.States {
		r.state[name] = def.Initial
	}
	r.currentTree = []byte(bundle.InitialVTree)
	return r
}

// LoadBundle parses a JSON bundle and returns a runtime.
func LoadBundle(data []byte) (*Runtime, error) {
	var bundle Bundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("parse bundle: %w", err)
	}
	return NewRuntime(&bundle), nil
}

// State returns the current value of a state variable.
func (r *Runtime) State(name string) any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state[name]
}

// AllState returns a snapshot of all current state values. Used by the
// renderer to interpolate {state} expressions in the vtree.
func (r *Runtime) AllState() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]any, len(r.state))
	for k, v := range r.state {
		out[k] = v
	}
	return out
}

// Version returns the current state version. Incremented on every mutation.
// The Android renderer polls this to detect changes without polling state.
func (r *Runtime) Version() uint64 {
	return atomic.LoadUint64(&r.version)
}

// CurrentTree returns the last-rendered vtree JSON.
func (r *Runtime) CurrentTree() []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]byte(nil), r.currentTree...)
}

// DispatchEvent processes an event from the Android UI. The token identifies
// which handler to run. The payload carries event-specific data (e.g. the
// new text value for onChange).
//
// Returns the new vtree JSON if state changed, or nil if no change.
func (r *Runtime) DispatchEvent(token string, payload any) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	handler, ok := r.bundle.Handlers[token]
	if !ok {
		return nil, fmt.Errorf("no handler registered for token %q", token)
	}

	// Apply each mutation in sequence.
	changed := false
	for _, m := range handler.Mutations {
		if r.applyMutation(m, payload) {
			changed = true
		}
	}

	if !changed {
		return nil, nil
	}

	// Bump version and re-render.
	atomic.AddUint64(&r.version, 1)
	newTree := r.renderTree()
	r.currentTree = newTree
	return newTree, nil
}

// applyMutation applies a single state mutation. Returns true if state changed.
func (r *Runtime) applyMutation(m Mutation, payload any) bool {
	switch m.Op {
	case "assign":
		val := r.evalExpr(m.Expr, payload)
		r.state[m.State] = val
		return true
	case "increment":
		if n, ok := r.state[m.State].(int); ok {
			r.state[m.State] = n + 1
			return true
		}
		if f, ok := r.state[m.State].(float64); ok {
			r.state[m.State] = f + 1
			return true
		}
	case "decrement":
		if n, ok := r.state[m.State].(int); ok {
			r.state[m.State] = n - 1
			return true
		}
		if f, ok := r.state[m.State].(float64); ok {
			r.state[m.State] = f - 1
			return true
		}
	case "toggle":
		if b, ok := r.state[m.State].(bool); ok {
			r.state[m.State] = !b
			return true
		}
	case "set":
		// Set state from the event payload (e.g. onChange text input).
		r.state[m.State] = payload
		return true
	}
	return false
}

// evalExpr evaluates a simple expression against the current state.
// Supported forms:
//   - "count + 1"        → state["count"] + 1
//   - "count - 1"        → state["count"] - 1
//   - "0"                → 0
//   - "\"text\""         → "text"
//   - "true" / "false"   → bool
//   - "$payload"         → the event payload
func (r *Runtime) evalExpr(expr string, payload any) any {
	expr = trimSpace(expr)

	// Payload reference.
	if expr == "$payload" || expr == "$value" {
		return payload
	}

	// String literal.
	if len(expr) >= 2 && expr[0] == '"' && expr[len(expr)-1] == '"' {
		return expr[1 : len(expr)-1]
	}

	// Boolean literal.
	if expr == "true" {
		return true
	}
	if expr == "false" {
		return false
	}

	// Addition/subtraction: "state + N" or "state - N"
	for _, op := range []string{"+", "-"} {
		if idx := indexStr(expr, " "+op+" "); idx > 0 {
			left := trimSpace(expr[:idx])
			right := trimSpace(expr[idx+3:])
			leftVal := r.resolveValue(left, payload)
			rightVal := r.resolveValue(right, payload)
			return arithmetic(leftVal, rightVal, op)
		}
	}

	// Try to parse as a number.
	return r.resolveValue(expr, payload)
}

// resolveValue resolves a single value reference (state name, number, or
// payload reference).
func (r *Runtime) resolveValue(ref string, payload any) any {
	ref = trimSpace(ref)
	if ref == "$payload" || ref == "$value" {
		return payload
	}
	// State reference.
	if v, ok := r.state[ref]; ok {
		return v
	}
	// Number literal.
	if n, err := parseInt(ref); err == nil {
		return n
	}
	if f, err := parseFloat(ref); err == nil {
		return f
	}
	// String fallback.
	return ref
}

// renderTree produces a new vtree JSON by interpolating current state values
// into the bundle's initial vtree template.
//
// In a full implementation, this would walk the vtree and replace {state}
// expressions. For the MVP, we return the initial tree with state values
// substituted into text nodes.
func (r *Runtime) renderTree() []byte {
	// Walk the JSON tree and replace {stateName} in string values with
	// the current state value.
	tree := make(map[string]any)
	if err := json.Unmarshal(r.bundle.InitialVTree, &tree); err != nil {
		return r.bundle.InitialVTree
	}
	r.interpolate(tree)
	out, err := json.Marshal(tree)
	if err != nil {
		return r.bundle.InitialVTree
	}
	return out
}

// interpolate walks a parsed JSON tree recursively and replaces {stateName}
// expressions in string values with the current state value.
func (r *Runtime) interpolate(node any) {
	switch v := node.(type) {
	case map[string]any:
		for key, val := range v {
			if s, ok := val.(string); ok {
				v[key] = r.interpolateString(s)
			} else {
				r.interpolate(val)
			}
		}
	case []any:
		for i, val := range v {
			if s, ok := val.(string); ok {
				v[i] = r.interpolateString(s)
			} else {
				r.interpolate(val)
			}
		}
	}
}

// interpolateString replaces {stateName} with the current value.
// e.g. "Count: {count}" → "Count: 42"
func (r *Runtime) interpolateString(s string) string {
	out := make([]byte, 0, len(s))
	i := 0
	for i < len(s) {
		if s[i] == '{' {
			j := indexByte(s[i:], '}')
			if j > 0 {
				name := trimSpace(s[i+1 : i+j])
				if val, ok := r.state[name]; ok {
					out = append(out, []byte(fmt.Sprintf("%v", val))...)
					i += j + 1
					continue
				}
			}
		}
		out = append(out, s[i])
		i++
	}
	return string(out)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func arithmetic(a, b any, op string) any {
	// Try int arithmetic first.
	ai, aOK := toInt(a)
	bi, bOK := toInt(b)
	if aOK && bOK {
		switch op {
		case "+":
			return ai + bi
		case "-":
			return ai - bi
		}
	}

	// Fall back to float.
	af, _ := toFloat(a)
	bf, _ := toFloat(b)
	switch op {
	case "+":
		return af + bf
	case "-":
		return af - bf
	}
	return 0
}

func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	}
	return 0, false
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not an int")
		}
		n = n*10 + int(c-'0')
	}
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	return n, nil
}

func parseFloat(s string) (float64, error) {
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
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	return f / divisor, nil
}

func trimSpace(s string) string {
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
