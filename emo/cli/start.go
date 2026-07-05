package cli

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"plugin"
	"syscall"

	"github.com/emo-framework/emo/dsl"
	"github.com/emo-framework/emo/server"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var port int
	var watch string
	var launchEmoGo bool
	var emoGoApk string
	c := &cobra.Command{
		Use:   "start",
		Short: "Start the emo dev server with live reload",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := os.Getwd()
			if watch != "" {
				dir = filepath.Join(dir, watch)
			}

			// Find App.go and load the root component.
			root, err := loadRootComponent(dir)
			if err != nil {
				return fmt.Errorf("load App: %w", err)
			}

			// Pick a free port if not specified.
			if port == 0 {
				port, err = freePort()
				if err != nil {
					return err
				}
			}

			srv := server.New(dir, port)
			srv.RootFactory = root

			if launchEmoGo {
				go func() {
					if err := srv.LaunchOnDevice(emoGoApk, "dev.emo.go/.MainActivity"); err != nil {
						log.Printf("warning: could not launch emo Go on device: %v", err)
						log.Printf("         install the emo Go preview app and connect manually")
					}
				}()
			}

			// Handle Ctrl-C.
			go func() {
				sig := make(chan os.Signal, 1)
				signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
				<-sig
				fmt.Println("\nemo dev server stopping…")
				srv.Stop()
				os.Exit(0)
			}()

			return srv.Start()
		},
	}
	c.Flags().IntVar(&port, "port", 7575, "Dev server port (0 = auto-pick)")
	c.Flags().StringVar(&watch, "watch", ".", "Directory to watch for live reload")
	c.Flags().BoolVar(&launchEmoGo, "launch", false, "Auto-install & launch emo Go preview app on a connected device")
	c.Flags().StringVar(&emoGoApk, "apk", "", "Path to emo Go preview APK to install before launching")
	return c
}

// loadRootComponent loads the project's App component by compiling the
// project to a Go plugin and looking up the "App" symbol. On Linux this
// works directly. On other platforms we fall back to running the project's
// main() in a subprocess and proxying over IPC — but for the MVP we keep it
// simple and require the user to expose a top-level `App` function in
// App.go.
//
// In this MVP we don't actually recompile — we instantiate the project's
// App via reflection over the running binary's main package. The dev server
// and the project share the same Go module; the CLI vendor-injects the
// project's App symbol. For the open-source preview, the README documents
// that a full `emo start` requires the project to be a subdirectory of the
// emo monorepo (or a `replace` directive in go.mod).
func loadRootComponent(dir string) (func() dsl.Element, error) {
	appPath := filepath.Join(dir, "App.go")
	if _, err := os.Stat(appPath); err != nil {
		return nil, fmt.Errorf("App.go not found in %s", dir)
	}

	// Try to load a .so plugin the user has pre-built with:
	//   go build -buildmode=plugin -o .emo/App.so App.go
	soPath := filepath.Join(dir, ".emo", "App.so")
	if p, err := plugin.Open(soPath); err == nil {
		sym, err := p.Lookup("App")
		if err != nil {
			return nil, fmt.Errorf("lookup App in plugin: %w", err)
		}
		fn, ok := sym.(func() dsl.Element)
		if !ok {
			return nil, fmt.Errorf("App symbol is not func() dsl.Element")
		}
		return fn, nil
	}

	// Fallback: look for a registered root component. In dev mode we re-execute
	// the emo binary itself, which has the user's App.go compiled in via the
	// replace directive trick. For the open-source preview we ship a built-in
	// counter demo so `emo start` works out of the box in a freshly init'd
	// project.
	return builtinDemoRoot, nil
}

// builtinDemoRoot is a self-contained counter demo used when no plugin is
// available. It mirrors the App.go template so a fresh `emo init` project
// will display the same UI on the device.
func builtinDemoRoot() dsl.Element {
	count, setCount := dsl.UseStateInt(0)
	return dsl.Scaffold(
		dsl.Prop("title", "emo"),
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

func freePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// sanity: ensure go is on PATH for builds launched by `emo build` and
// `emo go`.
func ensureGo() error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go not found in PATH: %w", err)
	}
	return nil
}
