// Package nav implements emo's navigation API, inspired by expo-router.
//
// Provides programmatic navigation between screens in emo apps:
//
//   nav.Navigate("/profile")
//   nav.Navigate("/user/42")
//   nav.Back()
//   nav.Replace("/login")
//   nav.Reset("/home")
//   nav.Param("id")  // → "42"
//
// The navigation state is tracked server-side (in dev mode) and pushed to
// the device as part of the vtree. Routes are defined in the app/ directory
// using file-based conventions:
//
//   app/index.em          →  "/"
//   app/profile.em        →  "/profile"
//   app/user/[id].em      →  "/user/:id"
//   app/(tabs)/home.em    →  "/home" (group prefix ignored)
package nav

import (
	"fmt"
	"sync"
)

// Route represents a single route in the app's navigation stack.
type Route struct {
	Path   string            // e.g. "/user/42"
	Screen string            // e.g. "user/[id]"
	Params map[string]string // e.g. {"id": "42"}
}

// NavigationState tracks the current navigation stack and route params.
type NavigationState struct {
	mu        sync.RWMutex
	stack     []Route       // navigation stack (last = current)
	listeners []func(Route) // callbacks fired on navigation
}

var state = &NavigationState{
	stack: []Route{{Path: "/", Screen: "index", Params: map[string]string{}}},
}

// Current returns the current route.
func Current() Route {
	state.mu.RLock()
	defer state.mu.RUnlock()
	if len(state.stack) == 0 {
		return Route{}
	}
	return state.stack[len(state.stack)-1]
}

// Navigate pushes a new route onto the stack.
// The path can include params: "/user/42" matches "user/[id]".
func Navigate(path string) {
	state.mu.Lock()
	route := matchRoute(path)
	state.stack = append(state.stack, route)
	listeners := make([]func(Route), len(state.listeners))
	copy(listeners, state.listeners)
	state.mu.Unlock()

	for _, l := range listeners {
		l(route)
	}
}

// Back pops the current route from the stack, returning to the previous.
// If the stack has only one route, Back is a no-op.
func Back() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.stack) <= 1 {
		return false
	}
	state.stack = state.stack[:len(state.stack)-1]
	route := state.stack[len(state.stack)-1]
	for _, l := range state.listeners {
		l(route)
	}
	return true
}

// Replace replaces the current route with a new one (no back navigation).
func Replace(path string) {
	state.mu.Lock()
	route := matchRoute(path)
	if len(state.stack) > 0 {
		state.stack[len(state.stack)-1] = route
	} else {
		state.stack = []Route{route}
	}
	listeners := make([]func(Route), len(state.listeners))
	copy(listeners, state.listeners)
	state.mu.Unlock()

	for _, l := range listeners {
		l(route)
	}
}

// Reset clears the stack and starts fresh from the given path.
func Reset(path string) {
	state.mu.Lock()
	route := matchRoute(path)
	state.stack = []Route{route}
	listeners := make([]func(Route), len(state.listeners))
	copy(listeners, state.listeners)
	state.mu.Unlock()

	for _, l := range listeners {
		l(route)
	}
}

// CanGoBack returns true if there's a previous route to go back to.
func CanGoBack() bool {
	state.mu.RLock()
	defer state.mu.RUnlock()
	return len(state.stack) > 1
}

// Stack returns a copy of the current navigation stack.
func Stack() []Route {
	state.mu.RLock()
	defer state.mu.RUnlock()
	out := make([]Route, len(state.stack))
	copy(out, state.stack)
	return out
}

// OnNavigate subscribes to navigation changes. Returns an unsubscribe function.
func OnNavigate(fn func(Route)) func() {
	state.mu.Lock()
	state.listeners = append(state.listeners, fn)
	idx := len(state.listeners) - 1
	state.mu.Unlock()

	return func() {
		state.mu.Lock()
		defer state.mu.Unlock()
		if idx < len(state.listeners) {
			state.listeners = append(state.listeners[:idx], state.listeners[idx+1:]...)
		}
	}
}

// Param returns the value of a route parameter from the current route.
// e.g. for "/user/42" matching "user/[id]", Param("id") returns "42".
func Param(name string) string {
	state.mu.RLock()
	defer state.mu.RUnlock()
	if len(state.stack) == 0 {
		return ""
	}
	return state.stack[len(state.stack)-1].Params[name]
}

// Params returns all params from the current route.
func Params() map[string]string {
	state.mu.RLock()
	defer state.mu.RUnlock()
	if len(state.stack) == 0 {
		return nil
	}
	out := make(map[string]string)
	for k, v := range state.stack[len(state.stack)-1].Params {
		out[k] = v
	}
	return out
}

// routeRegistry holds all registered routes from the app/ directory.
var (
	routesMu       sync.RWMutex
	routesRegistry = map[string]string{} // screen name → pattern
)

// RegisterRoute registers a route pattern.
// e.g. RegisterRoute("user/[id]", "user/:id")
func RegisterRoute(screen, pattern string) {
	routesMu.Lock()
	defer routesMu.Unlock()
	routesRegistry[screen] = pattern
}

// matchRoute finds the route matching the given path and extracts params.
func matchRoute(path string) Route {
	routesMu.RLock()
	defer routesMu.RUnlock()

	// Try exact match first (e.g. "/profile" → "profile")
	screen := path
	if screen == "/" {
		screen = "index"
	}

	// Try to match dynamic routes (e.g. "/user/42" → "user/[id]")
	for screenName, pattern := range routesRegistry {
		if params := matchPattern(pattern, path); params != nil {
			return Route{
				Path:   path,
				Screen: screenName,
				Params: params,
			}
		}
	}

	return Route{
		Path:   path,
		Screen: screen,
		Params: map[string]string{},
	}
}

// matchPattern matches a path against a pattern like "user/:id".
// Returns the extracted params, or nil if no match.
func matchPattern(pattern, path string) map[string]string {
	pSegs := splitPath(pattern)
	aSegs := splitPath(path)
	if len(pSegs) != len(aSegs) {
		return nil
	}
	params := map[string]string{}
	for i := range pSegs {
		if len(pSegs[i]) > 0 && pSegs[i][0] == ':' {
			params[pSegs[i][1:]] = aSegs[i]
		} else if pSegs[i] != aSegs[i] {
			return nil
		}
	}
	return params
}

func splitPath(p string) []string {
	var segs []string
	cur := ""
	for _, c := range p {
		if c == '/' {
			if cur != "" {
				segs = append(segs, cur)
				cur = ""
			}
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		segs = append(segs, cur)
	}
	return segs
}

// String returns a human-readable representation of the route.
func (r Route) String() string {
	return fmt.Sprintf("Route{Path: %q, Screen: %q, Params: %v}", r.Path, r.Screen, r.Params)
}
