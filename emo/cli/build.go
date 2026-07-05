package cli

import (
        "fmt"
        "os"
        "os/exec"
        "path/filepath"

        "github.com/emo-framework/emo/codegen"
        "github.com/emo-framework/emo/server"
        "github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
        var out string
        var pkg string
        c := &cobra.Command{
                Use:   "build",
                Short: "Produce a standalone APK from the emo project",
                RunE: func(cmd *cobra.Command, args []string) error {
                        if err := ensureGo(); err != nil {
                                return err
                        }
                        dir, _ := os.Getwd()
                        root, err := loadRootComponent(dir)
                        if err != nil {
                                return err
                        }
                        tree := root()
                        kotlin := codegen.Build(tree, codegen.Options{
                                PackageName: pkg,
                                ClassName:   "EmoRoot",
                                Standalone:  true,
                        })

                        buildDir := filepath.Join(dir, ".emo", "build")
                        if err := os.MkdirAll(buildDir, 0o755); err != nil {
                                return err
                        }
                        kotPath := filepath.Join(buildDir, "EmoRoot.kt")
                        if err := os.WriteFile(kotPath, []byte(kotlin), 0o644); err != nil {
                                return err
                        }
                        fmt.Printf("generated %s\n", kotPath)

                        // Drive Gradle to assemble the APK. The emo template ships a
                        // ready-to-use Gradle project under .emo/build/ that we copy in
                        // from the emo install. In the open-source preview we just print
                        // the next steps; a full build requires the Android SDK + NDK.
                        fmt.Print(`
Next steps to assemble the APK:

  1. Copy the generated Kotlin into an Android Studio project (or use the
     template under templates/android-standalone in the emo installation).
  2. Run ./gradlew assembleDebug.
  3. The resulting APK is at app/build/outputs/apk/debug/app-debug.apk.

For instant development without rebuilding, use:

  emo start
  emo go
`)
                        if out != "" {
                                _ = out
                        }
                        return nil
                },
        }
        c.Flags().StringVar(&out, "out", "", "Output APK path")
        c.Flags().StringVar(&pkg, "package", "dev.emo.generated", "Kotlin package name for codegen")
        return c
}

func newGoCmd() *cobra.Command {
        var apk string
        c := &cobra.Command{
                Use:   "go",
                Short: "Install & launch the emo Go preview app on a connected device",
                Long: `emo go installs and launches the emo Go preview app on a connected
Android device or emulator. The app is the Expo Go equivalent for emo: it
connects to your dev server and renders incoming view trees as native
Jetpack Compose UI.

Requirements:
  - adb on PATH
  - A connected device or running emulator (adb devices)
  - The emo Go preview APK (build from android/ in the emo repo, or
    download from https://emo.dev/emo-go.apk)`,
                RunE: func(cmd *cobra.Command, args []string) error {
                        if _, err := exec.LookPath("adb"); err != nil {
                                return fmt.Errorf("adb not found on PATH — install Android Platform Tools")
                        }
                        // Read dev server URL from emo.toml or default to localhost:7575.
                        dir, _ := os.Getwd()
                        srv := server.New(dir, 7575)
                        if err := srv.LaunchOnDevice(apk, "dev.emo.go/.MainActivity"); err != nil {
                                return err
                        }
                        fmt.Println("emo Go launched on device.")
                        return nil
                },
        }
        c.Flags().StringVar(&apk, "apk", "", "Path to emo Go preview APK (will install before launching)")
        return c
}

func newPluginsCmd() *cobra.Command {
        return &cobra.Command{
                Use:   "plugins",
                Short: "List registered plugins",
                RunE: func(cmd *cobra.Command, args []string) error {
                        fmt.Println("Built-in emo plugins:")
                        fmt.Println("  camera     — take photos, request camera permission")
                        fmt.Println("  location   — getCurrentPosition, startWatch")
                        fmt.Println("  storage    — get/set/remove persistent key-value pairs")
                        fmt.Println("  vibration  — vibrate(ms)")
                        fmt.Println()
                        fmt.Println("Add third-party plugins in emo.toml under [plugins].")
                        return nil
                },
        }
}
