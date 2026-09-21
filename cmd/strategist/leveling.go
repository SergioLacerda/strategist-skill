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

var levelingValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the customer LEVELING policy",
	RunE: func(cmd *cobra.Command, _ []string) error {
		policy, path, err := loadLevelingPolicy()
		if err != nil {
			return err
		}
		if levelingValidateExpectedDigest != "" {
			if err := policy.VerifyDigest(levelingValidateExpectedDigest); err != nil {
				return fmt.Errorf("leveling: verify digest: %w", err)
			}
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "policy=%s version=%d digest=%s providers=%d\n", path, policy.Version, policy.Digest(), len(policy.Providers)); err != nil {
			return fmt.Errorf("leveling: write validation result: %w", err)
		}
		return nil
	},
}

var levelingValidateExpectedDigest string

type levelingSuggestOptions struct {
	Provider, Role, Ambiguity, Risk, Scope, Evidence            string
	ArchitecturalChange, SecuritySensitive, ConflictingEvidence bool
	RepeatedFailures                                            int
	JSON                                                        bool
}

var levelingSuggestOpts levelingSuggestOptions

var levelingSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest a model and effort for a role",
	RunE: func(cmd *cobra.Command, _ []string) error {
		policy, _, err := loadLevelingPolicy()
		if err != nil {
			return err
		}
		suggestion, err := leveling.Suggest(policy, levelingSuggestOpts.Provider, levelingSuggestOpts.Role, leveling.Signals{
			Ambiguity: levelingSuggestOpts.Ambiguity, Risk: levelingSuggestOpts.Risk, Scope: levelingSuggestOpts.Scope, Evidence: levelingSuggestOpts.Evidence,
			ArchitecturalChange: levelingSuggestOpts.ArchitecturalChange, SecuritySensitive: levelingSuggestOpts.SecuritySensitive,
			ConflictingEvidence: levelingSuggestOpts.ConflictingEvidence, RepeatedFailures: levelingSuggestOpts.RepeatedFailures,
		})
		if err != nil {
			return fmt.Errorf("leveling: suggest: %w", err)
		}
		if levelingSuggestOpts.JSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(suggestion)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "provider=%s role=%s model=%s capability=%s effort=%s fallback_used=%t policy_digest=%s\n", suggestion.Provider, suggestion.Role, suggestion.Model, suggestion.Capability, suggestion.Effort, suggestion.FallbackUsed, suggestion.PolicyDigest)
		if err != nil {
			return fmt.Errorf("leveling: write suggestion: %w", err)
		}
		return nil
	},
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
	levelingSuggestCmd.Flags().StringVar(&levelingSuggestOpts.Provider, "provider", "CODEX", "ranked provider identifier")
	levelingSuggestCmd.Flags().StringVar(&levelingSuggestOpts.Role, "role", "ranger", "Strategist role")
	levelingSuggestCmd.Flags().StringVar(&levelingSuggestOpts.Ambiguity, "ambiguity", "", "ambiguity signal: low, medium, or high")
	levelingSuggestCmd.Flags().StringVar(&levelingSuggestOpts.Risk, "risk", "", "risk signal: low, medium, or high")
	levelingSuggestCmd.Flags().StringVar(&levelingSuggestOpts.Scope, "scope", "", "scope signal: bounded or cross_module")
	levelingSuggestCmd.Flags().StringVar(&levelingSuggestOpts.Evidence, "evidence", "", "evidence signal: sufficient, insufficient, or conflicting")
	levelingSuggestCmd.Flags().BoolVar(&levelingSuggestOpts.ArchitecturalChange, "architectural-change", false, "mark an architectural change")
	levelingSuggestCmd.Flags().BoolVar(&levelingSuggestOpts.SecuritySensitive, "security-sensitive", false, "mark a security-sensitive task")
	levelingSuggestCmd.Flags().BoolVar(&levelingSuggestOpts.ConflictingEvidence, "conflicting-evidence", false, "mark conflicting evidence")
	levelingSuggestCmd.Flags().IntVar(&levelingSuggestOpts.RepeatedFailures, "repeated-failures", 0, "number of repeated failures")
	levelingSuggestCmd.Flags().BoolVar(&levelingSuggestOpts.JSON, "json", false, "emit JSON")
	levelingValidateCmd.Flags().StringVar(&levelingValidateExpectedDigest, "expected-digest", "", "fail if the normalized policy digest differs")
	levelingCmd.AddCommand(levelingValidateCmd, levelingSuggestCmd)
	rootCmd.AddCommand(levelingCmd)
}
