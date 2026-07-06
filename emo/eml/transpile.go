package eml

import (
	"fmt"
	"os"
	"path/filepath"
)

// TranspileFile loads a .em file (and its referenced .css file, if any),
// parses them, and returns the File AST with the CSS attached.
//
// This is the entry point used by the emo dev server and CLI.
func TranspileFile(path string) (*File, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	f, err := Parse(string(src), path)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if f.StyleRef != "" {
		cssPath := f.StyleRef
		if !filepath.IsAbs(cssPath) {
			cssPath = filepath.Join(filepath.Dir(path), cssPath)
		}
		cssSrc, err := os.ReadFile(cssPath)
		if err == nil {
			ss, err := ParseCSS(string(cssSrc))
			if err == nil {
				f.CSS = ss
			}
		}
		// Missing CSS file is non-fatal — emit a warning in dev server logs.
	}
	return f, nil
}

// TranspileToGo loads a .em file and returns its Go source representation.
// Used by `emo build` to produce standalone Go files.
func TranspileToGo(path string, packageName string) (string, error) {
	f, err := TranspileFile(path)
	if err != nil {
		return "", err
	}
	return GenerateGo(f, CodegenOptions{PackageName: packageName})
}
