package dojo

import (
	"fmt"
	"time"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// integrationLayer reports whether a case is evaluated at the integration layer:
// the approval policy against scripted answers, with no provider and no consent.
func integrationLayer(cc caseContext) bool { return layerOf(cc.criteria) == "integration" }

// runIntegrationLayer runs the policy matrix and keeps its result, evidence and,
// on a failure, the lesson, like any other case.
func runIntegrationLayer(cmd *cobra.Command, deps Dependencies, cc caseContext) error {
	started := time.Now()
	result := domain.DojoCheckResult{Scenario: cc.criteria.Scenario, Items: deps.Calibration.Policy(started)}
	if err := printResult(cmd.OutOrStdout(), result); err != nil {
		return err
	}
	finished := time.Now()
	out := internaldojo.CheckOutcome{Result: result}
	evidence := internaldojo.NewEvidence(cc.criteria.Scenario, "integration", out, "", summaryZero(), finished)
	persistCalibration(cmd.ErrOrStderr(), cc, out, evidence, started, finished)
	if !result.Passed() {
		return fmt.Errorf("dojo: integration case %q failed (%d checks failed)", cc.criteria.Scenario, result.FailCount())
	}
	return nil
}

// liveLayer reports whether a case is the live smoke check: the real provider,
// synthetic text only, behind a consent term.
func liveLayer(cc caseContext) bool { return layerOf(cc.criteria) == "live" }

// runLiveLayer prints the consent term on --dry-run, and otherwise needs --consent
// before the live smoke check calls the real provider. Its result, evidence (with
// the assent) and lesson are kept like any other case's.
func runLiveLayer(cmd *cobra.Command, deps Dependencies, cc caseContext, flags calibrateFlags) error {
	term := liveTerm(deps, cc)
	if flags.dryRun {
		return say(cmd.OutOrStdout(), "case: %s  layer: live\n%s", cc.criteria.Scenario, term.Text())
	}
	consent, err := assent(term, flags)
	if err != nil {
		return err
	}
	started := time.Now()
	items, err := deps.Calibration.Live(cmd.Context(), cc.strategistRoot)
	if err != nil {
		return fmt.Errorf("dojo calibrate: %w", err)
	}
	result := domain.DojoCheckResult{Scenario: cc.criteria.Scenario, Items: items}
	if err := printResult(cmd.OutOrStdout(), result); err != nil {
		return err
	}
	finished := time.Now()
	out := internaldojo.CheckOutcome{Result: result}
	evidence := internaldojo.NewEvidence(cc.criteria.Scenario, "live", out, "", summaryZero(), finished)
	evidence.Consent = &consent
	persistCalibration(cmd.ErrOrStderr(), cc, out, evidence, started, finished)
	if !result.Passed() {
		return fmt.Errorf("dojo: live case %q failed (%d checks failed)", cc.criteria.Scenario, result.FailCount())
	}
	return nil
}
