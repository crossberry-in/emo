package embedded

import (
        "encoding/json"
        "testing"
)

func TestRuntimeStateInit(t *testing.T) {
        bundle := &Bundle{
                AppName:      "test",
                PackageName:  "dev.emo.test",
                InitialVTree: []byte(`{"kind":"text","text":"hello"}`),
                States: map[string]StateDef{
                        "count": {Name: "count", Type: "int", Initial: 0},
                        "name":  {Name: "name", Type: "string", Initial: "World"},
                },
                Handlers: map[string]Handler{},
        }
        r := NewRuntime(bundle)

        if r.State("count") != 0 {
                t.Errorf("count = %v, want 0", r.State("count"))
        }
        if r.State("name") != "World" {
                t.Errorf("name = %v, want World", r.State("name"))
        }
        if r.Version() != 0 {
                t.Errorf("version = %d, want 0", r.Version())
        }
}

func TestRuntimeDispatchIncrement(t *testing.T) {
        bundle := &Bundle{
                InitialVTree: []byte(`{"kind":"text","text":"Count: {count}"}`),
                States: map[string]StateDef{
                        "count": {Name: "count", Type: "int", Initial: 0},
                },
                Handlers: map[string]Handler{
                        "tok_123": {
                                Token: "tok_123",
                                Event: "click",
                                Mutations: []Mutation{
                                        {State: "count", Op: "assign", Expr: "count + 1"},
                                },
                        },
                },
        }
        r := NewRuntime(bundle)

        // Dispatch the increment handler.
        newTree, err := r.DispatchEvent("tok_123", nil)
        if err != nil {
                t.Fatalf("dispatch: %v", err)
        }
        if newTree == nil {
                t.Fatal("expected new tree, got nil")
        }
        if r.State("count") != 1 {
                t.Errorf("after increment, count = %v, want 1", r.State("count"))
        }
        if r.Version() != 1 {
                t.Errorf("version = %d, want 1", r.Version())
        }

        // Verify the new tree has the interpolated value.
        var tree map[string]any
        if err := json.Unmarshal(newTree, &tree); err != nil {
                t.Fatalf("parse tree: %v", err)
        }
        if tree["text"] != "Count: 1" {
                t.Errorf("tree text = %v, want 'Count: 1'", tree["text"])
        }
}

func TestRuntimeDispatchToggle(t *testing.T) {
        bundle := &Bundle{
                InitialVTree: []byte(`{"kind":"switch"}`),
                States: map[string]StateDef{
                        "enabled": {Name: "enabled", Type: "bool", Initial: false},
                },
                Handlers: map[string]Handler{
                        "tok_toggle": {
                                Token: "tok_toggle",
                                Event: "click",
                                Mutations: []Mutation{
                                        {State: "enabled", Op: "toggle"},
                                },
                        },
                },
        }
        r := NewRuntime(bundle)

        if r.State("enabled") != false {
                t.Errorf("initial enabled = %v, want false", r.State("enabled"))
        }

        _, err := r.DispatchEvent("tok_toggle", nil)
        if err != nil {
                t.Fatalf("dispatch: %v", err)
        }
        if r.State("enabled") != true {
                t.Errorf("after toggle, enabled = %v, want true", r.State("enabled"))
        }
}

func TestRuntimeDispatchSetFromPayload(t *testing.T) {
        bundle := &Bundle{
                InitialVTree: []byte(`{"kind":"text"}`),
                States: map[string]StateDef{
                        "text": {Name: "text", Type: "string", Initial: ""},
                },
                Handlers: map[string]Handler{
                        "tok_change": {
                                Token: "tok_change",
                                Event: "change",
                                Mutations: []Mutation{
                                        {State: "text", Op: "set"},
                                },
                        },
                },
        }
        r := NewRuntime(bundle)

        _, err := r.DispatchEvent("tok_change", "hello world")
        if err != nil {
                t.Fatalf("dispatch: %v", err)
        }
        if r.State("text") != "hello world" {
                t.Errorf("after set, text = %v, want 'hello world'", r.State("text"))
        }
}

func TestRuntimeNoChangeNoVersionBump(t *testing.T) {
        bundle := &Bundle{
                InitialVTree: []byte(`{}`),
                States:       map[string]StateDef{},
                Handlers: map[string]Handler{
                        "tok_noop": {
                                Token:     "tok_noop",
                                Event:     "click",
                                Mutations: []Mutation{},
                        },
                },
        }
        r := NewRuntime(bundle)

        newTree, err := r.DispatchEvent("tok_noop", nil)
        if err != nil {
                t.Fatalf("dispatch: %v", err)
        }
        if newTree != nil {
                t.Errorf("expected nil tree (no change), got %s", newTree)
        }
        if r.Version() != 0 {
                t.Errorf("version = %d, want 0 (no change)", r.Version())
        }
}

func TestLoadBundle(t *testing.T) {
        jsonStr := `{
                "appName": "test",
                "packageName": "dev.emo.test",
                "version": "1.0.0",
                "initialVTree": {"kind":"text","text":"hi"},
                "states": {
                        "count": {"name":"count","type":"int","initial":42}
                },
                "handlers": {}
        }`
        r, err := LoadBundle([]byte(jsonStr))
        if err != nil {
                t.Fatalf("load: %v", err)
        }
        // JSON unmarshals numbers as float64; convert to int for comparison.
        count, ok := r.State("count").(float64)
        if !ok {
                t.Fatalf("count type = %T, want float64", r.State("count"))
        }
        if int(count) != 42 {
                t.Errorf("count = %v, want 42", count)
        }
}

func TestInterpolateString(t *testing.T) {
        bundle := &Bundle{
                InitialVTree: []byte(`{"kind":"text","text":"Hello, {name}! Count: {count}"}`),
                States: map[string]StateDef{
                        "name":  {Name: "name", Type: "string", Initial: "Developer"},
                        "count": {Name: "count", Type: "int", Initial: 5},
                },
        }
        r := NewRuntime(bundle)

        result := r.interpolateString("Hello, {name}! Count: {count}")
        if result != "Hello, Developer! Count: 5" {
                t.Errorf("interpolate = %q, want 'Hello, Developer! Count: 5'", result)
        }
}

func TestEvalExprArithmetic(t *testing.T) {
        bundle := &Bundle{
                InitialVTree: []byte(`{}`),
                States: map[string]StateDef{
                        "count": {Name: "count", Type: "int", Initial: 10},
                },
        }
        r := NewRuntime(bundle)

        tests := []struct {
                expr string
                want any
        }{
                {"count + 1", 11},
                {"count - 1", 9},
                {"count + 5", 15},
                {"0", 0},
                {"42", 42},
                {"\"hello\"", "hello"},
                {"true", true},
                {"false", false},
        }

        for _, tt := range tests {
                got := r.evalExpr(tt.expr, nil)
                if got != tt.want {
                        t.Errorf("evalExpr(%q) = %v, want %v", tt.expr, got, tt.want)
                }
        }
}
