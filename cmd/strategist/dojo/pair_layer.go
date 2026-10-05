package dojo

import (
	"fmt"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// pairLayer reports whether a case evaluates requirement-to-task pairs, one
// atomic dimension at a time, instead of whole packages.
func pairLayer(cc caseContext) bool { return layerOf(cc.criteria) == "pair" }

// PairEvidence is the per-run evidence of a pair case: numbers and ids only.
type PairEvidence struct {
	Schema       string                      `json:"schema"`
	Case         string                      `json:"case"`
	At           string                      `json:"at"`
	Model        string                      `json:"model,omitempty"`
	Profile      string                      `json:"profile"`
	ProfileVer   string                      `json:"profile_version"`
	Calls        int                         `json:"calls"`
	Skipped      int                         `json:"skipped_calls"`
	InputTokens  int                         `json:"input_tokens"`
	OutputTokens int                         `json:"output_tokens"`
	Report       calibration.PairReport      `json:"report"`
	Consent      *internaldojo.ConsentRecord `json:"consent,omitempty"`
}

func pairPlan(deps Dependencies, cc caseContext, pairs []calibration.Pair, reps int) (internaldojo.ConsentTerm, error) {
	cal := *cc.criteria.Calibration
	bytes := 0
	for _, p := range pairs {
		bytes += len(p.State())
	}
	calls := len(pairs) * reps
	if cal.Budget > 0 && cal.Budget < calls {
		calls = cal.Budget
	}
	id, _ := deps.Calibration.PairProfile()
	return internaldojo.NewTerm(internaldojo.ConsentTerm{
		Case: cc.criteria.Scenario, Layer: "pair", Provider: providerName(cal), Endpoint: deps.Calibration.Endpoint(cc.strategistRoot), Profile: id,
		DataClass: internaldojo.DataPairs, Fields: "requirement (text, modality), target (statement, declared_kind, action, verification)", Cases: len(pairs), Calls: calls,
		Bytes: bytes * reps, EstimatedTokens: estimateTokens(bytes*reps, calls),
	}), nil
}

func runPairLayer(cmd *cobra.Command, deps Dependencies, cc caseContext, flags calibrateFlags) error {
	if flags.recompute {
		return checkPairCase(cmd, cc, "latest")
	}
	pairs, err := internaldojo.LoadPairs(cc.paths.PairsPath)
	if err != nil {
		return fmt.Errorf("dojo calibrate: %w", err)
	}
	reps := resolveRepetitions(flags, *cc.criteria.Calibration)
	term, err := pairPlan(deps, cc, pairs, reps)
	if err != nil {
		return err
	}
	if flags.dryRun {
		return printPairPlan(cmd, cc, pairs, reps, term)
	}
	consent, err := assent(term, flags)
	if err != nil {
		return err
	}
	return executePairs(cmd, deps, cc, pairs, flags, reps, consent)
}

func printPairPlan(cmd *cobra.Command, cc caseContext, pairs []calibration.Pair, reps int, term internaldojo.ConsentTerm) error {
	counts := internaldojo.PairCounts(pairs)
	parts := make([]string, 0, len(counts))
	for _, family := range internaldojo.PairFamilies(pairs) {
		parts = append(parts, fmt.Sprintf("%d %s", counts[family], family))
	}
	return say(cmd.OutOrStdout(), "case: %s  layer: pair  boundary: %s\npairs: %d (%s) from %d parents\ncalls: %d (%d pairs x %d repetitions); one request asks all %d dimensions\n%s",
		cc.criteria.Scenario, calibration.BoundaryPairName, len(pairs), strings.Join(parts, ", "), parentsOf(pairs), len(pairs)*reps, len(pairs), reps, len(calibration.PairDimensions()), term.Text())
}

func parentsOf(pairs []calibration.Pair) int {
	set := map[string]bool{}
	for _, p := range pairs {
		set[p.Parent] = true
	}
	return len(set)
}

func executePairs(cmd *cobra.Command, deps Dependencies, cc caseContext, pairs []calibration.Pair, flags calibrateFlags, reps int, consent internaldojo.ConsentRecord) error {
	judge, err := deps.Calibration.PairJudge(cc.strategistRoot)
	if err != nil {
		return fmt.Errorf("dojo calibrate: %w", err)
	}
	cal := *cc.criteria.Calibration
	budget := flags.budget
	if budget == 0 {
		budget = cal.Budget
	}
	profile, version := deps.Calibration.PairProfile()
	started := time.Now()
	out, err := internaldojo.RunPairs(cmd.Context(), internaldojo.PairRunInput{Paths: cc.paths, Criteria: cc.criteria, Judge: judge, Pairs: pairs, Profile: profile, ProfileVersion: version, Repetitions: reps, Budget: budget}, internaldojo.SessionID(started))
	if err != nil {
		return fmt.Errorf("dojo calibrate: %w", err)
	}
	if err := printResult(cmd.OutOrStdout(), out.Check.Result); err != nil {
		return err
	}
	_ = say(cmd.OutOrStdout(), "calls: %d made, %d failed, %d skipped by the budget; measured provider usage: %d input and %d output tokens; session %s\n", out.Summary.Calls, out.Summary.Failures, out.Summary.Skipped, out.Summary.InputTokens, out.Summary.OutputTokens, out.Session) //nolint:errcheck // best-effort summary line
	finished := time.Now()
	evidence := PairEvidence{Schema: "strategist-dojo-pair-evidence/v1", Case: cc.criteria.Scenario, At: finished.UTC().Format(time.RFC3339), Model: out.Check.Model, Profile: profile, ProfileVer: version,
		Calls: out.Summary.Calls, Skipped: out.Summary.Skipped, InputTokens: out.Summary.InputTokens, OutputTokens: out.Summary.OutputTokens, Report: out.Check.Report, Consent: &consent}
	persistPair(cmd, cc, out.Check.Result, evidence, started, finished)
	return verdictOfPair(cc, out.Check.Result)
}

func checkPairCase(cmd *cobra.Command, cc caseContext, session string) error {
	started := time.Now()
	check, err := internaldojo.CheckPairs(cc.criteria, cc.paths, session)
	if err != nil {
		return fmt.Errorf("dojo check: %w", err)
	}
	if err := printResult(cmd.OutOrStdout(), check.Result); err != nil {
		return err
	}
	evidence := PairEvidence{Schema: "strategist-dojo-pair-evidence/v1", Case: cc.criteria.Scenario, At: time.Now().UTC().Format(time.RFC3339), Model: check.Model, Report: check.Report}
	persistPair(cmd, cc, check.Result, evidence, started, time.Now())
	return verdictOfPair(cc, check.Result)
}

func persistPair(cmd *cobra.Command, cc caseContext, result domain.DojoCheckResult, evidence PairEvidence, started, finished time.Time) {
	warn := func(what string, err error) {
		if err != nil {
			_ = say(cmd.ErrOrStderr(), "dojo: warning: failed to persist %s (verdict preserved): %v\n", what, err) //nolint:errcheck // best-effort warning
		}
	}
	warn("result", internaldojo.PersistResult(cc.basePath, result, started, finished))
	warn("evidence", internaldojo.WriteJSON(cc.paths.EvidencePath, evidence))
	warn("lesson", internaldojo.WriteLesson(cc.basePath, result))
}

func verdictOfPair(cc caseContext, result domain.DojoCheckResult) error {
	if result.Passed() {
		return nil
	}
	return fmt.Errorf("dojo: pair case %q failed (%d checks failed)", cc.criteria.Scenario, result.FailCount())
}
