package dojo

import (
	"fmt"
	"io"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/spf13/cobra"
)

// recompute evaluates the stored observations again with no call.
func recompute(cmd *cobra.Command, deps Dependencies, cc caseContext, flags calibrateFlags) error {
	session := "latest"
	if flags.allSessions {
		session = ""
	}
	return checkCalibrationCase(cmd, deps, cc, session, flags.whatIf())
}

// checkExclusions allows what-ifs only for a recompute and only with known names.
func checkExclusions(flags calibrateFlags) error {
	if err := needRecompute(flags); err != nil {
		return err
	}
	if !calibration.KnownGateRule(flags.gateRule) {
		return fmt.Errorf("dojo calibrate: unknown gate rule %q", flags.gateRule)
	}
	for _, name := range append(append([]string(nil), flags.whatIfLocal...), flags.exclude...) {
		if _, ok := calibration.ByName(name); !ok {
			return fmt.Errorf("dojo calibrate: unknown operator %q", name)
		}
	}
	return nil
}

// needRecompute refuses an offline option on a run that would call out.
func needRecompute(flags calibrateFlags) error {
	if flags.recompute {
		return nil
	}
	for flag, set := range map[string]bool{"--local-scope": len(flags.whatIfLocal) > 0, "--exclude-operator": len(flags.exclude) > 0, "--exclude-criterion": len(flags.excludeCriteria) > 0, "--judge-only": flags.judgeOnly, "--gate-rule": flags.gateRule != ""} {
		if set {
			return fmt.Errorf("dojo calibrate: %s is an offline option and needs --recompute", flag)
		}
	}
	return nil
}

// layerCheck handles the cases that are not a judge-layer calibration: it reports
// whether the case was handled here and the error of that handling.
func layerCheck(cmd *cobra.Command, deps Dependencies, cc caseContext) (bool, error) {
	switch {
	case integrationLayer(cc):
		return true, runIntegrationLayer(cmd, deps, cc)
	case pairLayer(cc):
		return true, checkPairCase(cmd, cc, "latest")
	case liveLayer(cc):
		return true, fmt.Errorf("dojo check: %q is a live case; a check never calls out. Read the term with `strategist dojo calibrate %s --dry-run`, then run it with --consent", cc.criteria.Scenario, cc.criteria.Scenario)
	}
	return false, nil
}

// checkCalibrationCase is the `dojo check` of a calibration case: the same pass
// or fail a run ends with, computed from the observation ledger with no call.
func checkCalibrationCase(cmd *cobra.Command, deps Dependencies, cc caseContext, session string, what whatIf) error {
	if handled, err := layerCheck(cmd, deps, cc); handled {
		return err
	}
	cal := *cc.criteria.Calibration
	_, version := deps.Calibration.Profile(cal.Profile, cal.Boundary)
	started := time.Now()
	out, err := internaldojo.CheckCalibration(internaldojo.CheckInput{
		Criteria: cc.criteria, Paths: cc.paths, Current: deps.Calibration.Current(cc.strategistRoot), ProfileVersion: version, Session: session, ExcludeOperators: what.exclude, GateRule: what.gateRule, WhatIfLocalScope: what.local, ExcludeCriteria: what.criteria, JudgeOnly: what.judgeOnly, LocalFlags: localBackfill(deps, cc),
	})
	if err != nil {
		return fmt.Errorf("dojo check: %w", err)
	}
	if err := printResult(cmd.OutOrStdout(), out.Result); err != nil {
		return err
	}
	if what.exploratory() {
		return exploratoryVerdict(cmd, out)
	}
	return persistAndVerdict(cmd, cc, out, started)
}

func persistAndVerdict(cmd *cobra.Command, cc caseContext, out internaldojo.CheckOutcome, started time.Time) error {
	finished := time.Now()
	evidence := internaldojo.NewEvidence(cc.criteria.Scenario, layerOf(cc.criteria), out, internaldojo.ReadManifestDigest(cc.paths), summaryZero(), finished)
	persistCalibration(cmd.ErrOrStderr(), cc, out, evidence, started, finished)
	if !out.Result.Passed() {
		return fmt.Errorf("dojo check: case %q failed (%d checks failed)", cc.criteria.Scenario, out.Result.FailCount())
	}
	return nil
}

// persistCalibration writes the result, history, evidence and, for a failed
// case, the lesson. A persistence failure is reported and never changes the verdict.
func persistCalibration(errOut io.Writer, cc caseContext, out internaldojo.CheckOutcome, evidence internaldojo.Evidence, started, finished time.Time) {
	warn := func(what string, err error) {
		if err != nil {
			fmt.Fprintf(errOut, "dojo: warning: failed to persist %s (verdict preserved): %v\n", what, err) //nolint:errcheck // best-effort warning
		}
	}
	warn("result", internaldojo.PersistResult(cc.basePath, out.Result, started, finished))
	warn("evidence", internaldojo.WriteEvidence(cc.paths, evidence))
	warn("lesson", internaldojo.WriteCalibrationLesson(cc.basePath, out))
}

// runCalibrationCheck loads the case and evaluates its latest session.
func runCalibrationCheck(cmd *cobra.Command, deps Dependencies, root, name string) error {
	cc, err := loadCase(deps, root, name)
	if err != nil {
		return err
	}
	return checkCalibrationCase(cmd, deps, cc, "latest", whatIf{})
}

// exploratoryVerdict ends a what-if: nothing is persisted (it must not replace the
// case's real result, evidence or history) and the verdict is the same pass or fail.
func exploratoryVerdict(cmd *cobra.Command, out internaldojo.CheckOutcome) error {
	if err := say(cmd.OutOrStdout(), "exploratory what-if: nothing was persisted and no calibration is published\n"); err != nil {
		return err
	}
	if !out.Result.Passed() {
		return fmt.Errorf("dojo check: what-if for %q fails (%d checks failed)", out.Result.Scenario, out.Result.FailCount())
	}
	return nil
}

// localBackfill derives the local flags of every case of the case's corpus, but only
// when that corpus is the one the observations were made on (its digest equals the
// stored manifest's). Otherwise it returns nothing and the observations stay as recorded.
func localBackfill(deps Dependencies, cc caseContext) map[string][]string {
	cal := *cc.criteria.Calibration
	corpus, err := internaldojo.BuildCorpus(cc.basePath, cal, deps.Calibration.Reader)
	if err != nil || corpus.Manifest.Digest != internaldojo.ReadManifestDigest(cc.paths) {
		return nil
	}
	flags := map[string][]string{}
	for _, c := range internaldojo.BuildCases(corpus, cal) {
		flags[c.ID] = calibration.LocalFlags(c.Boundary, c.Fields)
	}
	return flags
}
