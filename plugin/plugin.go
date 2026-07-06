// Package plugin implements emo's plugin system.
//
// Plugins extend emo apps with native device capabilities — camera, location,
// storage, notifications, etc. — exactly like Expo SDK modules. A plugin is a
// Go type that registers itself with the registry at init time and exposes
// methods callable from any component via dsl.UsePlugin.
//
// On Android, the emo Go preview app ships a Kotlin counterpart for each
// built-in plugin that actually performs the native call (e.g. launches
// CameraX). When a Go plugin method is invoked, the dev server forwards the
// call to the connected device over the WebSocket and awaits a result.
package plugin

import (
	"fmt"
	"sync"
)

// Plugin is the interface every emo plugin implements.
type Plugin interface {
	// Name returns the plugin's public name, e.g. "camera".
	Name() string
	// Methods returns the methods this plugin exposes.
	Methods() []Method
}

// Method describes a single callable method on a plugin.
type Method struct {
	Name   string                       // e.g. "takePhoto"
	Params []Param                      // ordered input parameters
	Return string                       // human-readable return type, for docs
	Invoke func(params map[string]any, reply func(result any, err error)) // call into the device
}

// Param describes a method parameter.
type Param struct {
	Name string
	Type string // "string" | "int" | "bool" | "float" | "object"
	Desc string
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

var (
	mu        sync.RWMutex
	plugins   = map[string]Plugin{}
	listeners = []func(Plugin){}
)

// Register adds a plugin to the registry. Called from plugin init().
func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()
	plugins[p.Name()] = p
	for _, l := range listeners {
		l(p)
	}
}

// Get returns the named plugin, or nil if unregistered.
func Get(name string) Plugin {
	mu.RLock()
	defer mu.RUnlock()
	return plugins[name]
}

// All returns all registered plugins.
func All() []Plugin {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Plugin, 0, len(plugins))
	for _, p := range plugins {
		out = append(out, p)
	}
	return out
}

// OnRegister subscribes to plugin registration events.
func OnRegister(fn func(Plugin)) {
	mu.Lock()
	defer mu.Unlock()
	listeners = append(listeners, fn)
}

// Invoke calls a method on a plugin. The call is forwarded to the connected
// device via the dev server's transport. reply is invoked asynchronously when
// the device responds.
func Invoke(pluginName, methodName string, params map[string]any, reply func(result any, err error)) error {
	p := Get(pluginName)
	if p == nil {
		return fmt.Errorf("plugin %q not registered", pluginName)
	}
	for _, m := range p.Methods() {
		if m.Name == methodName {
			m.Invoke(params, reply)
			return nil
		}
	}
	return fmt.Errorf("plugin %q has no method %q", pluginName, methodName)
}

// ---------------------------------------------------------------------------
// Built-in plugins. These ship with the emo Go preview app and have
// Kotlin counterparts that perform the actual native calls.
// ---------------------------------------------------------------------------

// CameraPlugin exposes camera access.
type CameraPlugin struct{}

func (CameraPlugin) Name() string { return "camera" }
func (CameraPlugin) Methods() []Method {
	return []Method{
		{
			Name: "takePhoto",
			Params: []Param{
				{Name: "quality", Type: "int", Desc: "JPEG quality 0-100"},
			},
			Return: "string (base64 JPEG)",
			Invoke: func(params map[string]any, reply func(any, error)) {
				forwardToDevice("camera", "takePhoto", params, reply)
			},
		},
		{
			Name:   "requestPermission",
			Params: nil,
			Return: "bool",
			Invoke: func(params map[string]any, reply func(any, error)) {
				forwardToDevice("camera", "requestPermission", params, reply)
			},
		},
	}
}

// LocationPlugin exposes GPS access.
type LocationPlugin struct{}

func (LocationPlugin) Name() string { return "location" }
func (LocationPlugin) Methods() []Method {
	return []Method{
		{
			Name:   "getCurrentPosition",
			Params: []Param{{Name: "highAccuracy", Type: "bool", Desc: "Use GPS instead of network"}},
			Return: "object {lat,lng,accuracy,ts}",
			Invoke: func(params map[string]any, reply func(any, error)) {
				forwardToDevice("location", "getCurrentPosition", params, reply)
			},
		},
		{
			Name:   "startWatch",
			Params: nil,
			Return: "stream of positions",
			Invoke: func(params map[string]any, reply func(any, error)) {
				forwardToDevice("location", "startWatch", params, reply)
			},
		},
	}
}

// StoragePlugin exposes a key-value store backed by SharedPreferences on
// Android.
type StoragePlugin struct{}

func (StoragePlugin) Name() string { return "storage" }
func (StoragePlugin) Methods() []Method {
	return []Method{
		{
			Name:   "get",
			Params: []Param{{Name: "key", Type: "string"}},
			Return: "string",
			Invoke: func(params map[string]any, reply func(any, error)) {
				forwardToDevice("storage", "get", params, reply)
			},
		},
		{
			Name:   "set",
			Params: []Param{{Name: "key", Type: "string"}, {Name: "value", Type: "string"}},
			Return: "void",
			Invoke: func(params map[string]any, reply func(any, error)) {
				forwardToDevice("storage", "set", params, reply)
			},
		},
		{
			Name:   "remove",
			Params: []Param{{Name: "key", Type: "string"}},
			Return: "void",
			Invoke: func(params map[string]any, reply func(any, error)) {
				forwardToDevice("storage", "remove", params, reply)
			},
		},
	}
}

// VibrationPlugin exposes the vibrator.
type VibrationPlugin struct{}

func (VibrationPlugin) Name() string { return "vibration" }
func (VibrationPlugin) Methods() []Method {
	return []Method{
		{
			Name:   "vibrate",
			Params: []Param{{Name: "ms", Type: "int", Desc: "Duration in milliseconds"}},
			Return: "void",
			Invoke: func(params map[string]any, reply func(any, error)) {
				forwardToDevice("vibration", "vibrate", params, reply)
			},
		},
	}
}

// forwardToDevice is the transport hook. The dev server installs a real
// implementation; the default just errors out so static builds fail loudly.
var forwardToDevice = func(plugin, method string, params map[string]any, reply func(any, error)) {
	reply(nil, fmt.Errorf("no device transport installed; running in static mode"))
}

// SetTransport installs the device-call transport. Called by the dev server.
func SetTransport(fn func(plugin, method string, params map[string]any, reply func(any, error))) {
	forwardToDevice = fn
}

// init registers built-in plugins on package import.
func init() {
	Register(CameraPlugin{})
	Register(LocationPlugin{})
	Register(StoragePlugin{})
	Register(VibrationPlugin{})
}
