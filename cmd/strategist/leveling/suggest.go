package leveling

import (
	"encoding/json"
	"fmt"

	internal "github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/spf13/cobra"
)

// SuggestOptions holds options for suggesting model and effort for a role.
type SuggestOptions struct {
	Provider, Role, Ambiguity, Risk, Scope, Evidence            string
	ArchitecturalChange, SecuritySensitive, ConflictingEvidence bool
	RepeatedFailures                                            int
	JSON                                                        bool
}

// NewSuggest creates a new suggest Cobra command.
func NewSuggest(deps Dependencies) *cobra.Command {
	opts := SuggestOptions{}
	cmd := &cobra.Command{Use: "suggest", Short: "Suggest a model and effort for a role"}
	f := cmd.Flags()
	f.StringVar(&opts.Provider, "provider", "CODEX", "ranked provider identifier")
	f.StringVar(&opts.Role, "role", "ranger", "Strategist role")
	f.StringVar(&opts.Ambiguity, "ambiguity", "", "ambiguity signal")
	f.StringVar(&opts.Risk, "risk", "", "risk signal")
	f.StringVar(&opts.Scope, "scope", "", "scope signal")
	f.StringVar(&opts.Evidence, "evidence", "", "evidence signal")
	f.BoolVar(&opts.ArchitecturalChange, "architectural-change", false, "mark an architectural change")
	f.BoolVar(&opts.SecuritySensitive, "security-sensitive", false, "mark a security-sensitive task")
	f.BoolVar(&opts.ConflictingEvidence, "conflicting-evidence", false, "mark conflicting evidence")
	f.IntVar(&opts.RepeatedFailures, "repeated-failures", 0, "number of repeated failures")
	f.BoolVar(&opts.JSON, "json", false, "emit JSON")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunSuggest(cmd, deps, opts) }
	return cmd
}

// RunSuggest executes the suggest command logic.
func RunSuggest(cmd *cobra.Command, deps Dependencies, opts SuggestOptions) error {
	policy, _, err := deps.LoadPolicy()
	if err != nil {
		return err
	}
	suggestion, err := internal.Suggest(policy, opts.Provider, opts.Role, internal.Signals{Ambiguity: opts.Ambiguity, Risk: opts.Risk, Scope: opts.Scope, Evidence: opts.Evidence, ArchitecturalChange: opts.ArchitecturalChange, SecuritySensitive: opts.SecuritySensitive, ConflictingEvidence: opts.ConflictingEvidence, RepeatedFailures: opts.RepeatedFailures})
	if err != nil {
		return fmt.Errorf("leveling: suggest: %w", err)
	}
	if opts.JSON {
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(suggestion); err != nil {
			return fmt.Errorf("leveling: write suggestion: %w", err)
		}
		return nil
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "provider=%s role=%s model=%s capability=%s effort=%s fallback_used=%t policy_digest=%s\n", suggestion.Provider, suggestion.Role, suggestion.Model, suggestion.Capability, suggestion.Effort, suggestion.FallbackUsed, suggestion.PolicyDigest); err != nil {
		return fmt.Errorf("leveling: write suggestion: %w", err)
	}
	return nil
}
