// Package main is the thin executable launcher for the workflow builder.
// Handwritten CLI behavior lives in internal/generatecli so it can survive
// a later generated-command migration without edits to generated sources.
package main

import "github.com/arran4/arrans_overlay_workflow_builder/internal/generatecli"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	generatecli.Run(version, commit, date)
}
