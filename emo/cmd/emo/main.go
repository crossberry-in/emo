// Package main is the entry point for the emo CLI binary.
package main

import (
	"os"

	"github.com/emo-framework/emo/cli"
)

func main() {
	os.Exit(cli.Run())
}
