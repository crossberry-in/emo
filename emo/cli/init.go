package cli

import (
        "fmt"
        "os"
        "path/filepath"
        "text/template"

        "github.com/spf13/cobra"
)

const appEmTemplate = `// {{.Name}}.em — emo 0.1 SDK
// Edit this file and save — emo pushes the new view tree to your device
// instantly. No rebuild, no restart.

import { Header } from "./Header.em"

component App {
  state count = 0
  state name = "{{.Name}}"

  render {
    <Column className="container">
      <Text fontSize={28} fontWeight="bold">{name} counter</Text>
      <Text fontSize={18}>Count: {count}</Text>
      <Row className="buttonRow">
        <Button onClick={() => count = count - 1}>Decrement</Button>
        <Button onClick={() => count = count + 1}>Increment</Button>
      </Row>
      <Button onClick={() => count = 0}>Reset</Button>
      <Divider />
      <Text className="hint">Edit App.em and save for live reload!</Text>
    </Column>
  }
}

style "./App.css"
`

const appCssTemplate = `/* {{.Name}}.css — styles for App.em */
.container {
  background: #FFFFFF;
  padding: 24dp;
  spacing: 16dp;
}

.buttonRow {
  spacing: 8dp;
}

.hint {
  color: #888888;
  font-size: 14sp;
}
`

const headerEmTemplate = `// Header.em — example imported component
// Shows how to import and use components across .em files.

component Header {
  render {
    <Text fontSize={24} fontWeight="bold">emo app</Text>
  }
}
`

const configTemplate = `# emo.toml — project configuration (emo 0.1 SDK)
name = "{{.Name}}"
package = "dev.emo.{{.Slug}}"
version = "0.1.0"

[dev]
port = 7575
watch = "."

[build]
output = "build/app.apk"
kotlinPackage = "dev.emo.{{.Slug}}"

[plugins]
camera = true
location = true
storage = true
vibration = true
`

const gitignore = `# emo 0.1
/build/
/.emo/
*.apk
*.keystore
__emo_gen__/
`

func newInitCmd() *cobra.Command {
        var pkg string
        var template string
        c := &cobra.Command{
                Use:   "init [name]",
                Short: "Create a new emo project",
                Args:  cobra.ExactArgs(1),
                RunE: func(cmd *cobra.Command, args []string) error {
                        name := args[0]
                        if pkg == "" {
                                pkg = "dev.emo." + slug(name)
                        }
                        // If no template specified, use "default" (full project with
                        // app/, components/, hooks/, android/ — like create-expo-app).
                        if template == "" {
                                template = "default"
                        }
                        return templateInit(name, template)
                },
        }
        c.Flags().StringVar(&pkg, "package", "", "Kotlin package name (default dev.emo.<slug>)")
        c.Flags().StringVar(&template, "template", "", "Project template (default: 'default'. Run `emo templates` to list)")
        return c
}

func initProject(name, pkg string) error {
        dir := name
        if err := os.MkdirAll(dir, 0o755); err != nil {
                return fmt.Errorf("mkdir: %w", err)
        }

        data := map[string]string{"Name": name, "Slug": slug(name)}

        // App.em
        appFile := filepath.Join(dir, "App.em")
        if exists(appFile) {
                return fmt.Errorf("refusing to overwrite existing %s", appFile)
        }
        if err := writeTemplate(appFile, appEmTemplate, data); err != nil {
                return err
        }

        // App.css
        if err := writeTemplate(filepath.Join(dir, "App.css"), appCssTemplate, data); err != nil {
                return err
        }

        // Header.em (example imported component)
        if err := writeTemplate(filepath.Join(dir, "Header.em"), headerEmTemplate, data); err != nil {
                return err
        }

        // emo.toml
        if err := writeTemplate(filepath.Join(dir, "emo.toml"), configTemplate, data); err != nil {
                return err
        }

        // .gitignore
        if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignore), 0o644); err != nil {
                return err
        }

        fmt.Printf(`Created emo 0.1 project "%s".

Next steps:
  cd %s
  emo start

In another terminal, with an Android emulator running or a device connected via adb:
  emo go

The emo Go preview app will install and connect to your dev server.
Edit App.em and save — your UI updates instantly.
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
