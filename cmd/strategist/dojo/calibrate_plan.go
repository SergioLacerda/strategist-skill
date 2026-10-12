package dojo

import (
	"fmt"
	"io"
	"strings"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
)

// Rough size of the typed questions the provider receives besides the state,
// used only for the usage estimate of the dry run.
const questionOverheadTokens = 300

// estimateTokens is the estimated input tokens of calls that send bytes of projected
// text, fitted to the measured usage of two live runs (about 2.4 bytes per token plus
// the questions): it ran within 10 percent, where bytes/4 plus 250 ran 35 percent low.
func estimateTokens(bytes, calls int) int { return bytes*5/12 + calls*questionOverheadTokens }

func printPlan(out io.Writer, deps Dependencies, cc caseContext, corpus internaldojo.Corpus, cases []calibration.Case, reps int) error {
	cal := *cc.criteria.Calibration
	plan, err := deps.Calibration.Plan(cc.strategistRoot, cal.Profile, cal.Boundary, cases)
	if err != nil {
		return fmt.Errorf("dojo calibrate: %w", err)
	}
	counts := internaldojo.Summary(cases)
	golden, candidates, held := tallyEntries(corpus.Manifest.Entries)
	calls := calibration.Plan(cases, reps)
	var b strings.Builder
	fmt.Fprintf(&b, "case: %s  boundary: %s  layer: %s\n", cc.criteria.Scenario, cal.Boundary, layerOf(cc.criteria))
	fmt.Fprintf(&b, "corpus: %d packages (golden %d, candidate %d, held-out %d), digest %s\n", len(corpus.Manifest.Entries), golden, candidates, held, corpus.Manifest.Digest)
	fmt.Fprintf(&b, "cases: %d (%s)\n", len(cases), kindCounts(counts))
	fmt.Fprintf(&b, "calls: %d (%d cases x %d repetitions), budget %s\n", calls, len(cases), reps, budgetText(cal.Budget))
	fmt.Fprintf(&b, "fields allowed by the data policy: %s\n", strings.Join(plan.Fields, ", "))
	fmt.Fprintf(&b, "projected bytes: %d per repetition, %d in total; %d cases denied by the data policy\n", plan.Bytes, plan.Bytes*reps, plan.Denied)
	fmt.Fprintf(&b, "estimated provider usage: about %d input tokens (an estimate, not a measurement)\n", estimateTokens(plan.Bytes*reps, calls))
	b.WriteString(sampleLine(internaldojo.CountDefects(cases), internaldojo.DefectParents(cases), golden, cal))
	b.WriteString(termFor(deps, cc, corpus, cases, plan, reps).Text())
	if _, err := io.WriteString(out, b.String()); err != nil {
		return fmt.Errorf("dojo calibrate: write plan: %w", err)
	}
	return nil
}

func tallyEntries(entries []internaldojo.CorpusEntry) (golden, candidates, held int) {
	for _, e := range entries {
		if e.Label == string(calibration.KindGolden) {
			golden++
		} else {
			candidates++
		}
		if e.HeldOut {
			held++
		}
	}
	return golden, candidates, held
}

func kindCounts(counts map[calibration.Kind]int) string {
	kinds := []calibration.Kind{calibration.KindGolden, calibration.KindCandidate, calibration.KindNatural, calibration.KindSynthetic, calibration.KindBoundary, calibration.KindNeutral}
	parts := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		if counts[kind] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[kind], kind))
		}
	}
	return strings.Join(parts, ", ")
}

func budgetText(budget int) string {
	if budget == 0 {
		return "unlimited"
	}
	return fmt.Sprint(budget)
}

// sampleLine says what the corpus can and cannot certify before any call is made.
// Defects are counted at their effective size: mutants of one parent are not independent.
func sampleLine(defects, parents, golden int, cal domainCalibration) string {
	rho := calibration.ConfigForRule(cal.GateRule).Policy.ClusterRho
	effective := calibration.EffectiveSample(defects, parents, rho)
	needNeg := calibration.MinSample(cal.RiskPolicy.MaxFalseApproval)
	needPos := calibration.MinSample(cal.RiskPolicy.MaxFalseRejection)
	line := fmt.Sprintf("sample: %d golden parents (need at least %d) and %d defect negatives from %d parents, an effective %.0f at correlation %.2f (need an effective %d), to certify the policy; %s",
		golden, needPos, defects, parents, effective, rho, needNeg, readingText(cal.GateRule))
	if golden == 0 {
		line += "; no package is golden yet: review packages and run `strategist dojo promote`"
	}
	return line + "\n"
}

// readingText says which part of the corpus the gate rule reads.
func readingText(rule string) string {
	if rule == calibration.RuleGates3 {
		return "gates-3 reads all of them, holding each package out in turn by cross-validation"
	}
	return "the selection side holds about three quarters of them"
}
