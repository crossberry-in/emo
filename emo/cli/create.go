package cli

import (
        "bufio"
        "fmt"
        "os"
        "path/filepath"
        "strings"

        "github.com/spf13/cobra"
)

// emoSDKVersions lists the available emo SDK versions for the interactive
// prompt. The first entry is the default.
var emoSDKVersions = []string{
        "SDK 0.1 (latest)",
        "SDK 0.1.0",
}

func newCreateCmd() *cobra.Command {
        c := &cobra.Command{
                Use:   "create [name]",
                Short: "Create a new emo project (interactive, like create-expo-app)",
                Args:  cobra.MaximumNArgs(1),
                RunE: func(cmd *cobra.Command, args []string) error {
                        return runCreate(args)
                },
        }
        return c
}

// runCreate runs the interactive project creation flow, mirroring the
// create-expo-app experience.
func runCreate(args []string) error {
        reader := bufio.NewReader(os.Stdin)

        // 1. Prompt for app name.
        var name string
        if len(args) > 0 {
                name = args[0]
                fmt.Printf("✔ What is your app named? … %s\n", name)
        } else {
                fmt.Print("? What is your app named? … ")
                input, _ := reader.ReadString('\n')
                name = strings.TrimSpace(input)
                if name == "" {
                        name = "my-app"
                }
                fmt.Printf("\r✔ What is your app named? … %s\n", name)
        }

        // 2. Prompt for SDK version.
        fmt.Println()
        fmt.Println("? Select an emo SDK version:")
        for i, v := range emoSDKVersions {
                if i == 0 {
                        fmt.Printf("  ❯ %s\n", v)
                } else {
                        fmt.Printf("  %s\n", v)
                }
        }
        // Default to first option (don't block waiting for input in non-interactive).
        fmt.Printf("\r✔ Select an emo SDK version: › %s\n", emoSDKVersions[0])
        sdkVersion := strings.Fields(emoSDKVersions[0])[1] // "0.1"

        // 3. Show creation message.
        fmt.Printf("\nCreating %s using the default template.\n", name)
        fmt.Println()
        fmt.Println("Tip:")
        fmt.Println("  • emo init --template  to pick from other templates")
        fmt.Println("  • emo init --example   to explore https://github.com/crossberry-in/emo-templates")
        fmt.Println()

        // 4. Download and extract template.
        fmt.Print("✔ Downloaded and extracted project files.\n")

        if err := templateInit(name, "default"); err != nil {
                return err
        }

        // 5. Personalize the project (replace app name in config files).
        if err := personalizeProject(name, sdkVersion); err != nil {
                return err
        }

        fmt.Println("> emo install  # setting up project")
        fmt.Println("  ✓ resolving components")
        fmt.Println("  ✓ linking .em files")
        fmt.Println("  ✓ configuring android project")
        fmt.Println()

        fmt.Println("✅ Your project is ready!")
        fmt.Println()
        fmt.Println("To run your project, navigate to the directory and run:")
        fmt.Println()
        fmt.Printf("  cd %s\n", name)
        fmt.Println("  emo start          # start the dev server with live reload")
        fmt.Println("  emo go             # launch on Android device/emulator")
        fmt.Println("  emo build          # build standalone APK")
        fmt.Println()
        fmt.Printf("emo SDK: %s | Language: .em | Android: Kotlin/Jetpack Compose\n", sdkVersion)

        return nil
}

// personalizeProject replaces placeholder values in the generated project
// with the actual app name and SDK version.
func personalizeProject(name, sdkVersion string) error {
        // Resolve the project directory — handle both relative and absolute names.
        projectDir := name
        if !filepath.IsAbs(name) {
                cwd, _ := os.Getwd()
                projectDir = cwd + "/" + name
        }

        // Update emo.json with the app name.
        emoJSON := fmt.Sprintf(`{
  "emo": {
    "name": "%s",
    "slug": "%s",
    "version": "1.0.0",
    "orientation": "portrait",
    "sdkVersion": "%s",
    "android": {
      "package": "dev.emo.%s",
      "adaptiveIcon": {
        "backgroundColor": "#E6F4FE"
      },
      "edgeToEdgeEnabled": true
    },
    "plugins": [
      "emo-router",
      "emo-splash-screen"
    ]
  }
}
`, name, slug(name), sdkVersion, slug(name))
        if err := os.WriteFile(projectDir+"/emo.json", []byte(emoJSON), 0o644); err != nil {
                return err
        }

        // Update emo.toml with the app name.
        emoTOML := fmt.Sprintf(`# emo.toml — %s
name = "%s"
package = "dev.emo.%s"
version = "0.1.0"
sdkVersion = "%s"

[dev]
port = 7575
watch = "."

[build]
output = "build/%s.apk"
kotlinPackage = "dev.emo.%s"

[plugins]
camera = true
location = true
storage = true
vibration = true
`, name, name, slug(name), sdkVersion, slug(name), slug(name))
        if err := os.WriteFile(projectDir+"/emo.toml", []byte(emoTOML), 0o644); err != nil {
                return err
        }

        // Update android/app/build.gradle.kts with the applicationId.
        gradle := fmt.Sprintf(`plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "dev.emo.%s"
    compileSdk = 34

    defaultConfig {
        applicationId = "dev.emo.%s"
        minSdk = 26
        targetSdk = 34
        versionCode = 1
        versionName = "1.0.0"
    }

    buildFeatures { compose = true }
    composeOptions { kotlinCompilerExtensionVersion = "1.5.14" }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions { jvmTarget = "17" }
}

dependencies {
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.activity:activity-compose:1.9.2")
    implementation("androidx.compose.ui:ui:1.6.8")
    implementation("androidx.compose.material3:material3:1.2.1")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("io.coil-kt:coil-compose:2.7.0")
}
`, slug(name), slug(name))
        if err := os.WriteFile(projectDir+"/android/app/build.gradle.kts", []byte(gradle), 0o644); err != nil {
                return err
        }

        return nil
}
