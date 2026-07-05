package eml

import (
        "fmt"
        "strings"
        "unicode"
)

// TokenKind enumerates the lexer's token types.
type TokenKind int

const (
        TKEOF TokenKind = iota
        TKIdent     // identifier
        TKKeyword   // component, state, render, style, import, from
        TKString    // "..."
        TKNumber    // 123, 12.5
        TKLBrace    // {
        TKRBrace    // }
        TKLParen    // (
        TKRParen    // )
        TKLT        // <
        TKGT        // >
        TKSlashClose // /> (self-closing)
        TKSlash     // /
        TKEq        // =
        TKArrow     // =>
        TKColon     // :
        TKDot       // .
        TKComma     // ,
        TKExpr      // { ... } — a Go expression captured verbatim
        TKText      // raw text between JSX tags
        TKComment   // // ... or /* ... */
)

// Token is a single lexer token.
type Token struct {
        Kind  TokenKind
        Value string
        Pos   int // byte offset in source
        Line  int // 1-indexed line
}

// Lexer turns .em source text into a stream of tokens. The lexer is
// context-aware: inside a JSX tag it produces structural tokens (<, >, />),
// while inside `{...}` expression braces it captures the entire expression
// as a single TKExpr token (so we don't have to fully parse Go).
type Lexer struct {
        src      string
        pos      int
        line     int
        tokens   []Token
        inJSXTag bool // tracks whether we're inside <Tag ...>
}

// Lex tokenises the source. The result is a flat token list; the parser
// interprets structure.
func Lex(src string) ([]Token, error) {
        l := &Lexer{src: src, line: 1}
        for l.pos < len(l.src) {
                l.skipWhitespaceAndComments()
                if l.pos >= len(l.src) {
                        break
                }
                c := l.src[l.pos]
                switch {
                case c == '/' && l.peek(1) == '/':
                        l.consumeLineComment()
                case c == '/' && l.peek(1) == '*':
                        l.consumeBlockComment()
                case c == '"':
                        t, err := l.consumeString()
                        if err != nil {
                                return nil, err
                        }
                        l.tokens = append(l.tokens, t)
                case c == '<':
                        // Could be <, </, or /> (handled by parser context).
                        if l.peek(1) == '/' {
                                l.tokens = append(l.tokens, Token{Kind: TKLT, Value: "<", Pos: l.pos, Line: l.line})
                                l.pos++
                                l.tokens = append(l.tokens, Token{Kind: TKSlash, Value: "/", Pos: l.pos, Line: l.line})
                                l.pos++
                                l.inJSXTag = true // closing tags also end with >
                        } else {
                                l.tokens = append(l.tokens, Token{Kind: TKLT, Value: "<", Pos: l.pos, Line: l.line})
                                l.pos++
                                l.inJSXTag = true
                        }
                case c == '>' && l.inJSXTag:
                        l.tokens = append(l.tokens, Token{Kind: TKGT, Value: ">", Pos: l.pos, Line: l.line})
                        l.pos++
                        l.inJSXTag = false
                case c == '/' && l.peek(1) == '>' && l.inJSXTag:
                        l.tokens = append(l.tokens, Token{Kind: TKSlashClose, Value: "/>", Pos: l.pos, Line: l.line})
                        l.pos += 2
                        l.inJSXTag = false
                case c == '{':
                        // Always emit structural brace. The parser decides whether a brace
                        // is a block boundary or an expression by context, and uses token
                        // positions to extract raw expression text from the source.
                        l.tokens = append(l.tokens, Token{Kind: TKLBrace, Value: "{", Pos: l.pos, Line: l.line})
                        l.pos++
                case c == '}':
                        l.tokens = append(l.tokens, Token{Kind: TKRBrace, Value: "}", Pos: l.pos, Line: l.line})
                        l.pos++
                case c == '(':
                        l.tokens = append(l.tokens, Token{Kind: TKLParen, Value: "(", Pos: l.pos, Line: l.line})
                        l.pos++
                case c == ')':
                        l.tokens = append(l.tokens, Token{Kind: TKRParen, Value: ")", Pos: l.pos, Line: l.line})
                        l.pos++
                case c == '=' && l.peek(1) == '>':
                        l.tokens = append(l.tokens, Token{Kind: TKArrow, Value: "=>", Pos: l.pos, Line: l.line})
                        l.pos += 2
                case c == '=':
                        l.tokens = append(l.tokens, Token{Kind: TKEq, Value: "=", Pos: l.pos, Line: l.line})
                        l.pos++
                case c == ':':
                        l.tokens = append(l.tokens, Token{Kind: TKColon, Value: ":", Pos: l.pos, Line: l.line})
                        l.pos++
                case c == '.':
                        l.tokens = append(l.tokens, Token{Kind: TKDot, Value: ".", Pos: l.pos, Line: l.line})
                        l.pos++
                case c == ',':
                        l.tokens = append(l.tokens, Token{Kind: TKComma, Value: ",", Pos: l.pos, Line: l.line})
                        l.pos++
                case c == '/':
                        l.tokens = append(l.tokens, Token{Kind: TKSlash, Value: "/", Pos: l.pos, Line: l.line})
                        l.pos++
                case isPunct(c):
                        // Collect a run of operator/punctuation characters as a single
                        // "op" token. This is permissive — we don't try to validate Go
                        // expressions, we just capture them verbatim for the parser to
                        // pass through to the generated code.
                        start := l.pos
                        for l.pos < len(l.src) && isPunct(l.src[l.pos]) {
                                l.pos++
                        }
                        l.tokens = append(l.tokens, Token{Kind: TKIdent, Value: l.src[start:l.pos], Pos: start, Line: l.line})
                case isDigit(c):
                        t := l.consumeNumber()
                        l.tokens = append(l.tokens, t)
                case isIdentStart(c):
                        t := l.consumeIdent()
                        l.tokens = append(l.tokens, t)
                default:
                        return nil, fmt.Errorf("line %d: unexpected character %q", l.line, c)
                }
        }
        l.tokens = append(l.tokens, Token{Kind: TKEOF, Pos: l.pos, Line: l.line})
        return l.tokens, nil
}

func (l *Lexer) peek(off int) byte {
        if l.pos+off >= len(l.src) {
                return 0
        }
        return l.src[l.pos+off]
}

func (l *Lexer) skipWhitespaceAndComments() {
        for l.pos < len(l.src) {
                c := l.src[l.pos]
                if c == ' ' || c == '\t' || c == '\r' {
                        l.pos++
                } else if c == '\n' {
                        l.pos++
                        l.line++
                } else {
                        break
                }
        }
}

func (l *Lexer) consumeLineComment() {
        for l.pos < len(l.src) && l.src[l.pos] != '\n' {
                l.pos++
        }
}

func (l *Lexer) consumeBlockComment() {
        l.pos += 2 // /*
        for l.pos < len(l.src) {
                if l.src[l.pos] == '*' && l.peek(1) == '/' {
                        l.pos += 2
                        return
                }
                if l.src[l.pos] == '\n' {
                        l.line++
                }
                l.pos++
        }
}

func (l *Lexer) consumeString() (Token, error) {
        start := l.pos
        l.pos++ // "
        var sb strings.Builder
        for l.pos < len(l.src) {
                c := l.src[l.pos]
                if c == '\\' && l.pos+1 < len(l.src) {
                        sb.WriteByte(c)
                        sb.WriteByte(l.src[l.pos+1])
                        l.pos += 2
                        continue
                }
                if c == '"' {
                        l.pos++
                        return Token{Kind: TKString, Value: l.src[start:l.pos], Pos: start, Line: l.line}, nil
                }
                if c == '\n' {
                        l.line++
                }
                sb.WriteByte(c)
                l.pos++
        }
        return Token{}, fmt.Errorf("line %d: unterminated string", l.line)
}

// consumeBrace reads a `{ ... }` block. If the brace is in a context where it
// represents an expression (e.g. inside JSX text or as an attribute value),
// the caller will have set l.inJSXTag=false and we capture the inner text
// verbatim as a TKExpr. Otherwise we emit TKLBrace and let the parser handle
// the block structurally.
//
// Heuristic: a brace is an expression if it appears after `>` (we're in JSX
// text content) or after `=` (attribute value). We detect this by checking
// the previous token.
func (l *Lexer) consumeBrace() (string, bool) {
        // Find matching close brace, respecting nested braces, parens, and strings.
        start := l.pos
        l.pos++ // {
        depth := 1
        for l.pos < len(l.src) {
                c := l.src[l.pos]
                switch c {
                case '{':
                        depth++
                case '}':
                        depth--
                        if depth == 0 {
                                inner := l.src[start+1 : l.pos]
                                l.pos++
                                return inner, true
                        }
                case '"':
                        // Skip string literal.
                        l.pos++
                        for l.pos < len(l.src) && l.src[l.pos] != '"' {
                                if l.src[l.pos] == '\\' && l.pos+1 < len(l.src) {
                                        l.pos += 2
                                        continue
                                }
                                if l.src[l.pos] == '\n' {
                                        l.line++
                                }
                                l.pos++
                        }
                case '\n':
                        l.line++
                }
                l.pos++
        }
        return "", false
}

func (l *Lexer) consumeNumber() Token {
        start := l.pos
        for l.pos < len(l.src) && (isDigit(l.src[l.pos]) || l.src[l.pos] == '.') {
                l.pos++
        }
        return Token{Kind: TKNumber, Value: l.src[start:l.pos], Pos: start, Line: l.line}
}

func (l *Lexer) consumeIdent() Token {
        start := l.pos
        for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
                l.pos++
        }
        v := l.src[start:l.pos]
        kind := TKIdent
        switch v {
        case "component", "state", "render", "style", "import", "from":
                kind = TKKeyword
        }
        return Token{Kind: kind, Value: v, Pos: start, Line: l.line}
}

func isDigit(c byte) bool   { return c >= '0' && c <= '9' }
func isIdentStart(c byte) bool {
        return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}
func isIdentPart(c byte) bool {
        return isIdentStart(c) || isDigit(c) || c == '-'
}

// LexJSXText tokenises raw text content between JSX tags. Returns a slice of
// tokens mixing TKText and TKExpr (for {…} interpolations).
func LexJSXText(src string) []JSXChild {
        var out []JSXChild
        i := 0
        for i < len(src) {
                // Find next {
                j := strings.IndexByte(src[i:], '{')
                if j < 0 {
                        text := strings.TrimSpace(src[i:])
                        if text != "" {
                                out = append(out, JSXChild{Kind: "text", Text: text})
                        }
                        break
                }
                text := strings.TrimSpace(src[i : i+j])
                if text != "" {
                        out = append(out, JSXChild{Kind: "text", Text: text})
                }
                // Capture { ... } expression verbatim.
                k := i + j + 1
                depth := 1
                for k < len(src) && depth > 0 {
                        if src[k] == '{' {
                                depth++
                        } else if src[k] == '}' {
                                depth--
                                if depth == 0 {
                                        break
                                }
                        }
                        k++
                }
                expr := strings.TrimSpace(src[i+j+1 : k])
                out = append(out, JSXChild{Kind: "expr", Expr: expr})
                i = k + 1
        }
        return out
}

// trimWS trims ASCII whitespace from s.
func trimWS(s string) string { return strings.TrimFunc(s, unicode.IsSpace) }
