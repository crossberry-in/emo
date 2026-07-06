// Package linking implements deep linking and URL handling, inspired by
// expo-linking.
//
// Provides functions for:
//   - Opening external URLs (browser, other apps)
//   - Creating deep links back to your app
//   - Parsing incoming deep links
//
//   linking.OpenURL("https://example.com")
//   linking.OpenURL("mailto:support@example.com")
//   linking.CreateURL("/profile/42")  // → "myapp://profile/42"
//   linking.Parse("myapp://profile/42")  // → {path: "/profile/42", scheme: "myapp"}
package linking

import (
        "fmt"
        "sync"

        "github.com/emo-framework/emo/plugin"
)

// ParsedURL represents a parsed deep link.
type ParsedURL struct {
        Scheme string // e.g. "myapp"
        Host   string // e.g. "profile"
        Path   string // e.g. "/42"
        Query  string // e.g. "ref=notification"
}

var (
        mu          sync.RWMutex
        appScheme   = "emo"
        listeners   []func(ParsedURL)
)

// SetScheme sets the app's URL scheme (e.g. "myapp").
// This is used by CreateURL() to build deep links.
func SetScheme(scheme string) {
        mu.Lock()
        appScheme = scheme
        mu.Unlock()
}

// Scheme returns the current app scheme.
func Scheme() string {
        mu.RLock()
        defer mu.RUnlock()
        return appScheme
}

// OpenURL opens an external URL.
// On Android, this calls the system's Intent.ACTION_VIEW.
func OpenURL(url string) error {
        var err error
        plugin.Invoke("linking", "openURL", map[string]any{
                "url": url,
        }, func(result any, e error) {
                if e != nil {
                        err = e
                }
        })
        return err
}

// OpenSettings opens the device's settings app.
func OpenSettings() error {
        var err error
        plugin.Invoke("linking", "openSettings", nil, func(result any, e error) {
                if e != nil {
                        err = e
                }
        })
        return err
}

// CreateURL builds a deep link for the given path.
// e.g. CreateURL("/profile/42") → "myapp://profile/42"
func CreateURL(path string) string {
        scheme := Scheme()
        return fmt.Sprintf("%s://%s", scheme, path)
}

// Parse parses a deep link URL into its components.
func Parse(url string) ParsedURL {
        scheme := ""
        rest := url

        // Try "scheme://" first
        if i := indexOf(rest, "://"); i >= 0 {
                scheme = rest[:i]
                rest = rest[i+3:]
        } else if i := indexOf(rest, ":"); i >= 0 {
                // Handle "mailto:..." style URLs
                scheme = rest[:i]
                rest = rest[i+1:]
                // Strip leading "//" if present
                if len(rest) >= 2 && rest[:2] == "//" {
                        rest = rest[2:]
                }
        }

        host := ""
        path := ""
        query := ""

        if i := indexOf(rest, "?"); i >= 0 {
                query = rest[i+1:]
                rest = rest[:i]
        }

        if i := indexOf(rest, "/"); i >= 0 {
                host = rest[:i]
                path = rest[i:]
        } else {
                host = rest
                path = ""
        }

        return ParsedURL{
                Scheme: scheme,
                Host:   host,
                Path:   path,
                Query:  query,
        }
}

// AddListener subscribes to incoming deep links.
// Returns an unsubscribe function.
func AddListener(fn func(ParsedURL)) func() {
        mu.Lock()
        listeners = append(listeners, fn)
        idx := len(listeners) - 1
        mu.Unlock()

        return func() {
                mu.Lock()
                defer mu.Unlock()
                if idx < len(listeners) {
                        listeners = append(listeners[:idx], listeners[idx+1:]...)
                }
        }
}

// HandleIncomingURL is called by the runtime when the app receives a deep
// link. It parses the URL and fires all registered listeners.
func HandleIncomingURL(url string) {
        parsed := Parse(url)
        mu.RLock()
        snapshot := make([]func(ParsedURL), len(listeners))
        copy(snapshot, listeners)
        mu.RUnlock()

        for _, l := range snapshot {
                l(parsed)
        }
}

func indexOf(s, sub string) int {
        for i := 0; i+len(sub) <= len(s); i++ {
                if s[i:i+len(sub)] == sub {
                        return i
                }
        }
        return -1
}
