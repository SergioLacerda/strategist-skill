// Package eval contains Cobra adapters for Strategist evaluation utilities.
package eval

import "github.com/spf13/cobra"

// NewParent returns the "eval" parent Cobra command for Strategist evaluation utilities.
func NewParent() *cobra.Command {
	return &cobra.Command{Use: "eval", Short: "Strategist eval harness utilities"}
}
