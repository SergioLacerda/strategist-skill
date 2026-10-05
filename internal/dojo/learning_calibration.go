package dojo

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Failure reasons of a judge_calibration case. They join the stable reasons of
// the pipeline scenarios and are mined from the same history.
const (
	// FailureFalseApproval: defects were approved above the policy limit.
	FailureFalseApproval FailureReason = "false_approval"
	// FailureFalseRejection: golden positives were rejected above the policy limit.
	FailureFalseRejection FailureReason = "false_rejection"
	// FailureInsensitiveOperator: the judge does not respond to an injected defect.
	FailureInsensitiveOperator FailureReason = "insensitive_operator"
	// FailureNonMonotonic: a worse defect scored better than a milder one.
	FailureNonMonotonic FailureReason = "non_monotonic_response"
	// FailureInstability: the same case scored differently across repetitions.
	FailureInstability FailureReason = "instability"
	// FailureNotSeparating: golden positives and defects are not distinguishable.
	FailureNotSeparating FailureReason = "not_separating"
	// FailureStaleCalibration: the model or the question profile changed.
	FailureStaleCalibration FailureReason = "stale_calibration"
	// FailureNotCalibrated: there are no observations or no recommendation.
	FailureNotCalibrated FailureReason = "not_calibrated"
	// FailurePolicyViolation: the Strategist policy did not do what the contract says
	// with a scripted provider answer (the integration layer).
	FailurePolicyViolation FailureReason = "policy_violation"
	// FailureDimensionNotFit: a requirement-pair dimension is not fit to judge (or lacks a sample).
	FailureDimensionNotFit FailureReason = "dimension_not_fit"
	// FailureLiveCheck: the live smoke check against the provider failed (the provider,
	// the credential or the shape of the answer), not the Strategist policy.
	FailureLiveCheck FailureReason = "live_check_failed"
)

var calibrationHints = map[FailureReason]string{
	FailureFalseApproval:       "The configured threshold approves defects: raise it, or pin a calibration whose held-out false approval is within the policy.",
	FailureFalseRejection:      "Golden positives are rejected: review the oracle of those cases, or lower the threshold if the false approval limit allows.",
	FailureInsensitiveOperator: "The judge ignores an injected defect: tighten the question for that criterion or add a deterministic check on main.",
	FailureNonMonotonic:        "A worse defect scored better than a milder one: the response is not trustworthy for ranking; review the question wording.",
	FailureInstability:         "Scores vary across repetitions: use more repetitions or a pinned model before trusting any threshold.",
	FailureNotSeparating:       "The judge does not separate good packages from defects: do not calibrate a threshold; change the question profile.",
	FailureStaleCalibration:    "The question profile changed since the observations were taken: run `strategist dojo calibrate` again.",
	FailurePolicyViolation:     "The approval policy misbehaved with a scripted answer: this is a Strategist defect, not a provider one; fix handoffconsumer before trusting any threshold.",
	FailureDimensionNotFit:     "A pair dimension is not fit: read its AUC, margin and stability, and rewrite the question or drop that dimension from what the provider decides.",
	FailureLiveCheck:           "The provider smoke check failed: run `strategist integrations doctor`, check the credential reference and the pinned model, then repeat with --consent.",
	FailureNotCalibrated:       "No usable calibration: run `strategist dojo calibrate <case> --dry-run`, then with consent, or add golden promotions and more cases.",
}

func hintFor(reason FailureReason) (string, bool) {
	if hint, ok := nextActionHints[reason]; ok {
		return hint, true
	}
	hint, ok := calibrationHints[reason]
	return hint, ok
}

// classifyCalibrationLabel maps a calibration check label to its stable reason.
func classifyCalibrationLabel(label string) FailureReason {
	for _, rule := range []struct {
		needle string
		reason FailureReason
	}{
		{"calibration policy:", FailurePolicyViolation}, {"calibration dimension", FailureDimensionNotFit}, {"false_approval", FailureFalseApproval}, {"false_rejection", FailureFalseRejection}, {"insensitive", FailureInsensitiveOperator},
		{"gate monotonic", FailureNonMonotonic}, {"gate stability", FailureInstability}, {"gate separation", FailureNotSeparating},
		{"stale", FailureStaleCalibration},
	} {
		if strings.Contains(label, rule.needle) {
			return rule.reason
		}
	}
	return FailureNotCalibrated
}

// GenerateCalibrationLesson extends the generic lesson with what this case
// learned: the weak spots of the judge and the audit disagreements.
func GenerateCalibrationLesson(result domain.DojoCheckResult, report calibration.Report, audits []AuditEntry) string {
	var b strings.Builder
	b.WriteString(GenerateLesson(result, ClassifyFailures(result.Items)))
	b.WriteString("## Weak Spots\n\n")
	writeLines(&b, weakSpots(report))
	b.WriteString("\n## Audit Disagreements\n\n")
	writeLines(&b, disagreements(audits))
	return b.String()
}

func weakSpots(report calibration.Report) []string {
	var lines []string
	for _, op := range report.Operators {
		if op.Insensitive() {
			lines = append(lines, fmt.Sprintf("%s: responded in %d of %d pairs", op.Operator, op.Sensitivity.K, op.Sensitivity.N))
		}
	}
	return lines
}

func disagreements(audits []AuditEntry) []string {
	var lines []string
	for _, entry := range audits {
		if entry.Verdict == VerdictDisagree {
			lines = append(lines, fmt.Sprintf("%s (%s): %s", entry.CaseID, entry.By, entry.Note))
		}
	}
	return lines
}

func writeLines(b *strings.Builder, lines []string) {
	if len(lines) == 0 {
		b.WriteString("- none recorded\n")
	}
	for _, line := range lines {
		b.WriteString("- " + line + "\n")
	}
}

// WriteCalibrationLesson writes the lesson of a failed calibration case; a
// passing case writes none (it writes evidence instead).
func WriteCalibrationLesson(basePath string, out CheckOutcome) error {
	return writeLessonContent(basePath, out.Result, func() string {
		return GenerateCalibrationLesson(out.Result, out.Report, out.Audits)
	})
}
