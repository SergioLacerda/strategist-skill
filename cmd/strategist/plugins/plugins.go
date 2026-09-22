// Package plugins contains Cobra adapters for the Strategist plugins command
// family. Authorization, write enforcement and embedded-skill ingestion remain
// owned by internal/authorization, internal/plugins and internal/install.
package plugins

import "github.com/spf13/cobra"

// New creates the complete plugins command family.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "Inspect and evaluate Strategist plugin enforcement",
	}
	cmd.AddCommand(NewAuthorize(), NewEvaluateWrite(), NewPrepareEmbedded())
	return cmd
}

// Register attaches the complete plugins family at the supplied root.
func Register(root *cobra.Command) {
	root.AddCommand(New())
}
