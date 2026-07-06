package eml

import (
        "fmt"
        "strings"
)

// Parser converts a token stream into a File AST.
type Parser struct {
        tokens []Token
        pos    int
        src    string // original source, for extracting raw expression text
}

// Parse parses a complete .em file.
func Parse(src string, filePath string) (*File, error) {
        tokens, err := Lex(src)
        if err != nil {
                return nil, err
        }
        p := &Parser{tokens: tokens, src: src}
        f := &File{Path: filePath}

        for !p.atEnd() {
                switch p.peek().Kind {
                case TKKeyword:
                        switch p.peek().Value {
                        case "import":
                                imp, err := p.parseImport()
                                if err != nil {
                                        return nil, err
                                }
                                f.Imports = append(f.Imports, imp)
                        case "component":
                                c, err := p.parseComponent()
                                if err != nil {
                                        return nil, err
                                }
                                f.Components = append(f.Components, c)
                        case "style":
                                p.advance() // style
                                if !p.check(TKString) {
                                        return nil, p.errorf("expected string path after 'style'")
                                }
                                f.StyleRef = unquote(p.advance().Value)
                        default:
                                return nil, p.errorf("unexpected keyword %q at top level", p.peek().Value)
                        }
                case TKComment:
                        p.advance()
                default:
                        return nil, p.errorf("unexpected token %q at top level", p.peek().Value)
                }
        }
        return f, nil
}

func (p *Parser) parseImport() (Import, error) {
        p.advance() // import
        if !p.check(TKLBrace) {
                return Import{}, p.errorf("expected '{' after 'import'")
        }
        p.advance() // {
        var names []string
        for {
                if !p.check(TKIdent) {
                        return Import{}, p.errorf("expected identifier in import list")
                }
                names = append(names, p.advance().Value)
                if p.check(TKComma) {
                        p.advance()
                        continue
                }
                break
        }
        if !p.check(TKRBrace) {
                return Import{}, p.errorf("expected '}' after import names")
        }
        p.advance() // }
        if !p.checkKeyword("from") {
                return Import{}, p.errorf("expected 'from' after import list")
        }
        p.advance() // from
        if !p.check(TKString) {
                return Import{}, p.errorf("expected string path after 'from'")
        }
        from := unquote(p.advance().Value)
        return Import{Names: names, From: from}, nil
}

func (p *Parser) parseComponent() (Component, error) {
        p.advance() // component
        if !p.check(TKIdent) {
                return Component{}, p.errorf("expected component name after 'component'")
        }
        name := p.advance().Value
        c := Component{Name: name}

        if !p.check(TKLBrace) {
                return Component{}, p.errorf("expected '{' after component name %q", name)
        }
        p.advance() // {

        for !p.check(TKRBrace) && !p.atEnd() {
                switch {
                case p.checkKeyword("state"):
                        sd, err := p.parseStateDecl()
                        if err != nil {
                                return Component{}, err
                        }
                        c.States = append(c.States, sd)
                case p.checkKeyword("render"):
                        p.advance() // render
                        if !p.check(TKLBrace) {
                                return Component{}, p.errorf("expected '{' after 'render'")
                        }
                        p.advance() // {
                        el, err := p.parseJSXElement()
                        if err != nil {
                                return Component{}, err
                        }
                        c.Render = el
                        if !p.check(TKRBrace) {
                                return Component{}, p.errorf("expected '}' after render block")
                        }
                        p.advance() // }
                default:
                        return Component{}, p.errorf("unexpected token %q in component body", p.peek().Value)
                }
        }
        if !p.check(TKRBrace) {
                return Component{}, p.errorf("expected '}' at end of component %q", name)
        }
        p.advance() // }
        return c, nil
}

func (p *Parser) parseStateDecl() (StateDecl, error) {
        p.advance() // state
        if !p.check(TKIdent) {
                return StateDecl{}, p.errorf("expected state name after 'state'")
        }
        name := p.advance().Value
        sd := StateDecl{Name: name}

        // Optional : type
        if p.check(TKColon) {
                p.advance()
                if !p.check(TKIdent) {
                        return StateDecl{}, p.errorf("expected type after ':' in state decl")
                }
                sd.Type = p.advance().Value
        }

        if !p.check(TKEq) {
                return StateDecl{}, p.errorf("expected '=' in state decl")
        }
        p.advance()

        // The default value is a Go expression — capture tokens until newline or
        // end of component. We grab raw source instead of structured tokens to
        // avoid having to implement a full Go expression parser.
        expr, err := p.captureExpr()
        if err != nil {
                return StateDecl{}, err
        }
        sd.Default = Expr{Raw: expr}
        return sd, nil
}

// captureExpr captures a Go expression as raw source text. The expression
// ends when we hit `}` (end of component) or another top-level keyword at
// depth 0. We use token positions to slice the original source, so the
// captured text preserves exact formatting (no space-joining artefacts).
func (p *Parser) captureExpr() (string, error) {
        startPos := p.peek().Pos
        depth := 0
        for !p.atEnd() {
                t := p.peek()
                if depth == 0 {
                        if t.Kind == TKRBrace || t.Kind == TKKeyword {
                                break
                        }
                }
                switch t.Kind {
                case TKLBrace, TKLParen:
                        depth++
                case TKRBrace, TKRParen:
                        depth--
                case TKEOF:
                        return "", p.errorf("unexpected EOF in expression")
                }
                p.advance()
        }
        endPos := p.peek().Pos
        return strings.TrimSpace(p.src[startPos:endPos]), nil
}

// parseJSXElement parses a single JSX element: <Tag attrs>children</Tag> or
// <Tag attrs />.
func (p *Parser) parseJSXElement() (*JSXElement, error) {
        if !p.check(TKLT) {
                return nil, p.errorf("expected '<' to start JSX element")
        }
        p.advance() // <
        if !p.check(TKIdent) {
                return nil, p.errorf("expected tag name after '<'")
        }
        tag := p.advance().Value
        el := &JSXElement{Tag: tag}

        // Attributes.
        for !p.check(TKGT) && !p.check(TKSlashClose) && !p.atEnd() {
                if !p.check(TKIdent) {
                        return nil, p.errorf("expected attribute name, got %q", p.peek().Value)
                }
                name := p.advance().Value
                if !p.check(TKEq) {
                        return nil, p.errorf("expected '=' after attribute %q", name)
                }
                p.advance() // =
                val, err := p.parseAttrValue()
                if err != nil {
                        return nil, err
                }
                el.Attrs = append(el.Attrs, JSXAttr{Name: name, Value: val})
        }

        if p.check(TKSlashClose) {
                p.advance() // />
                el.SelfClose = true
                return el, nil
        }
        if !p.check(TKGT) {
                return nil, p.errorf("expected '>' or '/>' to close JSX tag <%s>", tag)
        }
        p.advance() // >

        // Parse children until </tag>.
        children, err := p.parseJSXChildren(tag)
        if err != nil {
                return nil, err
        }
        el.Children = children
        return el, nil
}

func (p *Parser) parseAttrValue() (JSXAttrValue, error) {
        t := p.peek()
        switch t.Kind {
        case TKString:
                p.advance()
                return JSXAttrValue{Kind: "string", String: unquote(t.Value)}, nil
        case TKNumber:
                p.advance()
                return JSXAttrValue{Kind: "number", Number: t.Value}, nil
        case TKLBrace:
                // {expr} — capture raw expression text from source.
                expr := p.captureBraceExpr()
                return JSXAttrValue{Kind: "expr", Expr: expr}, nil
        default:
                return JSXAttrValue{}, p.errorf("expected attribute value (string, number, or {expr})")
        }
}

// captureBraceExpr consumes a { ... } block from the token stream and returns
// the raw source text between the braces. The current token must be `{`.
// After return, the current token is the one immediately after the matching `}`.
func (p *Parser) captureBraceExpr() string {
        startTok := p.advance() // {
        startPos := startTok.Pos + 1 // position after the {
        depth := 1
        endPos := startPos
        for !p.atEnd() {
                t := p.advance()
                if t.Kind == TKLBrace {
                        depth++
                } else if t.Kind == TKRBrace {
                        depth--
                        if depth == 0 {
                                endPos = t.Pos
                                break
                        }
                }
        }
        return strings.TrimSpace(p.src[startPos:endPos])
}

// parseJSXChildren parses children until we hit a closing </tag>. Children
// can be text, {expression}, or nested <element>.
func (p *Parser) parseJSXChildren(closeTag string) ([]JSXChild, error) {
        var out []JSXChild
        for !p.atEnd() {
                // Check for closing tag </closeTag>
                if p.check(TKLT) && p.peekN(1).Kind == TKSlash {
                        p.advance() // <
                        p.advance() // /
                        if !p.check(TKIdent) || p.peek().Value != closeTag {
                                return nil, p.errorf("expected </%s>, got </%s>", closeTag, p.peek().Value)
                        }
                        p.advance() // tag name
                        if !p.check(TKGT) {
                                return nil, p.errorf("expected '>' after </%s>", closeTag)
                        }
                        p.advance() // >
                        return out, nil
                }
                // Nested element
                if p.check(TKLT) {
                        el, err := p.parseJSXElement()
                        if err != nil {
                                return nil, err
                        }
                        out = append(out, JSXChild{Kind: "element", Element: el})
                        continue
                }
                // Expression
                if p.check(TKLBrace) {
                        out = append(out, JSXChild{Kind: "expr", Expr: p.captureBraceExpr()})
                        continue
                }
                // Text — anything else is treated as text. This is rare because the
                // lexer is structural, but stray tokens fall through here.
                t := p.advance()
                if strings.TrimSpace(t.Value) != "" {
                        out = append(out, JSXChild{Kind: "text", Text: t.Value})
                }
        }
        return nil, p.errorf("unexpected EOF in children of <%s>", closeTag)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (p *Parser) peek() Token      { return p.tokens[p.pos] }
func (p *Parser) peekN(n int) Token {
        if p.pos+n >= len(p.tokens) {
                return Token{Kind: TKEOF}
        }
        return p.tokens[p.pos+n]
}
func (p *Parser) advance() Token {
        t := p.tokens[p.pos]
        if p.pos < len(p.tokens)-1 {
                p.pos++
        }
        return t
}
func (p *Parser) check(k TokenKind) bool { return p.peek().Kind == k }
func (p *Parser) checkKeyword(v string) bool {
        t := p.peek()
        return t.Kind == TKKeyword && t.Value == v
}
func (p *Parser) atEnd() bool { return p.peek().Kind == TKEOF }
func (p *Parser) errorf(format string, args ...any) error {
        line := p.peek().Line
        return fmt.Errorf("line %d: "+format, append([]any{line}, args...)...)
}

func unquote(s string) string {
        if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
                return s[1 : len(s)-1]
        }
        return s
}
