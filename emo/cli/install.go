package cli

import (
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "os"
        "path/filepath"
        "strings"

        "github.com/spf13/cobra"
)

// ComponentsRepo is the GitHub repo that holds installable emo components.
const ComponentsRepo = "crossberry-in/emo-templates"
const ComponentsBranch = "main"

// GitHubToken is an optional auth token for GitHub API requests. Set via
// GITHUB_TOKEN env var to avoid rate limits on `emo install`.
var GitHubToken = os.Getenv("GITHUB_TOKEN")

// ghGet performs an authenticated GET request to the GitHub API if a token
// is available, otherwise an unauthenticated request.
func ghGet(url string) (*http.Response, error) {
        req, err := http.NewRequest("GET", url, nil)
        if err != nil {
                return nil, err
        }
        if GitHubToken != "" {
                req.Header.Set("Authorization", "token "+GitHubToken)
        }
        req.Header.Set("Accept", "application/vnd.github.v3+json")
        return http.DefaultClient.Do(req)
}

// Component describes an installable component from emo-templates repo.
type Component struct {
        Name        string `json:"name"`
        Description string `json:"description"`
        Path        string `json:"path"` // subdirectory in the repo, e.g. "components/Card"
        Files       int    `json:"files,omitempty"`
}

func newInstallCmd() *cobra.Command {
        c := &cobra.Command{
                Use:   "install [component]",
                Short: "Install a component from the emo component registry (like expo install)",
                Args:  cobra.ExactArgs(1),
                RunE: func(cmd *cobra.Command, args []string) error {
                        return installComponent(args[0])
                },
        }
        return c
}

func newComponentsCmd() *cobra.Command {
        c := &cobra.Command{
                Use:   "components",
                Short: "List installable components from the emo registry",
                RunE: func(cmd *cobra.Command, args []string) error {
                        return listComponents()
                },
        }
        return c
}

// listComponents fetches the component list from the emo-templates repo.
func listComponents() error {
        url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/components.json", ComponentsRepo, ComponentsBranch)
        resp, err := http.Get(url)
        if err != nil {
                return fmt.Errorf("fetch components: %w", err)
        }
        defer resp.Body.Close()
        if resp.StatusCode != 200 {
                return fmt.Errorf("components.json not found (HTTP %d)", resp.StatusCode)
        }
        var comps []Component
        if err := json.NewDecoder(resp.Body).Decode(&comps); err != nil {
                return fmt.Errorf("parse components.json: %w", err)
        }
        fmt.Printf("Installable emo components (%s):\n\n", ComponentsRepo)
        for _, c := range comps {
                fmt.Printf("  %-15s  %s\n", c.Name, c.Description)
        }
        fmt.Println("\nUsage: emo install <component>")
        return nil
}

// installComponent downloads a component from the registry and copies its
// .em (and .css) files into the project's components/ directory.
func installComponent(name string) error {
        fmt.Printf("Installing component %q from GitHub…\n", name)

        // Fetch components.json.
        url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/components.json", ComponentsRepo, ComponentsBranch)
        resp, err := http.Get(url)
        if err != nil {
                return err
        }
        defer resp.Body.Close()
        if resp.StatusCode != 200 {
                return fmt.Errorf("components.json not found")
        }
        var comps []Component
        if err := json.NewDecoder(resp.Body).Decode(&comps); err != nil {
                return err
        }

        var comp *Component
        for i := range comps {
                if comps[i].Name == name {
                        comp = &comps[i]
                        break
                }
        }
        if comp == nil {
                return fmt.Errorf("component %q not found — run `emo components` to see available components", name)
        }

        // List files in the component's directory via the GitHub Contents API.
        contentsURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s", ComponentsRepo, comp.Path, ComponentsBranch)
        cresp, err := ghGet(contentsURL)
        if err != nil {
                return err
        }
        defer cresp.Body.Close()
        if cresp.StatusCode != 200 {
                return fmt.Errorf("fetch component contents: HTTP %d (set GITHUB_TOKEN env var if rate-limited)", cresp.StatusCode)
        }

        var entries []struct {
                Name string `json:"name"`
                Path string `json:"path"`
                Type string `json:"type"` // "file" or "dir"
                URL  string `json:"download_url"`
        }
        if err := json.NewDecoder(cresp.Body).Decode(&entries); err != nil {
                return err
        }

        // Create components/ directory in the current project.
        cwd, _ := os.Getwd()
        compDir := filepath.Join(cwd, "components", strings.ToLower(name))
        if err := os.MkdirAll(compDir, 0o755); err != nil {
                return err
        }

        // Download each file.
        downloaded := 0
        for _, e := range entries {
                if e.Type != "file" {
                        continue
                }
                // Only download .em and .css files.
                ext := filepath.Ext(e.Name)
                if ext != ".em" && ext != ".css" {
                        continue
                }
                fmt.Printf("  downloading %s…\n", e.Name)
                fresp, err := http.Get(e.URL)
                if err != nil {
                        return err
                }
                if fresp.StatusCode != 200 {
                        fresp.Body.Close()
                        return fmt.Errorf("download %s: HTTP %d", e.Name, fresp.StatusCode)
                }
                target := filepath.Join(compDir, e.Name)
                f, err := os.Create(target)
                if err != nil {
                        fresp.Body.Close()
                        return err
                }
                if _, err := io.Copy(f, fresp.Body); err != nil {
                        f.Close()
                        fresp.Body.Close()
                        return err
                }
                f.Close()
                fresp.Body.Close()
                downloaded++
        }

        if downloaded == 0 {
                return fmt.Errorf("no .em or .css files found in component %q", name)
        }

        fmt.Printf("\n✓ Installed %s (%d files) into components/%s/\n", name, downloaded, strings.ToLower(name))
        fmt.Println("\nUsage in your .em file:")
        fmt.Printf("  import { %s } from \"./components/%s/%s.em\"\n", name, strings.ToLower(name), name)
        fmt.Println("\nThen use it in your render block:")
        fmt.Printf("  <%s />\n", name)
        return nil
}
