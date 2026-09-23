// Package leveling contains Cobra adapters for Strategist LEVELING commands.
package leveling

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internal "github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/spf13/cobra"
)

// Dependencies defines injected dependencies for leveling commands.
type Dependencies struct {
	LoadPolicy    func() (internal.Policy, string, error)
	WorkspaceRoot func() (string, error)
	LoadConfig    func(root string) (domain.LevelingConfig, string)
	LoadRegistry  func(root string) (domain.RoleRegistry, string)
	LedgerName    string
	RotateMax     int
	EmitRoleLevel func(ctx context.Context, missionID, run string, level internal.Level, reason string)
}

// NewParent creates the parent Cobra command for leveling.
func NewParent() *cobra.Command {
	return &cobra.Command{Use: "leveling", Short: "Suggest model and effort by role"}
}

// New creates the complete LEVELING command family. Policy, role registry,
// ledger, and telemetry authorities stay behind injected dependencies.
func New(deps Dependencies) *cobra.Command {
	cmd := NewParent()
	Attach(cmd, deps, NewLabel(deps, &LabelOptions{}))
	return cmd
}

// Register attaches the complete LEVELING family at the supplied root.
func Register(root *cobra.Command, deps Dependencies) {
	root.AddCommand(New(deps))
}

// Attach composes the complete LEVELING family onto an existing parent command.
func Attach(cmd *cobra.Command, deps Dependencies, label *cobra.Command) {
	cmd.AddCommand(NewValidate(deps), NewSuggest(deps), label)
}

// NewValidate creates the validate command for checking leveling policy.
func NewValidate(deps Dependencies) *cobra.Command {
	var expected string
	cmd := &cobra.Command{Use: "validate", Short: "Validate the customer LEVELING policy"}
	cmd.Flags().StringVar(&expected, "expected-digest", "", "fail if the normalized policy digest differs")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunValidate(cmd, deps, expected) }
	return cmd
}

// RunValidate executes the validate command logic.
func RunValidate(cmd *cobra.Command, deps Dependencies, expected string) error {
	policy, path, err := deps.LoadPolicy()
	if err != nil {
		return err
	}
	if expected != "" {
		if err := policy.VerifyDigest(expected); err != nil {
			return fmt.Errorf("leveling: verify digest: %w", err)
		}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "policy=%s version=%d digest=%s providers=%d\n", path, policy.Version, policy.Digest(), len(policy.Providers)); err != nil {
		return fmt.Errorf("leveling: write validation result: %w", err)
	}
	return nil
}
