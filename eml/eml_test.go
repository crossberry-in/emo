package eml

import (
	"strings"
	"testing"
)

const counterEM = `// Counter.em — emo 0.1 SDK example
import { Header } from "./Header.em"

component Counter {
  state count = 0
  state name = "World"

  render {
    <Column spacing={16} padding={24} className="container">
      <Text fontSize={28} fontWeight="bold">{name} counter</Text>
      <Text fontSize={18}>Count: {count}</Text>
      <Row spacing={8}>
        <Button onClick={() => count = count - 1}>Decrement</Button>
        <Button onClick={() => count = count + 1}>Increment</Button>
      </Row>
      <Button onClick={() => count = 0}>Reset</Button>
      <Divider />
    </Column>
  }
}

style "./Counter.css"
`

const counterCSS = `.container {
  background: #FFFFFF;
  padding: 24dp;
  spacing: 16dp;
}
`

func TestParse(t *testing.T) {
	f, err := Parse(counterEM, "Counter.em")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(f.Imports) != 1 {
		t.Errorf("imports = %d, want 1", len(f.Imports))
	}
	if len(f.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(f.Components))
	}
	c := f.Components[0]
	if c.Name != "Counter" {
		t.Errorf("component name = %q, want Counter", c.Name)
	}
	if len(c.States) != 2 {
		t.Fatalf("states = %d, want 2", len(c.States))
	}
	if c.States[0].Name != "count" {
		t.Errorf("state 0 name = %q, want count", c.States[0].Name)
	}
	if c.States[1].Name != "name" {
		t.Errorf("state 1 name = %q, want name", c.States[1].Name)
	}
	if c.Render == nil {
		t.Fatal("render is nil")
	}
	if c.Render.Tag != "Column" {
		t.Errorf("render tag = %q, want Column", c.Render.Tag)
	}
	if len(c.Render.Children) != 5 {
		t.Errorf("render children = %d, want 5", len(c.Render.Children))
	}
	if f.StyleRef != "./Counter.css" {
		t.Errorf("styleRef = %q, want ./Counter.css", f.StyleRef)
	}
}

func TestGenerateGo(t *testing.T) {
	f, err := Parse(counterEM, "Counter.em")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ss, _ := ParseCSS(counterCSS)
	f.CSS = ss

	out, err := GenerateGo(f, CodegenOptions{PackageName: "main"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Must contain the component function.
	if !strings.Contains(out, "func Counter() dsl.Element") {
		t.Errorf("output missing Counter function:\n%s", out)
	}
	// Must contain App() entry point.
	if !strings.Contains(out, "func App() dsl.Element") {
		t.Errorf("output missing App entry point:\n%s", out)
	}
	// State declarations.
	if !strings.Contains(out, "dsl.UseStateInt(0)") {
		t.Errorf("output missing UseStateInt for count:\n%s", out)
	}
	if !strings.Contains(out, `dsl.UseStateString("World")`) {
		t.Errorf("output missing UseStateString for name:\n%s", out)
	}
	// CSS-derived props.
	if !strings.Contains(out, `dsl.Bg("#FFFFFF")`) {
		t.Errorf("output missing CSS background:\n%s", out)
	}
	// Event handler with state-assignment rewrite.
	if !strings.Contains(out, "setCount(count - 1)") {
		t.Errorf("output missing setCount(count - 1):\n%s", out)
	}
	if !strings.Contains(out, "setCount(0)") {
		t.Errorf("output missing setCount(0):\n%s", out)
	}
}

func TestParseCSS(t *testing.T) {
	ss, err := ParseCSS(counterCSS)
	if err != nil {
		t.Fatalf("parse css: %v", err)
	}
	if len(ss.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(ss.Rules))
	}
	r := ss.Rules[0]
	if r.Selector != ".container" {
		t.Errorf("selector = %q, want .container", r.Selector)
	}
	if r.Props["background"] != "#FFFFFF" {
		t.Errorf("background = %q, want #FFFFFF", r.Props["background"])
	}
	if r.Props["padding"] != "24dp" {
		t.Errorf("padding = %q, want 24dp", r.Props["padding"])
	}
}

func TestCSSPropToEmo(t *testing.T) {
	cases := []struct {
		cssK, cssV string
		wantK      string
		wantV      any
	}{
		{"background", "#FFFFFF", "background", "#FFFFFF"},
		{"padding", "16dp", "padding", float64(16)},
		{"font-size", "28sp", "fontSize", float64(28)},
		{"width", "match", "width", "match"},
	}
	for _, c := range cases {
		k, v, ok := CSSPropToEmo(c.cssK, c.cssV)
		if !ok {
			t.Errorf("CSSPropToEmo(%q,%q): not ok", c.cssK, c.cssV)
			continue
		}
		if k != c.wantK {
			t.Errorf("CSSPropToEmo(%q,%q): key = %q, want %q", c.cssK, c.cssV, k, c.wantK)
		}
		if v != c.wantV {
			t.Errorf("CSSPropToEmo(%q,%q): val = %v, want %v", c.cssK, c.cssV, v, c.wantV)
		}
	}
}

func TestRewriteStateAssign(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"count = count + 1", "setCount(count + 1)"},
		{"count = 0", "setCount(0)"},
		{"() => count = count - 1", "setCount(count - 1)"},
		{"() => count = 0", "setCount(0)"},
	}
	for _, c := range cases {
		got := rewriteStateAssign(c.in)
		if got != c.want {
			t.Errorf("rewriteStateAssign(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
