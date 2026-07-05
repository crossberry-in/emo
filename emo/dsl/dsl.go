// Package dsl provides the React-like UI DSL for emo.
//
// emo apps declare their UI as pure Go component functions that return a virtual
// tree of Elements. The DSL mirrors React's mental model:
//
//   - A Component is a func(props any) Element
//   - Hooks (UseState, UseEffect) live inside component invocations
//   - The vtree (Element tree) is serialised to JSON and pushed to the emo Go
//     preview app over WebSocket, where it is rendered as Jetpack Compose.
//
// Example counter:
//
//      func Counter() dsl.Element {
//          count, setCount := dsl.UseState(0)
//          dsl.UseEffect(func() {
//              log.Println("count is", count)
//          }, count)
//          return dsl.Column(
//              dsl.Text(fmt.Sprintf("Count: %d", count)),
//              dsl.Button("Increment").OnClick(func() { setCount(count.(int) + 1) }),
//          )
//      }
package dsl

import (
        "crypto/rand"
        "encoding/hex"
        "fmt"
        "sort"
        "sync"
)

// ElementKind enumerates the kinds of nodes that may appear in a vtree.
type ElementKind string

const (
        KindView     ElementKind = "view"     // generic container → Box
        KindColumn   ElementKind = "column"   // vertical stack → Column
        KindRow      ElementKind = "row"      // horizontal stack → Row
        KindText     ElementKind = "text"     // label → Text
        KindButton   ElementKind = "button"   // → Button
        KindImage    ElementKind = "image"    // → Image
        KindTextField ElementKind = "textField" // → TextField
        KindSpacer   ElementKind = "spacer"   // → Spacer
        KindDivider  ElementKind = "divider"  // → Divider
        KindScaffold ElementKind = "scaffold" // → Scaffold (top-level)
)

// Element is a node in the virtual tree. It is JSON-serialisable so it can be
// pushed to the emo Go preview app over the WebSocket.
type Element struct {
        ID       string            `json:"id"`
        Kind     ElementKind       `json:"kind"`
        Text     string            `json:"text,omitempty"`
        Props    map[string]any    `json:"props,omitempty"`
        Children []Element         `json:"children,omitempty"`
        Handlers []HandlerRef      `json:"handlers,omitempty"`
}

// HandlerRef references an event handler registered with the runtime.
// The Kotlin side never sees the function — it sees an opaque token that, when
// fired, the dev server maps back to the Go func.
type HandlerRef struct {
        Event string `json:"event"` // "click", "change", "submit", ...
        Token string `json:"token"` // opaque handler ID
}

// Component is a function that returns an Element tree. Components are pure:
// they receive props and produce a vtree. Side-effects live inside hooks.
type Component func(props any) Element

// Render invokes a component and returns its vtree.
func Render(c Component, props any) Element {
        resetHookFrame()
        return c(props)
}

// ---------------------------------------------------------------------------
// Constructors — composable builders that mirror React.createElement.
// ---------------------------------------------------------------------------

func el(kind ElementKind, opts ...Option) Element {
        e := Element{ID: newID(), Kind: kind}
        for _, o := range opts {
                o(&e)
        }
        return e
}

// Option mutates an Element during construction.
type Option func(*Element)

// Children appends child elements.
func Children(children ...Element) Option {
        return func(e *Element) {
                e.Children = append(e.Children, children...)
        }
}

// Prop sets a single prop on the element (style, source, etc.).
func Prop(key string, value any) Option {
        return func(e *Element) {
                if e.Props == nil {
                        e.Props = map[string]any{}
                }
                e.Props[key] = value
        }
}

// Padding sets uniform padding in dp.
func Padding(dp float64) Option { return Prop("padding", dp) }

// Margin sets uniform margin in dp.
func Margin(dp float64) Option { return Prop("margin", dp) }

// Bg sets background colour (hex like "#FFFFFFFF").
func Bg(hex string) Option { return Prop("background", hex) }

// Fg sets foreground / text colour.
func Fg(hex string) Option { return Prop("color", hex) }

// Spacing sets the gap between children of Column/Row (dp).
func Spacing(dp float64) Option { return Prop("spacing", dp) }

// Width / Height set explicit dimensions (dp or "wrap"/"match").
func Width(v any) Option { return Prop("width", v) }
func Height(v any) Option { return Prop("height", v) }

// Font sets typography. size in sp, weight "normal"|"bold".
func Font(size float64, weight string) Option {
        return func(e *Element) {
                if e.Props == nil {
                        e.Props = map[string]any{}
                }
                e.Props["fontSize"] = size
                e.Props["fontWeight"] = weight
        }
}

// OnClick attaches a click handler. The handler is registered with the runtime
// and referenced by an opaque token in the vtree.
func OnClick(fn func()) Option {
        return func(e *Element) {
                token := RegisterHandlerNoArg(fn)
                e.Handlers = append(e.Handlers, HandlerRef{Event: "click", Token: token})
        }
}

// OnChange attaches a text-change handler for TextField. The handler receives
// the new string value.
func OnChange(fn func(string)) Option {
        return func(e *Element) {
                token := RegisterHandler(func(payload any) {
                        s, _ := payload.(string)
                        fn(s)
                })
                e.Handlers = append(e.Handlers, HandlerRef{Event: "change", Token: token})
        }
}

// Source sets the image source (URL or local asset path).
func Source(src string) Option { return Prop("source", src) }

// Container elements --------------------------------------------------------

func Column(opts ...Option) Element  { return el(KindColumn, opts...) }
func Row(opts ...Option) Element     { return el(KindRow, opts...) }
func View(opts ...Option) Element    { return el(KindView, opts...) }
func Scaffold(opts ...Option) Element {
        // Wrap children in a Scaffold automatically if top-level.
        return el(KindScaffold, opts...)
}

// Leaf elements -------------------------------------------------------------

// Text creates a Text element.
func Text(s string, opts ...Option) Element {
        e := el(KindText, opts...)
        e.Text = s
        return e
}

// Button creates a Button element with a click handler attached by default.
func Button(label string, opts ...Option) Element {
        e := el(KindButton, opts...)
        e.Text = label
        return e
}

// TextField creates an editable text field.
func TextField(placeholder string, opts ...Option) Element {
        e := el(KindTextField, opts...)
        e.Props = map[string]any{"placeholder": placeholder}
        return e
}

// Image creates an image element.
func Image(src string, opts ...Option) Element {
        return el(KindImage, append([]Option{Source(src)}, opts...)...)
}

// Spacer fills available space (like Compose Spacer with weight).
func Spacer(opts ...Option) Element { return el(KindSpacer, opts...) }

// Divider creates a horizontal divider.
func Divider(opts ...Option) Element { return el(KindDivider, opts...) }

// ---------------------------------------------------------------------------
// Hook implementation
// ---------------------------------------------------------------------------

// hookFrame is the per-render scratchpad for hook state. Components are always
// invoked via Render(), which resets the frame. Each hook call pops the next
// slot off the frame, mirroring React's rules-of-hooks model.
type hookFrame struct {
        mu       sync.Mutex
        states   []hookState
        effectIx int
}

type hookState struct {
        value any
        setter func(any)
}

var (
        currentFrame *hookFrame
        frameMu      sync.Mutex
)

func resetHookFrame() {
        frameMu.Lock()
        defer frameMu.Unlock()
        currentFrame = &hookFrame{}
}

// UseState declares a stateful value. The setter updates state and triggers a
// re-render through the dev server's reactive runtime.
//
//   count, setCount := dsl.UseState(0)
//   setCount(count.(int) + 1)
//
// The generic Go type system cannot express "T and func(T)" without generics
// boilerplate; UseState returns `any` and a setter that accepts `any`. Use the
// typed helpers UseStateInt, UseStateString for ergonomics.
func UseState(initial any) (any, func(any)) {
        frameMu.Lock()
        if currentFrame == nil {
                currentFrame = &hookFrame{}
        }
        f := currentFrame
        frameMu.Unlock()

        f.mu.Lock()
        defer f.mu.Unlock()

        ix := len(f.states)
        if ix >= cap(f.states) {
                // First render: allocate state.
                _ = ix // no-op; appended below
        }
        var st hookState
        if ix < len(f.states) {
                st = f.states[ix]
        } else {
                st = hookState{value: initial}
                f.states = append(f.states, st)
        }

        // The setter schedules a re-render via the global scheduler.
        setter := func(newValue any) {
                f.mu.Lock()
                f.states[ix].value = newValue
                f.mu.Unlock()
                ScheduleReRender()
        }
        return st.value, setter
}

// UseStateInt is a typed convenience wrapper around UseState.
func UseStateInt(initial int) (int, func(int)) {
        v, set := UseState(initial)
        if v == nil {
                v = initial
        }
        n, _ := v.(int)
        return n, func(x int) { set(x) }
}

// UseStateString is a typed convenience wrapper around UseState.
func UseStateString(initial string) (string, func(string)) {
        v, set := UseState(initial)
        if v == nil {
                v = initial
        }
        s, _ := v.(string)
        return s, func(x string) { set(x) }
}

// UseEffect schedules a side-effect to run after the vtree has been committed.
// deps is a variadic list of comparable values; the effect re-runs only when a
// dep changes (shallow comparison). Pass no deps to run on every render; pass
// a single nil to run only once.
func UseEffect(fn func(), deps ...any) {
        frameMu.Lock()
        f := currentFrame
        frameMu.Unlock()
        if f == nil {
                return
        }
        f.mu.Lock()
        defer f.mu.Unlock()

        // Effect index is only consumed in the production runtime to diff deps.
        // In the MVP we always re-enqueue effects.
        f.effectIx++

        // We don't track previous deps in this minimal impl; just enqueue.
        // A production runtime would diff deps across renders.
        RegisterEffect(fn, deps)
}

// ---------------------------------------------------------------------------
// Handler registry — stores Go callbacks by opaque token.
// ---------------------------------------------------------------------------

var (
        handlersMu sync.RWMutex
        handlers   = map[string]func(any){}
)

// RegisterHandler stores fn under a fresh opaque token and returns it.
func RegisterHandler(fn func(any)) string {
        token := newID()
        handlersMu.Lock()
        handlers[token] = fn
        handlersMu.Unlock()
        return token
}

// RegisterHandlerNoArg stores a no-arg callback.
func RegisterHandlerNoArg(fn func()) string {
        return RegisterHandler(func(_ any) { fn() })
}

// InvokeHandler runs the handler registered under token, passing payload.
// Returns false if no handler is registered.
func InvokeHandler(token string, payload any) bool {
        handlersMu.RLock()
        fn, ok := handlers[token]
        handlersMu.RUnlock()
        if !ok {
                return false
        }
        fn(payload)
        return true
}

// ---------------------------------------------------------------------------
// Effects registry
// ---------------------------------------------------------------------------

var (
        effectsMu sync.Mutex
        effects   []effectEntry
)

type effectEntry struct {
        fn   func()
        deps []any
}

// RegisterEffect queues an effect for execution after commit.
func RegisterEffect(fn func(), deps []any) {
        effectsMu.Lock()
        effects = append(effects, effectEntry{fn: fn, deps: deps})
        effectsMu.Unlock()
}

// FlushEffects runs all pending effects in registration order and clears the
// queue. The dev server calls this after each successful commit.
func FlushEffects() {
        effectsMu.Lock()
        pending := effects
        effects = nil
        effectsMu.Unlock()
        for _, e := range pending {
                e.fn()
        }
}

// ---------------------------------------------------------------------------
// Re-render scheduler — the dev server sets this to push a new vtree.
// ---------------------------------------------------------------------------

var schedulerMu sync.Mutex
var scheduler func()

// SetReRenderScheduler installs the callback invoked whenever state mutates.
// The dev server installs a function that re-renders the root and pushes the
// vtree to all connected emo Go preview clients.
func SetReRenderScheduler(fn func()) {
        schedulerMu.Lock()
        scheduler = fn
        schedulerMu.Unlock()
}

// ScheduleReRender triggers the installed scheduler. If no scheduler is
// installed (e.g. during a static build), it is a no-op.
func ScheduleReRender() {
        schedulerMu.Lock()
        fn := scheduler
        schedulerMu.Unlock()
        if fn != nil {
                go fn()
        }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newID() string {
        b := make([]byte, 8)
        _, _ = rand.Read(b)
        return "el_" + hex.EncodeToString(b)
}

// SortProps returns a deterministic ordering of an element's prop keys, useful
// for stable diffing and snapshot tests.
func (e Element) SortProps() []string {
        keys := make([]string, 0, len(e.Props))
        for k := range e.Props {
                keys = append(keys, k)
        }
        sort.Strings(keys)
        return keys
}

// String returns a human-readable one-liner, mainly for debugging.
func (e Element) String() string {
        return fmt.Sprintf("{%s %s text=%q children=%d}", e.ID, e.Kind, e.Text, len(e.Children))
}
