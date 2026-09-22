package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/spf13/cobra"
)

var levelingCmd = &cobra.Command{
	Use:   "leveling",
	Short: "Suggest model and effort by role",
}

var levelingValidateCmd = newLevelingValidateCommand()

func newLevelingValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the customer LEVELING policy",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runLevelingValidate(cmd) },
	}
	cmd.Flags().String("expected-digest", "", "fail if the normalized policy digest differs")
	return cmd
}

func runLevelingValidate(cmd *cobra.Command) error {
	policy, path, err := loadLevelingPolicy()
	if err != nil {
		return err
	}
	if expectedDigest := stringFlag(cmd, "expected-digest", ""); expectedDigest != "" {
		if err := policy.VerifyDigest(expectedDigest); err != nil {
			return fmt.Errorf("leveling: verify digest: %w", err)
		}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "policy=%s version=%d digest=%s providers=%d\n", path, policy.Version, policy.Digest(), len(policy.Providers)); err != nil {
		return fmt.Errorf("leveling: write validation result: %w", err)
	}
	return nil
}

type levelingSuggestOptions struct {
	Provider, Role, Ambiguity, Risk, Scope, Evidence            string
	ArchitecturalChange, SecuritySensitive, ConflictingEvidence bool
	RepeatedFailures                                            int
	JSON                                                        bool
}

var levelingSuggestCmd = newLevelingSuggestCommand()

func newLevelingSuggestCommand() *cobra.Command {
	opts := levelingSuggestOptions{}
	cmd := &cobra.Command{
		Use:   "suggest",
		Short: "Suggest a model and effort for a role",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runLevelingSuggest(cmd, opts) },
	}
	configureLevelingSuggestFlags(cmd, &opts)
	return cmd
}

func runLevelingSuggest(cmd *cobra.Command, opts levelingSuggestOptions) error {
	policy, _, err := loadLevelingPolicy()
	if err != nil {
		return err
	}
	suggestion, err := leveling.Suggest(policy, opts.Provider, opts.Role, leveling.Signals{
		Ambiguity: opts.Ambiguity, Risk: opts.Risk, Scope: opts.Scope, Evidence: opts.Evidence,
		ArchitecturalChange: opts.ArchitecturalChange, SecuritySensitive: opts.SecuritySensitive,
		ConflictingEvidence: opts.ConflictingEvidence, RepeatedFailures: opts.RepeatedFailures,
	})
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

func configureLevelingSuggestFlags(cmd *cobra.Command, opts *levelingSuggestOptions) {
	cmd.Flags().StringVar(&opts.Provider, "provider", "CODEX", "ranked provider identifier")
	cmd.Flags().StringVar(&opts.Role, "role", "ranger", "Strategist role")
	cmd.Flags().StringVar(&opts.Ambiguity, "ambiguity", "", "ambiguity signal: low, medium, or high")
	cmd.Flags().StringVar(&opts.Risk, "risk", "", "risk signal: low, medium, or high")
	cmd.Flags().StringVar(&opts.Scope, "scope", "", "scope signal: bounded or cross_module")
	cmd.Flags().StringVar(&opts.Evidence, "evidence", "", "evidence signal: sufficient, insufficient, or conflicting")
	cmd.Flags().BoolVar(&opts.ArchitecturalChange, "architectural-change", false, "mark an architectural change")
	cmd.Flags().BoolVar(&opts.SecuritySensitive, "security-sensitive", false, "mark a security-sensitive task")
	cmd.Flags().BoolVar(&opts.ConflictingEvidence, "conflicting-evidence", false, "mark conflicting evidence")
	cmd.Flags().IntVar(&opts.RepeatedFailures, "repeated-failures", 0, "number of repeated failures")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "emit JSON")
}

func loadLevelingPolicy() (leveling.Policy, string, error) {
	root, err := levelingWorkspaceRoot()
	if err != nil {
		return leveling.Policy{}, "", err
	}
	path := filepath.Join(root, "leveling.yaml")
	override, err := readLevelingOverride(path)
	if err != nil {
		return leveling.Policy{}, path, fmt.Errorf("leveling_policy_missing: read policy: %w", err)
	}
	defaults, err := readEmbeddedLevelingDefaults()
	if err != nil {
		return leveling.Policy{}, path, fmt.Errorf("leveling_policy_stale: read embedded defaults: %w", err)
	}
	effective, _, err := leveling.LoadAuthorized(root, defaults, override, path)
	if err != nil {
		return leveling.Policy{}, path, fmt.Errorf("leveling: load authorized policy: %w", err)
	}
	return effective.Policy, path, nil
}

func levelingWorkspaceRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("leveling: get cwd: %w", err)
	}
	root, _, err := findStrategistRoot(cwd)
	if err != nil {
		return "", fmt.Errorf("leveling: %w", err)
	}
	return root, nil
}

func readLevelingOverride(path string) ([]byte, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is resolved below .strategist
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return raw, nil
}

func readEmbeddedLevelingDefaults() ([]byte, error) {
	raw, err := (embed.Extractor{}).ReadFile("leveling.yaml")
	if err != nil {
		return nil, fmt.Errorf("read embedded leveling.yaml: %w", err)
	}
	return raw, nil
}

func init() {
	humanStatusCommands["leveling"] = true
}

// registerLeveling attaches LEVELING commands at root composition. Policy and
// suggestion behavior remain owned by internal/leveling.
func registerLeveling(root *cobra.Command) {
	levelingCmd.AddCommand(levelingValidateCmd, levelingSuggestCmd, levelingLabelCmd)
	root.AddCommand(levelingCmd)
}

// newLevelingCommand creates an isolated adapter tree. Policy loading,
// validation and model/effort selection remain in internal/leveling.
func newLevelingCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "leveling", Short: "Suggest model and effort by role"}
	cmd.AddCommand(newLevelingValidateCommand(), newLevelingSuggestCommand(), newLevelingLabelCommand())
	return cmd
}
