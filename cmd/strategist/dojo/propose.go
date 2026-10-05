package dojo

import (
	"fmt"
	"time"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/spf13/cobra"
)

func newPropose(deps Dependencies) *cobra.Command {
	var root, description, reason, by string
	cmd := &cobra.Command{
		Use:   "propose <operator>",
		Short: "Record an operator proposal pending a reviewed change",
		Long:  "Record a proposal for a new mutation operator under <base_path>/dojo/mutations. It adds nothing to the catalog: a proposal waits for a reviewed change.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, basePath, err := resolveRoots(deps, root)
			if err != nil {
				return err
			}
			proposal := internaldojo.Proposal{Name: args[0], Description: description, Reason: reason, By: by}
			if err := internaldojo.AppendProposal(basePath, proposal, time.Now()); err != nil {
				return fmt.Errorf("dojo propose: %w", err)
			}
			return say(cmd.OutOrStdout(), "proposal %s recorded as pending; it changes no catalog\n", args[0])
		},
	}
	f := cmd.Flags()
	f.StringVar(&root, "root", "", rootFlagUsage)
	f.StringVar(&description, "description", "", "what the operator injects (required)")
	f.StringVar(&reason, "reason", "", "which evasion or weak spot motivates it")
	f.StringVar(&by, "by", "agent", "who proposes it")
	_ = cmd.MarkFlagRequired("description") //nolint:errcheck // flag is defined above
	return cmd
}
