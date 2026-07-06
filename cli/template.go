package cli

import (
        "archive/tar"
        "compress/gzip"
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "os"
        "path/filepath"
        "strings"

        "github.com/spf13/cobra"
)

// TemplatesRepo is the GitHub repo that holds emo project templates.
const TemplatesRepo = "crossberry-in/emo-templates"
const TemplatesBranch = "main"

// Template describes a project template available in emo-templates repo.
type Template struct {
        Name        string `json:"name"`
        Description string `json:"description"`
        Path        string `json:"path"` // subdirectory in the repo
}

func newTemplateCmd() *cobra.Command {
        c := &cobra.Command{
                Use:   "template",
                Short: "Show template info",
                RunE: func(cmd *cobra.Command, args []string) error {
                        return listTemplates()
                },
        }
        return c
}

func newTemplatesCmd() *cobra.Command {
        c := &cobra.Command{
                Use:   "templates",
                Short: "List available project templates",
                RunE: func(cmd *cobra.Command, args []string) error {
                        return listTemplates()
                },
        }
        return c
}

// newTemplateInitCmd handles `emo init <name> --template <template>`
func templateInit(name, templateName string) error {
        fmt.Printf("Downloading template %q from GitHub…\n", templateName)
        tmp, err := downloadTemplate(templateName)
        if err != nil {
                return err
        }
        defer os.RemoveAll(tmp)

        // Copy template contents into the target directory.
        if err := os.MkdirAll(name, 0o755); err != nil {
                return err
        }
        return copyDir(tmp, name)
}

// listTemplates fetches the template list from the emo-templates repo.
func listTemplates() error {
        url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/templates.json", TemplatesRepo, TemplatesBranch)
        resp, err := http.Get(url)
        if err != nil {
                return fmt.Errorf("fetch templates: %w", err)
        }
        defer resp.Body.Close()
        if resp.StatusCode != 200 {
                return fmt.Errorf("templates.json not found (HTTP %d) — make sure the emo-templates repo is published", resp.StatusCode)
        }
        var templates []Template
        if err := json.NewDecoder(resp.Body).Decode(&templates); err != nil {
                return fmt.Errorf("parse templates.json: %w", err)
        }
        fmt.Printf("Available emo templates (%s):\n\n", TemplatesRepo)
        for _, t := range templates {
                fmt.Printf("  %-15s  %s\n", t.Name, t.Description)
        }
        fmt.Println("\nUsage: emo init myapp --template <name>")
        return nil
}

// downloadTemplate downloads a template subdirectory from the emo-templates
// repo as a tarball and extracts it to a temp directory.
func downloadTemplate(name string) (string, error) {
        // Fetch templates.json to find the path.
        url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/templates.json", TemplatesRepo, TemplatesBranch)
        resp, err := http.Get(url)
        if err != nil {
                return "", err
        }
        defer resp.Body.Close()
        if resp.StatusCode != 200 {
                return "", fmt.Errorf("templates.json not found")
        }
        var templates []Template
        if err := json.NewDecoder(resp.Body).Decode(&templates); err != nil {
                return "", err
        }

        var tmpl *Template
        for i := range templates {
                if templates[i].Name == name {
                        tmpl = &templates[i]
                        break
                }
        }
        if tmpl == nil {
                return "", fmt.Errorf("template %q not found — run `emo templates` to see available templates", name)
        }

        // Download the entire emo-templates repo as a tarball.
        tarURL := fmt.Sprintf("https://api.github.com/repos/%s/tarball/%s", TemplatesRepo, TemplatesBranch)
        fmt.Printf("  downloading %s…\n", tarURL)
        tresp, err := http.Get(tarURL)
        if err != nil {
                return "", err
        }
        defer tresp.Body.Close()
        if tresp.StatusCode != 200 {
                return "", fmt.Errorf("download tarball: HTTP %d", tresp.StatusCode)
        }

        tmpDir, err := os.MkdirTemp("", "emo-template-")
        if err != nil {
                return "", err
        }

        // Extract tarball.
        gz, err := gzip.NewReader(tresp.Body)
        if err != nil {
                return "", err
        }
        defer gz.Close()
        tr := tar.NewReader(gz)
        // GitHub tarballs have a top-level dir like "crossberry-in-emo-templates-abc123/".
        // We strip that prefix and look for tmpl.Path/ entries.
        topPrefix := ""
        extracted := 0
        for {
                hdr, err := tr.Next()
                if err == io.EOF {
                        break
                }
                if err != nil {
                        return "", err
                }
                if topPrefix == "" {
                        // First entry — extract the top-level dir name.
                        parts := strings.SplitN(hdr.Name, "/", 2)
                        if len(parts) < 2 {
                                continue
                        }
                        topPrefix = parts[0] + "/"
                }
                // Strip top prefix.
                rel := strings.TrimPrefix(hdr.Name, topPrefix)
                if rel == "" {
                        continue
                }
                // Only extract files under tmpl.Path.
                if !strings.HasPrefix(rel, tmpl.Path+"/") && rel != tmpl.Path {
                        continue
                }
                // Strip the template path prefix.
                rel = strings.TrimPrefix(rel, tmpl.Path+"/")
                if rel == tmpl.Path {
                        continue
                }
                target := filepath.Join(tmpDir, rel)
                switch hdr.Typeflag {
                case tar.TypeDir:
                        os.MkdirAll(target, 0o755)
                case tar.TypeReg:
                        os.MkdirAll(filepath.Dir(target), 0o755)
                        f, err := os.Create(target)
                        if err != nil {
                                return "", err
                        }
                        if _, err := io.Copy(f, tr); err != nil {
                                f.Close()
                                return "", err
                        }
                        f.Close()
                        extracted++
                }
        }
        if extracted == 0 {
                return "", fmt.Errorf("template %q is empty or path %q not found in repo", name, tmpl.Path)
        }
        fmt.Printf("  extracted %d files\n", extracted)
        return tmpDir, nil
}

// copyDir copies src into dst recursively.
func copyDir(src, dst string) error {
        return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
                if err != nil {
                        return err
                }
                rel, err := filepath.Rel(src, path)
                if err != nil {
                        return err
                }
                target := filepath.Join(dst, rel)
                if info.IsDir() {
                        return os.MkdirAll(target, 0o755)
                }
                data, err := os.ReadFile(path)
                if err != nil {
                        return err
                }
                return os.WriteFile(target, data, 0o644)
        })
}
