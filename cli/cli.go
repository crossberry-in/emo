// Package cli implements the emo command-line interface.
//
// Commands:
//
//   emo init <name>        Create a new emo project
//   emo start               Start the dev server with live reload
//   emo build               Produce a standalone APK via full codegen
//   emo go                  Install & launch the emo Go preview app on a
//                           connected Android device/emulator
//   emo plugins             List registered plugins
package cli

import (
        "fmt"
        "os"

        "github.com/spf13/cobra"
)

// New returns the root command.
func New() *cobra.Command {
        root := &cobra.Command{
                Use:   "emo",
                Short: "emo — Expo-like Android development framework in Go",
                Long: `emo is an Expo-like framework for building Android apps in Go.

Write your UI as pure Go component functions using the emo DSL. Save a file
and emo instantly pushes the new view tree to the emo Go preview app on your
device or emulator — no Gradle rebuild, no app restart.

Quickstart:
  emo init myapp
  cd myapp
  emo start

In another terminal:
  emo go        # installs and launches emo Go preview app
`,
                SilenceUsage: true,
        }
        root.AddCommand(newInitCmd())
        root.AddCommand(newStartCmd())
        root.AddCommand(newBuildCmd())
        root.AddCommand(newGoCmd())
        root.AddCommand(newPluginsCmd())
        root.AddCommand(newTemplateCmd())
        root.AddCommand(newTemplatesCmd())
        root.AddCommand(newInstallCmd())
        root.AddCommand(newComponentsCmd())
        return root
}

// Run executes the CLI with the given args.
func Run() int {
        if err := New().Execute(); err != nil {
                fmt.Fprintln(os.Stderr, "error:", err)
                return 1
        }
        return 0
}
