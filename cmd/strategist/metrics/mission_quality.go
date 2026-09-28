package metrics

import (
	"fmt"
	"io"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// missionQualityDocument is the explicit input of `metrics mission-quality`.
// A nil previous-open list or scope list leaves that predicate not applicable;
// an empty list makes it applicable with nothing to preserve or match.
type missionQualityDocument struct {
	Decisions                 []domain.Decision `yaml:"decisions"`
	Evidence                  []domain.Evidence `yaml:"evidence"`
	AcceptanceCriteria        []string          `yaml:"acceptance_criteria"`
	PreviouslyOpenDecisionIDs []string          `yaml:"previously_open_decision_ids"`
	ApprovedScopePrefixes     []string          `yaml:"approved_scope_prefixes"`
}

// NewMissionQuality builds the `metrics mission-quality` command: an advisory
// evaluation of the six mission_quality predicates over an explicit document.
func NewMissionQuality(deps Dependencies) *cobra.Command {
	var root, mission string
	cmd := &cobra.Command{
		Use:   "mission-quality",
		Short: "Evaluate the mission_quality predicates over a decisions/evidence document",
		Long:  "Read a YAML document from standard input (decisions, evidence, acceptance_criteria, and optionally previously_open_decision_ids and approved_scope_prefixes) and report each mission_quality predicate. The result is advisory: a failed predicate is reported and the command still exits 0. The command reads no mission artifact on its own; the caller supplies the ledger and the evidence it cites.",
	}
	f := cmd.Flags()
	f.StringVar(&root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&mission, "mission", "", "mission id (required)")
	if err := cmd.MarkFlagRequired("mission"); err != nil {
		panic(err)
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunMissionQuality(cmd, deps, root, mission) }
	return cmd
}

// RunMissionQuality evaluates and prints the mission_quality predicates.
func RunMissionQuality(cmd *cobra.Command, deps Dependencies, explicitRoot, mission string) error {
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	if _, err := deps.ResolveRoot(cmd, "mission-quality", explicitRoot); err != nil {
		return err
	}
	doc, err := readMissionQualityDocument(cmd.InOrStdin())
	if err != nil {
		return err
	}
	return printMissionQuality(cmd.OutOrStdout(), mission, doc)
}

func readMissionQualityDocument(in io.Reader) (missionQualityDocument, error) {
	raw, err := io.ReadAll(in)
	if err != nil {
		return missionQualityDocument{}, fmt.Errorf("metrics mission-quality: read standard input: %w", err)
	}
	if strings.TrimSpace(string(raw)) == "" {
		return missionQualityDocument{}, fmt.Errorf("metrics mission-quality: standard input is empty; pipe the decisions/evidence document")
	}
	var top map[string]yaml.Node
	if err := yaml.Unmarshal(raw, &top); err != nil {
		return missionQualityDocument{}, fmt.Errorf("metrics mission-quality: parse document: %w", err)
	}
	for key := range top {
		if !missionQualityKeys[key] {
			return missionQualityDocument{}, fmt.Errorf("metrics mission-quality: unknown top-level key %q; expected decisions, evidence, acceptance_criteria, previously_open_decision_ids, approved_scope_prefixes", key)
		}
	}
	var doc missionQualityDocument
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return missionQualityDocument{}, fmt.Errorf("metrics mission-quality: parse document: %w", err)
	}
	return doc, nil
}

var missionQualityKeys = map[string]bool{
	"decisions": true, "evidence": true, "acceptance_criteria": true,
	"previously_open_decision_ids": true, "approved_scope_prefixes": true,
}

func printMissionQuality(w io.Writer, mission string, doc missionQualityDocument) error {
	var b strings.Builder
	fmt.Fprintf(&b, "mission: %s\n", mission)
	if len(doc.Decisions) == 0 && len(doc.Evidence) == 0 {
		b.WriteString("mission_quality: not_evaluated\n")
	} else {
		result := domain.EvaluateMissionQuality(domain.MissionQualityInput{
			Decisions: doc.Decisions, Evidence: doc.Evidence, AcceptanceCriteria: doc.AcceptanceCriteria,
			PreviouslyOpenDecisionIDs: doc.PreviouslyOpenDecisionIDs, ApprovedScopePrefixes: doc.ApprovedScopePrefixes,
		})
		writeQualityResult(&b, result)
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("metrics mission-quality: write output: %w", err)
	}
	return nil
}

func writeQualityResult(b *strings.Builder, result domain.MissionQualityResult) {
	verdict := "passed"
	if !result.Passed() {
		verdict = "failed"
	}
	fmt.Fprintf(b, "mission_quality: %s\n", verdict)
	for _, check := range result.Checks {
		fmt.Fprintf(b, "check: %s applicable=%t passed=%t\n", check.Check, check.Applicable, check.Passed)
		for _, violation := range check.Violations {
			fmt.Fprintf(b, "violation: %s\n", violation)
		}
	}
}
