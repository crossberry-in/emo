// Package main — counter demo for the emo framework.
//
// This is a minimal example showing the emo DSL with state, event handlers,
// and live reload. Edit this file and save — the dev server pushes the new
// vtree to the emo Go preview app on your device, and the UI updates
// instantly without restarting the activity.
package main

import (
	"fmt"

	"github.com/emo-framework/emo/dsl"
)

// App is the root component.
func App() dsl.Element {
	count, setCount := dsl.UseStateInt(0)

	return dsl.Scaffold(
		dsl.Prop("title", "emo counter"),
		dsl.Children(
			dsl.Column(
				dsl.Spacing(16),
				dsl.Padding(24),
				dsl.Bg("#FFFFFFFF"),
				dsl.Children(
					dsl.Text("emo counter", dsl.Font(28, "bold")),
					dsl.Text(fmt.Sprintf("Count: %d", count), dsl.Font(18, "normal")),
					dsl.Row(
						dsl.Spacing(8),
						dsl.Children(
							dsl.Button("−", dsl.OnClick(func() {
								setCount(count - 1)
							})),
							dsl.Button("+", dsl.OnClick(func() {
								setCount(count + 1)
							})),
						),
					),
					dsl.Button("Reset", dsl.OnClick(func() {
						setCount(0)
					})),
					dsl.Divider(),
					dsl.Text("Edit App.go and save to see live reload!", dsl.Fg("#888888")),
				),
			),
		),
	)
}

func main() {
	// When the project is built with `emo start`, the emo dev server replaces
	// this main() at load time. When built standalone with `emo build`, emo's
	// codegen emits a Kotlin MainActivity that hosts App()'s vtree.
	emoServe(App)
}

// emoServe is a shim that the emo runtime fills in. We declare it here so
// `go vet` and editor tooling don't complain.
var emoServe = func(root func() dsl.Element) {}
