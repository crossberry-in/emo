package eml

import (
	"fmt"
	"strconv"
	"strings"
)

// Stylesheet is a parsed .css file: a map of className → properties.
type Stylesheet struct {
	Rules []CSSRule
}

// CSSRule is a single `.class { ... }` rule.
type CSSRule struct {
	Selector string            // e.g. ".container" — leading dot is kept
	Props    map[string]string // e.g. {"background": "#FFFFFF", "padding": "24dp"}
}

// LookupClass returns the property map for a className (without leading dot),
// or nil if not found.
func (s *Stylesheet) LookupClass(className string) map[string]string {
	if s == nil {
		return nil
	}
	sel := "." + className
	for _, r := range s.Rules {
		if r.Selector == sel {
			return r.Props
		}
	}
	return nil
}

// ParseCSS parses a simple CSS subset: `.class { prop: value; ... }`.
// Comments (/* … */) are skipped. Selectors are kept verbatim; only single
// class selectors are meaningful for emo (id, tag, and compound selectors
// are ignored at lookup time but stored in the AST for completeness).
func ParseCSS(src string) (*Stylesheet, error) {
	ss := &Stylesheet{}
	i := 0
	for i < len(src) {
		i = skipCSSWS(src, i)
		if i >= len(src) {
			break
		}
		// Skip /* … */ comments.
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '*' {
			end := strings.Index(src[i:], "*/")
			if end < 0 {
				return nil, fmt.Errorf("unterminated block comment in CSS")
			}
			i += end + 2
			continue
		}
		// Read selector up to {
		br := strings.IndexByte(src[i:], '{')
		if br < 0 {
			return nil, fmt.Errorf("expected '{' after selector")
		}
		selector := strings.TrimSpace(src[i : i+br])
		i += br + 1
		// Read declarations up to }
		end := strings.IndexByte(src[i:], '}')
		if end < 0 {
			return nil, fmt.Errorf("expected '}' after CSS rule body")
		}
		body := src[i : i+end]
		i += end + 1
		props := parseCSSDecls(body)
		ss.Rules = append(ss.Rules, CSSRule{Selector: selector, Props: props})
	}
	return ss, nil
}

func parseCSSDecls(body string) map[string]string {
	props := map[string]string{}
	for _, decl := range strings.Split(body, ";") {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		colon := strings.IndexByte(decl, ':')
		if colon < 0 {
			continue
		}
		k := strings.TrimSpace(decl[:colon])
		v := strings.TrimSpace(decl[colon+1:])
		props[k] = v
	}
	return props
}

func skipCSSWS(src string, i int) int {
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}
		break
	}
	return i
}

// CSSPropToEmo converts a CSS key/value pair into an emo DSL prop entry.
// Returns ("", "", false) if the CSS property has no emo equivalent.
//
// CSS property      → emo DSL prop key
// ------------------------------------
// background        → background
// color             → color
// padding           → padding (parsed as float, dp assumed)
// margin            → margin
// spacing           → spacing
// font-size         → fontSize
// font-weight       → fontWeight
// width             → width
// height            → height
//
// Units: "dp" and "sp" are stripped (the value is unitless in emo); "px" is
// treated as dp; pure numbers are passed through. Hex colors (#RRGGBB or
// #AARRGGBB) are passed through verbatim.
func CSSPropToEmo(key, value string) (emoKey string, emoVal any, ok bool) {
	switch key {
	case "background", "color", "fontWeight":
		return key, value, true
	case "padding", "margin", "spacing":
		return key, parseCSSDim(value), true
	case "font-size":
		return "fontSize", parseCSSDim(value), true
	case "width", "height":
		return key, parseCSSDimOrKeyword(value), true
	}
	return "", nil, false
}

// parseCSSDim converts a CSS dimension string like "24dp" or "16sp" or "12.5"
// to a float64. Unrecognised units fall through to plain number parse.
func parseCSSDim(v string) float64 {
	v = strings.TrimSpace(v)
	v = strings.TrimSuffix(v, "dp")
	v = strings.TrimSuffix(v, "sp")
	v = strings.TrimSuffix(v, "px")
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0
	}
	return f
}

// parseCSSDimOrKeyword is like parseCSSDim but also accepts "match" / "wrap"
// as keyword values for width/height.
func parseCSSDimOrKeyword(v string) any {
	v = strings.TrimSpace(v)
	if v == "match" || v == "wrap" {
		return v
	}
	return parseCSSDim(v)
}
