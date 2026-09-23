// Package eval contains Cobra adapters for Strategist evaluation utilities.
package eval

import "github.com/spf13/cobra"

// NewParent returns the "eval" parent Cobra command for Strategist evaluation utilities.
func NewParent() *cobra.Command {
	return &cobra.Command{Use: "eval", Short: "Strategist eval harness utilities"}
}

// Register composes the eval command family (run, harvest) under one parent
// and attaches it to root, mirroring the metrics/plugins/mission/install
// adapter packages' Register entrypoint.
func Register(root *cobra.Command, deps Dependencies, harvestDeps HarvestDependencies) {
	parent := NewParent()
	parent.AddCommand(NewRun(deps), NewHarvest(harvestDeps))
	root.AddCommand(parent)
}
