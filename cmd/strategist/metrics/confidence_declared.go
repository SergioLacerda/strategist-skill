package metrics

import (
	"fmt"
	"io"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// declaredStdin is the only accepted value of --declared: the summary is transient
// input, the same rule as `metrics record --claim-file -`.
const declaredStdin = "-"

// validateDeclaredFlag checks --declared before anything is read or printed.
func validateDeclaredFlag(mission, declared string) error {
	if declared == "" {
		return nil
	}
	if declared != declaredStdin {
		return fmt.Errorf("metrics confidence: --declared accepts only %q (the summary is read from standard input)", declaredStdin)
	}
	if mission == "" {
		return fmt.Errorf("metrics confidence: --declared requires --mission")
	}
	return nil
}

// declaredDocument is a handoff block: only its confidence_summary is read, so the
// block of a tasks.md can be piped as it is.
type declaredDocument struct {
	Summary *domain.ConfidenceSummary `yaml:"confidence_summary"`
}

// readDeclaredSummary decodes and validates the summary on in. An invalid summary
// fails here, before any output.
func readDeclaredSummary(in io.Reader) (domain.ConfidenceSummary, error) {
	raw, err := io.ReadAll(in)
	if err != nil {
		return domain.ConfidenceSummary{}, fmt.Errorf("metrics confidence: read declared summary: %w", err)
	}
	var doc declaredDocument
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return domain.ConfidenceSummary{}, fmt.Errorf("metrics confidence: parse declared summary: %w", err)
	}
	if doc.Summary == nil {
		return domain.ConfidenceSummary{}, fmt.Errorf("metrics confidence: the document on standard input has no confidence_summary key")
	}
	if err := validateDeclaredClaims(*doc.Summary); err != nil {
		return domain.ConfidenceSummary{}, fmt.Errorf("metrics confidence: declared summary invalid: %w", err)
	}
	return *doc.Summary, nil
}

// validateDeclaredClaims checks only what the comparison keys on: an id, an agent,
// a known kind and a percent in range for every claim, and no repeated pair. It is
// deliberately not domain.ValidateConfidenceSummary: the handoff blocks Archivists
// write carry a policy_version of "1" and evidence entries without a confidence, which
// the strict validator rejects, and a comparison meant to expose drift must accept the
// blocks the drift is in.
func validateDeclaredClaims(summary domain.ConfidenceSummary) error {
	seen := make(map[string]struct{})
	claims := append(append([]domain.ConfidenceClaim(nil), summary.Claims...), summary.OpenQuestions...)
	for _, claim := range claims {
		if err := validateDeclaredClaim(claim); err != nil {
			return err
		}
		key := claim.Agent + "/" + claim.ID
		if _, dup := seen[key]; dup {
			return fmt.Errorf("claim %q is declared twice", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateDeclaredClaim(claim domain.ConfidenceClaim) error {
	switch {
	case claim.ID == "":
		return fmt.Errorf("a claim has no id")
	case claim.Agent == "":
		return fmt.Errorf("claim %q requires agent", claim.ID)
	case claim.ClaimKind != domain.ClaimKindAssertion && claim.ClaimKind != domain.ClaimKindQuestion:
		return fmt.Errorf("claim %q has unknown claim_kind %q", claim.ID, claim.ClaimKind)
	case claim.ConfidencePercent < domain.ConfidencePercentMin || claim.ConfidencePercent > domain.ConfidencePercentMax:
		return fmt.Errorf("claim %q has confidence_percent %d out of range", claim.ID, claim.ConfidencePercent)
	}
	return nil
}

// PrintDeclaredComparison appends the declared-versus-persisted block after the
// existing confidence lines. It is read-only and never changes review_required.
func PrintDeclaredComparison(w io.Writer, cmp telemetry.DeclaredComparison) error {
	var b strings.Builder
	if cmp.MissingRecord {
		b.WriteString("declared_missing_record: true\n")
	} else {
		fmt.Fprintf(&b, "declared_claims: %d\ndeclared_assertions: %d\ndeclared_questions: %d\npersisted_claims: %d\n", cmp.DeclaredClaims, cmp.DeclaredAssertions, cmp.DeclaredQuestions, cmp.PersistedClaims)
		fmt.Fprintf(&b, "declared_unpersisted: %s\npersisted_rejected: %s\ndeclared_mismatched: %s\npersisted_under_other_agent: %s\n", idList(cmp.Unpersisted), idList(cmp.PersistedRejected), idList(cmp.Mismatched), idList(cmp.UnderOtherAgent))
		fmt.Fprintf(&b, "question_preservation_declared: %s\n", questionsDeclared(cmp))
		fmt.Fprintf(&b, "declared_review: %s\n", declaredReview(cmp))
	}
	if _, err := fmt.Fprint(w, b.String()); err != nil {
		return fmt.Errorf("metrics confidence: write output: %w", err)
	}
	return nil
}

func idList(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}

func questionsDeclared(cmp telemetry.DeclaredComparison) string {
	if cmp.DeclaredQuestions == 0 {
		return "n/a (no questions declared)"
	}
	return fmt.Sprintf("%d/%d", cmp.PersistedQuestions, cmp.DeclaredQuestions)
}

func declaredReview(cmp telemetry.DeclaredComparison) string {
	if cmp.ReviewRecommended() {
		return "recommended"
	}
	return "none"
}

// runConfidenceDeclared prints the review as RunConfidence does, then the
// comparison of the summary on cmd's stdin with the records the review projected.
func runConfidenceDeclared(cmd *cobra.Command, deps Dependencies, explicitRoot, mission string) error {
	summary, err := readDeclaredSummary(cmd.InOrStdin())
	if err != nil {
		return err
	}
	if deps.SilenceRun != nil {
		deps.SilenceRun(cmd)
	}
	root, err := deps.ResolveRoot(cmd, "confidence", explicitRoot)
	if err != nil {
		return err
	}
	review, records, err := telemetry.LoadConfidenceGateReviewWithRecords(root, mission)
	if err != nil {
		return fmt.Errorf("metrics confidence: %w", err)
	}
	if err := PrintConfidenceMetrics(cmd.OutOrStdout(), review); err != nil {
		return err
	}
	if err := PrintGateOutcome(cmd.OutOrStdout(), root, mission); err != nil {
		return err
	}
	return PrintDeclaredComparison(cmd.OutOrStdout(), telemetry.CompareDeclaredToPersisted(summary, records))
}
