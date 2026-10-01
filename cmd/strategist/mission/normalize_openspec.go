package mission

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NormalizeOptions carries the `mission normalize-openspec` flag values.
type NormalizeOptions struct {
	Root, MissionID, ChangeID, RuntimeRoot, Pending string
	// Amend switches to the post-publication amendment mode; Amends and
	// AuthorizationRef are its required companions.
	Amend                    bool
	Amends, AuthorizationRef string
	// HandoffFacts is an optional YAML file with the typed handoff_policy_facts
	// mapping Archivist declares at publication.
	HandoffFacts string
}

// NormalizeDependencies injects mission-id validation and path resolution.
type NormalizeDependencies struct {
	RootFlag         string
	RequireMissionID func(string) error
	ResolvePaths     func(NormalizeOptions) (string, string, string, error)
	RecordConfidence func(NormalizeOptions, domain.ConfidenceClaim, []domain.Evidence) error
	// GateLabel returns the mission's gate outcome label ("" when none); it is
	// consulted only by --amend. Nil means no label is known.
	GateLabel func(NormalizeOptions) (string, error)
}

// NewNormalizeOpenSpec builds `mission normalize-openspec`.
func NewNormalizeOpenSpec(deps NormalizeDependencies) *cobra.Command {
	opts := NormalizeOptions{}
	cmd := &cobra.Command{
		Use:   "normalize-openspec",
		Short: "Publish a completed OpenSpec change into the refined mission package",
		Long: `Validates a private OpenSpec change and atomically promotes its proposal,
design, tasks, and the mission analysis into <base_path>/refined/<mission_id>.
OpenSpec specs and archive history remain private provider scratch.`,
	}
	f := cmd.Flags()
	f.StringVar(&opts.Root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&opts.MissionID, "mission-id", "", "canonical Strategist mission id (required)")
	f.StringVar(&opts.ChangeID, "change-id", "", "completed OpenSpec change id (required)")
	f.StringVar(&opts.RuntimeRoot, "runtime-root", "", "OpenSpec runtime root (default: <strategist-root>/openspec)")
	f.StringVar(&opts.Pending, "pending-analysis", "", "pending analysis path (default: <base_path>/pending/<mission-id>-analysis.md)")
	f.BoolVar(&opts.Amend, "amend", false, "amend an already published package with a new change instead of publishing (analysis.md, the mission status and the original provider_change_id are kept; the previous files are snapshotted under .amendments/)")
	f.StringVar(&opts.Amends, "amends", "", "with --amend: the change being amended (the package's provider_change_id, or the previous amendment's change id)")
	f.StringVar(&opts.AuthorizationRef, "authorization-ref", "", "with --amend: the human authorization for the amendment (a quote or a gate event), recorded verbatim")
	f.StringVar(&opts.HandoffFacts, "handoff-facts", "", "YAML file with the typed handoff_policy_facts mapping, written into the published analysis.md frontmatter so `handoff evaluate` can derive the policy (publication only; an amendment keeps analysis.md byte-identical)")
	if err := cmd.MarkFlagRequired("change-id"); err != nil {
		panic(err)
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunNormalizeOpenSpec(cmd, deps, opts) }
	return cmd
}

// RunNormalizeOpenSpec publishes a completed OpenSpec change through
// refinement.NormalizeOpenSpec.
func RunNormalizeOpenSpec(cmd *cobra.Command, deps NormalizeDependencies, opts NormalizeOptions) error {
	if err := deps.RequireMissionID(opts.MissionID); err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	if err := validateAmendFlags(cmd, opts); err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	basePath, runtimeRoot, pending, err := deps.ResolvePaths(opts)
	if err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	if opts.Amend {
		return runAmendChecked(cmd, deps, opts, basePath, runtimeRoot)
	}
	return runPublish(cmd, deps, opts, basePath, runtimeRoot, pending)
}

// runAmendChecked refuses facts with --amend: an amendment keeps analysis.md
// byte-identical, so the facts are declared at publication.
func runAmendChecked(cmd *cobra.Command, deps NormalizeDependencies, opts NormalizeOptions, basePath, runtimeRoot string) error {
	if opts.HandoffFacts != "" {
		return fmt.Errorf("mission normalize-openspec: --handoff-facts cannot be used with --amend: an amendment keeps analysis.md byte-identical; declare the facts at publication")
	}
	return runAmend(cmd, deps, opts, basePath, runtimeRoot)
}

// runPublish publishes the change and, when facts were not declared, says so on
// stderr: the package then has no evaluable handoff policy.
func runPublish(cmd *cobra.Command, deps NormalizeDependencies, opts NormalizeOptions, basePath, runtimeRoot, pending string) error {
	facts, err := readHandoffFacts(opts.HandoffFacts)
	if err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	if deps.RecordConfidence == nil {
		return fmt.Errorf("mission normalize-openspec: Archivist confidence recorder is unavailable")
	}
	result, err := refinement.NormalizeOpenSpec(refinement.OpenSpecInput{
		MissionID: opts.MissionID, BasePath: basePath, RuntimeRoot: runtimeRoot,
		ChangeID: opts.ChangeID, PendingAnalysisPath: pending, HandoffFacts: facts,
		RecordConfidence: func(claim domain.ConfidenceClaim, evidence []domain.Evidence) error {
			return deps.RecordConfidence(opts, claim, evidence)
		},
	})
	if err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	return reportPublished(cmd, opts, result, facts == nil)
}

func reportPublished(cmd *cobra.Command, opts NormalizeOptions, result refinement.OpenSpecResult, factsMissing bool) error {
	if factsMissing {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "[Strategist] warning=handoff_policy_facts_not_declared mission_id=%s: `strategist handoff evaluate` rejects this package until handoff_policy_facts is declared (--handoff-facts)\n", opts.MissionID); err != nil {
			return fmt.Errorf("mission normalize-openspec: write output: %w", err)
		}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s provider_change_id=%s refined=%s status=archivist_done\n", opts.MissionID, result.ProviderChangeID, result.RefinedPath); err != nil {
		return fmt.Errorf("mission normalize-openspec: write output: %w", err)
	}
	return nil
}

// readHandoffFacts loads the optional facts file; an empty path declares none.
func readHandoffFacts(path string) (map[string]any, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path) //nolint:gosec // G304: operator-supplied facts file path, same trust level as the other normalize inputs
	if err != nil {
		return nil, fmt.Errorf("read handoff facts: %w", err)
	}
	var facts map[string]any
	if err := yaml.Unmarshal(raw, &facts); err != nil {
		return nil, fmt.Errorf("parse handoff facts: %w", err)
	}
	if len(facts) == 0 {
		return nil, fmt.Errorf("handoff facts file %s is empty", path)
	}
	return facts, nil
}
