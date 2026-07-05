package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

const projectTemplate = `package main

import (
	"github.com/emo-framework/emo/dsl"
)

// App is the root component of your emo app.
//
// Edit this function and save the file — emo will push the new view tree
// to your device instantly. Try changing the text, adding new elements,
// or wiring up new state with dsl.UseState.
func App() dsl.Element {
	count, setCount := dsl.UseStateInt(0)

	return dsl.Scaffold(
		dsl.Prop("title", "{{.Name}}"),
		dsl.Children(
			dsl.Column(
				dsl.Spacing(16),
				dsl.Padding(24),
				dsl.Children(
					dsl.Text("emo counter", dsl.Font(28, "bold")),
					dsl.Text(fmt.Sprintf("Count: %d", count), dsl.Font(18, "normal")),
					dsl.Button("Increment", dsl.OnClick(func() {
						setCount(count + 1)
					})),
					dsl.Button("Reset", dsl.OnClick(func() {
						setCount(0)
					})),
					dsl.Divider(),
					dsl.Text("Edit App.go and save to see live reload!", dsl.Fg("#666666")),
				),
			),
		),
	)
}

func main() {
	// In emo, main() is invoked only when running the dev server.
	// emo calls App() on every render — you don't need to do anything here.
	// For standalone builds, emo generates a Kotlin MainActivity that hosts
	// the same vtree.
	emoServe(App)
}

// emoServe is provided by the emo runtime; the dev server shim installs it
// at build time. We declare it here so ` + "`go build`" + ` succeeds in the editor.
var emoServe = func(root func() dsl.Element) {}
`

const configTemplate = `# emo.toml — project configuration
name = "{{.Name}}"
package = "dev.emo.{{.Slug}}"
version = "0.1.0"

[dev]
port = 7575
# Directory to watch for live reload (relative to project root).
watch = "."

[build]
# Output APK path for ` + "`emo build`" + `.
output = "build/app.apk"
# Kotlin package name for codegen.
kotlinPackage = "dev.emo.{{.Slug}}"

[plugins]
# Built-in plugins ship with emo. Add third-party plugins here as
# ` + "`name = \"github.com/foo/emo-plugin-bar\"`" + `
camera = true
location = true
storage = true
vibration = true
`

const gitignore = `# emo
/build/
/.emo/
*.apk
*.keystore
`

func newInitCmd() *cobra.Command {
	var pkg string
	c := &cobra.Command{
		Use:   "init [name]",
		Short: "Create a new emo project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if pkg == "" {
				pkg = "dev.emo." + slug(name)
			}
			return initProject(name, pkg)
		},
	}
	c.Flags().StringVar(&pkg, "package", "", "Kotlin package name (default dev.emo.<slug>)")
	return c
}

func initProject(name, pkg string) error {
	dir := name
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// App.go
	appFile := filepath.Join(dir, "App.go")
	if exists(appFile) {
		return fmt.Errorf("refusing to overwrite existing %s", appFile)
	}
	if err := writeTemplate(appFile, projectTemplate, map[string]string{
		"Name": name,
		"Slug": slug(name),
	}); err != nil {
		return err
	}

	// emo.toml
	if err := writeTemplate(filepath.Join(dir, "emo.toml"), configTemplate, map[string]string{
		"Name": name,
		"Slug": slug(name),
	}); err != nil {
		return err
	}

	// .gitignore
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		return err
	}

	fmt.Printf(`Created emo project "%s".

Next steps:
  cd %s
  emo start

In another terminal, with an Android emulator running or a device connected via adb:
  emo go

The emo Go preview app will install and connect to your dev server.
Edit App.go and save — your UI updates instantly.
`, name, name)
	return nil
}

func writeTemplate(path, tmpl string, data any) error {
	t, err := template.New("t").Parse(tmpl)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func slug(s string) string {
	out := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			out += string(toLower(r))
		} else {
			out += "_"
		}
	}
	if out == "" {
		out = "app"
	}
	return out
}

func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}
