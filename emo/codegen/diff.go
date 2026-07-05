// Package codegen — diff.go — vtree diffing for hot reload.
//
// The diff function produces a list of mutations that the emo Go preview app
// can apply to its live Compose state without restarting the activity. This is
// the magic that makes emo feel like Expo: edit a Go file, save, and the
// device's UI updates in under a second.
package codegen

import (
	"reflect"

	"github.com/emo-framework/emo/dsl"
)

// Op is a single mutation in a diff.
type Op struct {
	Kind    string      `json:"kind"`    // "replace" | "insert" | "remove" | "updateProp" | "updateText" | "updateHandler"
	Path    []int       `json:"path"`    // child indices from root to target
	Element *dsl.Element `json:"element,omitempty"` // for replace/insert
	Prop    string      `json:"prop,omitempty"`
	Value   any         `json:"value,omitempty"`
	Text    string      `json:"text,omitempty"`
	Handler *dsl.HandlerRef `json:"handler,omitempty"`
}

// Diff computes the minimal list of Ops to transform old into neu. The
// algorithm is a simple recursive walk that matches children by ID, so
// reordering is treated as remove+insert. Good enough for live editing where
// the structure rarely changes wholesale.
func Diff(old, neu dsl.Element) []Op {
	var ops []Op
	diffElement(&ops, []int{}, old, neu)
	return ops
}

func diffElement(ops *[]Op, path []int, old, neu dsl.Element) {
	if old.ID == neu.ID && old.Kind == neu.Kind && old.Text == neu.Text && reflect.DeepEqual(old.Props, neu.Props) && sameHandlers(old.Handlers, neu.Handlers) && len(old.Children) == len(neu.Children) {
		// Fast path: shallow equal. Still need to recurse to be safe.
	}

	// Kind change → replace whole element.
	if old.Kind != neu.Kind {
		*ops = append(*ops, Op{Kind: "replace", Path: path, Element: &neu})
		return
	}

	// Text change.
	if old.Text != neu.Text {
		*ops = append(*ops, Op{Kind: "updateText", Path: path, Text: neu.Text})
	}

	// Prop changes.
	for k, v := range neu.Props {
		ov, ok := old.Props[k]
		if !ok || !reflect.DeepEqual(ov, v) {
			*ops = append(*ops, Op{Kind: "updateProp", Path: path, Prop: k, Value: v})
		}
	}
	for k := range old.Props {
		if _, ok := neu.Props[k]; !ok {
			*ops = append(*ops, Op{Kind: "updateProp", Path: path, Prop: k, Value: nil})
		}
	}

	// Handler changes (rare; usually a fresh token because handler closures
	// capture new state). We treat any handler set change as a full update.
	if !sameHandlers(old.Handlers, neu.Handlers) {
		*ops = append(*ops, Op{Kind: "updateHandler", Path: path, Handler: nil})
	}

	// Children: match by ID, emit inserts/removes/replaces for the rest.
	diffChildren(ops, path, old.Children, neu.Children)
}

func diffChildren(ops *[]Op, parent []int, old, neu []dsl.Element) {
	// Index old children by ID for lookup.
	oldIx := map[string]int{}
	for i, c := range old {
		oldIx[c.ID] = i
	}

	// Walk neu children in order; emit inserts when an ID is new.
	for i, c := range neu {
		childPath := append(append([]int{}, parent...), i)
		if oi, ok := oldIx[c.ID]; ok {
			// Exists in old — recurse. If positions differ we still emit a
			// move-less model: the preview app rebuilds children in order.
			diffElement(ops, childPath, old[oi], c)
			continue
		}
		// New child.
		ccopy := c
		*ops = append(*ops, Op{Kind: "insert", Path: childPath, Element: &ccopy})
	}

	// Detect removed IDs.
	neuIx := map[string]bool{}
	for _, c := range neu {
		neuIx[c.ID] = true
	}
	for i, c := range old {
		if !neuIx[c.ID] {
			childPath := append(append([]int{}, parent...), i)
			*ops = append(*ops, Op{Kind: "remove", Path: childPath})
		}
	}
}

func sameHandlers(a, b []dsl.HandlerRef) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Event != b[i].Event || a[i].Token != b[i].Token {
			return false
		}
	}
	return true
}
