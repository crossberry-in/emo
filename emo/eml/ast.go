// Package eml implements the emo Markup Language — emo's custom DSL for
// declaring UI in a Svelte/Vue-like single-file-component style.
//
// An .em file contains:
//
//   1. Optional `import` declarations for other .em components.
//   2. A single `component <Name> { ... }` block with:
//        - `state` declarations
//        - a `render { ... }` block with JSX-like syntax
//   3. An optional `style "<path>.css"` reference at the end.
//
// The transpiler converts this to a Go source file that uses the emo dsl
// package. CSS classes referenced via className="..." on elements are looked
// up in the parsed .css file and merged into the element's props at build
// time, so the runtime sees a single flat prop map.
//
// This package is the heart of emo 0.1 SDK.
package eml

// File is the AST root for a single .em file.
type File struct {
        Path       string       // source .em file path
        Imports    []Import     // import declarations
        Components []Component  // top-level components (typically just one)
        StyleRef   string       // optional path to .css file
        CSS        *Stylesheet  // parsed CSS, if StyleRef was resolvable
}

// Import references another .em component file.
type Import struct {
        Names []string // imported component names
        From  string   // path literal, e.g. "./Button.em"
}

// Component is a single component definition.
type Component struct {
        Name   string     // PascalCase component name
        States []StateDecl
        Render *JSXElement
}

// StateDecl declares a reactive state variable.
type StateDecl struct {
        Name    string // variable name
        Type    string // "int" | "string" | "bool" | "float" | "" (inferred)
        Default Expr    // initial value expression (raw Go)
}

// Expr wraps a raw Go expression captured from the .em source. We don't
// fully parse Go expressions — we capture their source text verbatim.
type Expr struct {
        Raw string
}

// JSXElement is a node in the JSX-like render tree.
type JSXElement struct {
        Tag        string     // element tag, e.g. "Column", "Text", "Button"
        Attrs      []JSXAttr  // ordered attributes
        Children   []JSXChild // child nodes
        SelfClose  bool       // true if <Tag .../>
}

// JSXAttr is a single attribute on a JSX element.
type JSXAttr struct {
        Name  string // attribute name, e.g. "spacing", "onClick"
        Value JSXAttrValue
}

// JSXAttrValue is the value of an attribute — either a string literal, an
// expression (in `{...}`), or a number.
type JSXAttrValue struct {
        Kind  string // "string" | "expr" | "number"
        String string
        Expr   string
        Number string
}

// JSXChild is one child of a JSX element — either text, an `{expression}`,
// or a nested element.
type JSXChild struct {
        Kind     string // "text" | "expr" | "element"
        Text     string
        Expr     string
        Element  *JSXElement
}
