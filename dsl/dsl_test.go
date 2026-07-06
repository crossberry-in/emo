package dsl

import "testing"

func TestUseStateInt(t *testing.T) {
        resetHookFrame()
        v, set := UseStateInt(42)
        if v != 42 {
                t.Fatalf("initial = %d, want 42", v)
        }
        // Capture the frame the hook ran against so we can inspect it after set.
        frame := currentFrame
        set(43)
        if got := frame.states[0].value.(int); got != 43 {
                t.Fatalf("after set: %d, want 43", got)
        }
}

func TestElementConstruction(t *testing.T) {
        tree := Column(
                Spacing(8),
                Padding(16),
                Bg("#FFFFFFFF"),
                Children(
                        Text("hello", Font(18, "bold")),
                        Button("click me", OnClick(func() {})),
                ),
        )
        if tree.Kind != KindColumn {
                t.Fatalf("kind = %s, want column", tree.Kind)
        }
        if len(tree.Children) != 2 {
                t.Fatalf("children = %d, want 2", len(tree.Children))
        }
        if tree.Children[0].Text != "hello" {
                t.Fatalf("child 0 text = %q, want hello", tree.Children[0].Text)
        }
        if tree.Children[1].Kind != KindButton {
                t.Fatalf("child 1 kind = %s, want button", tree.Children[1].Kind)
        }
        if len(tree.Children[1].Handlers) != 1 || tree.Children[1].Handlers[0].Event != "click" {
                t.Fatalf("button missing click handler")
        }
}

func TestHandlerRegistry(t *testing.T) {
        called := false
        token := RegisterHandlerNoArg(func() { called = true })
        if !InvokeHandler(token, nil) {
                t.Fatal("invoke returned false")
        }
        if !called {
                t.Fatal("handler not called")
        }
        if InvokeHandler("nonexistent", nil) {
                t.Fatal("expected false for unknown token")
        }
}
