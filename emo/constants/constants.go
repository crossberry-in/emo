// Package constants provides app constants, inspired by expo-constants.
//
// Exposes app metadata (name, version, build number, package name) that
// can be read at runtime from .em files or Go DSL code.
//
//   constants.AppName()       // → "my-app"
//   constants.AppVersion()    // → "1.0.0"
//   constants.BuildVersion()  // → "1"
//   constants.PackageName()   // → "dev.emo.my_app"
//   constants.SdkVersion()    // → "0.2"
package constants

import (
	"os"
	"sync"
)

// Constants holds the app's constant values.
type Constants struct {
	AppName      string
	AppVersion   string
	BuildVersion string
	PackageName  string
	SdkVersion   string
}

var (
	mu  sync.RWMutex
	val = Constants{
		AppName:      "emo app",
		AppVersion:   "0.1.0",
		BuildVersion: "1",
		PackageName:  "dev.emo.app",
		SdkVersion:   "0.2",
	}
)

// Set updates the constants (called by the CLI on startup).
func Set(c Constants) {
	mu.Lock()
	val = c
	mu.Unlock()
}

// LoadFromEnv reads constants from environment variables.
func LoadFromEnv() {
	mu.Lock()
	defer mu.Unlock()
	if v := os.Getenv("EMO_APP_NAME"); v != "" {
		val.AppName = v
	}
	if v := os.Getenv("EMO_APP_VERSION"); v != "" {
		val.AppVersion = v
	}
	if v := os.Getenv("EMO_BUILD_VERSION"); v != "" {
		val.BuildVersion = v
	}
	if v := os.Getenv("EMO_PACKAGE_NAME"); v != "" {
		val.PackageName = v
	}
	if v := os.Getenv("EMO_SDK_VERSION"); v != "" {
		val.SdkVersion = v
	}
}

// AppName returns the app's display name.
func AppName() string {
	mu.RLock()
	defer mu.RUnlock()
	return val.AppName
}

// AppVersion returns the app's version string (e.g. "1.0.0").
func AppVersion() string {
	mu.RLock()
	defer mu.RUnlock()
	return val.AppVersion
}

// BuildVersion returns the app's build number (e.g. "1").
func BuildVersion() string {
	mu.RLock()
	defer mu.RUnlock()
	return val.BuildVersion
}

// PackageName returns the app's package name (e.g. "dev.emo.my_app").
func PackageName() string {
	mu.RLock()
	defer mu.RUnlock()
	return val.PackageName
}

// SdkVersion returns the emo SDK version (e.g. "0.2").
func SdkVersion() string {
	mu.RLock()
	defer mu.RUnlock()
	return val.SdkVersion
}

// All returns all constants as a map.
func All() map[string]string {
	mu.RLock()
	defer mu.RUnlock()
	return map[string]string{
		"appName":      val.AppName,
		"appVersion":   val.AppVersion,
		"buildVersion": val.BuildVersion,
		"packageName":  val.PackageName,
		"sdkVersion":   val.SdkVersion,
	}
}
