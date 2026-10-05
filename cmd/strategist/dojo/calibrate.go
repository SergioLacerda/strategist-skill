package dojo

import (
	"fmt"
	"io"
	"time"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

type calibrateFlags struct {
	root            string
	dryRun          bool
	consent         bool
	consentTerm     string
	recompute       bool
	exclude         []string
	gateRule        string
	whatIfLocal     []string
	excludeCriteria []string
	judgeOnly       bool
	allSessions     bool
	budget          int
	repetitions     int
	boundaryMargin  float64
}

func newCalibrate(deps Dependencies) *cobra.Command {
	var flags calibrateFlags
	cmd := &cobra.Command{
		Use:   "calibrate <case>",
		Short: "Run a judge calibration case against labeled cases",
		Long: `Evaluate a judge (the provider handoff pre-check) against cases whose expected verdict is known,
keep every raw observation and publish a calibration record. Nothing is trained: the run measures the
judge and recommends a threshold only when the judge is fit and the sample supports it.

  --dry-run    print the exact cases, fields, bytes and estimated usage; no call is made
  --recompute  evaluate the stored observations again, with no call
  --consent    required to send real package fields to the provider`,
		Args: cobra.ExactArgs(1),
		// A case that fails its checks is a result, not a usage error: no usage dump.
		SilenceUsage: true,
		RunE:         func(cmd *cobra.Command, args []string) error { return runCalibrate(cmd, deps, flags, args[0]) },
	}
	f := cmd.Flags()
	f.StringVar(&flags.root, "root", "", rootFlagUsage)
	f.BoolVar(&flags.dryRun, "dry-run", false, "print the plan and send nothing")
	f.BoolVar(&flags.consent, "consent", false, "assent to the consent term: the run calls the external provider")
	f.StringVar(&flags.consentTerm, "consent-term", "", "bind the assent to exactly the term id printed by --dry-run; a changed plan is refused")
	f.BoolVar(&flags.recompute, "recompute", false, "evaluate the stored observations again with no call")
	f.StringSliceVar(&flags.exclude, "exclude-operator", nil, "with --recompute: leave an operator out of the evaluation (an exploratory what-if, labelled; no calibration is published from it)")
	f.StringSliceVar(&flags.whatIfLocal, "local-scope", nil, "with --recompute: delegate this operator to the local checks for the evaluation only (an exploratory what-if)")
	f.StringSliceVar(&flags.excludeCriteria, "exclude-criterion", nil, "with --recompute: read an atomic profile's stored answers without this criterion (an exploratory what-if over answers already given)")
	f.BoolVar(&flags.judgeOnly, "judge-only", false, "with --recompute: ignore the case's delegation to the local checks, so the judge alone answers for every operator (an exploratory what-if)")
	f.StringVar(&flags.gateRule, "gate-rule", "", "with --recompute: read the observations under gates-1 (the first pilots' rule), gates-2 (default) or gates-3 (cross-validated)")
	f.BoolVar(&flags.allSessions, "all-sessions", false, "with --recompute: use every stored session instead of the latest")
	f.IntVar(&flags.budget, "budget", 0, "maximum provider calls for this run (0 means the case's budget)")
	f.IntVar(&flags.repetitions, "repetitions", 0, "repetitions per case (0 means the case's value)")
	f.Float64Var(&flags.boundaryMargin, "boundary-margin", 0, "repeat cases whose score is within this margin of the current threshold")
	return cmd
}

func runCalibrate(cmd *cobra.Command, deps Dependencies, flags calibrateFlags, name string) error {
	cc, err := loadCase(deps, flags.root, name)
	if err != nil {
		return err
	}
	if err := checkExclusions(flags); err != nil {
		return err
	}
	switch {
	case integrationLayer(cc):
		return runIntegrationLayer(cmd, deps, cc)
	case liveLayer(cc):
		return runLiveLayer(cmd, deps, cc, flags)
	case pairLayer(cc):
		return runPairLayer(cmd, deps, cc, flags)
	case flags.recompute:
		return recompute(cmd, deps, cc, flags)
	}
	return runJudgeLayer(cmd, deps, cc, flags)
}

// runJudgeLayer prints the plan and consent term on --dry-run and otherwise needs
// the operator's assent before any provider call.
func runJudgeLayer(cmd *cobra.Command, deps Dependencies, cc caseContext, flags calibrateFlags) error {
	cal := *cc.criteria.Calibration
	corpus, err := internaldojo.BuildCorpus(cc.basePath, cal, deps.Calibration.Reader)
	if err != nil {
		return fmt.Errorf("dojo calibrate: %w", err)
	}
	cases := internaldojo.BuildCases(corpus, cal)
	reps := resolveRepetitions(flags, cal)
	if flags.dryRun {
		return printPlan(cmd.OutOrStdout(), deps, cc, corpus, cases, reps)
	}
	plan, err := deps.Calibration.Plan(cc.strategistRoot, cal.Profile, cal.Boundary, cases)
	if err != nil {
		return fmt.Errorf("dojo calibrate: %w", err)
	}
	record, err := assent(termFor(deps, cc, corpus, cases, plan, reps), flags)
	if err != nil {
		return err
	}
	return runLive(cmd, deps, cc, corpus, flags, reps, record)
}

func resolveRepetitions(flags calibrateFlags, cal domain.DojoCalibration) int {
	if flags.repetitions > 0 {
		return flags.repetitions
	}
	return internaldojo.Repetitions(cal)
}

func runLive(cmd *cobra.Command, deps Dependencies, cc caseContext, corpus internaldojo.Corpus, flags calibrateFlags, reps int, consent internaldojo.ConsentRecord) error {
	cal := *cc.criteria.Calibration
	judge, err := deps.Calibration.Judge(cc.strategistRoot, cal.Profile)
	if err != nil {
		return fmt.Errorf("dojo calibrate: %w", err)
	}
	profile, version := deps.Calibration.Profile(cal.Profile, cal.Boundary)
	budget := flags.budget
	if budget == 0 {
		budget = cal.Budget
	}
	started := time.Now()
	out, runErr := internaldojo.RunCalibration(cmd.Context(), internaldojo.RunInput{
		BasePath: cc.basePath, Criteria: cc.criteria, Paths: cc.paths, Corpus: corpus, Judge: judge, Profile: profile, ProfileVersion: version,
		Repetitions: reps, Budget: budget, BoundaryMargin: flags.boundaryMargin, Current: deps.Calibration.Current(cc.strategistRoot), Now: started,
	})
	if runErr != nil {
		return fmt.Errorf("dojo calibrate: %w", runErr)
	}
	return finishRun(cmd, cc, out, started, consent)
}

func finishRun(cmd *cobra.Command, cc caseContext, out internaldojo.RunOutcome, started time.Time, consent internaldojo.ConsentRecord) error {
	finished := time.Now()
	if err := printResult(cmd.OutOrStdout(), out.Check.Result); err != nil {
		return err
	}
	printRunSummary(cmd.OutOrStdout(), out)
	evidence := internaldojo.NewEvidence(cc.criteria.Scenario, layerOf(cc.criteria), out.Check, corpusDigest(out), out.Summary, finished)
	evidence.Consent = &consent
	persistCalibration(cmd.ErrOrStderr(), cc, out.Check, evidence, started, finished)
	if !out.Check.Result.Passed() {
		return fmt.Errorf("dojo calibrate: case %q failed (%d checks failed)", cc.criteria.Scenario, out.Check.Result.FailCount())
	}
	return nil
}

func corpusDigest(out internaldojo.RunOutcome) string {
	if out.Record == nil {
		return ""
	}
	return out.Record.CorpusDigest
}

func layerOf(criteria domain.DojoCriteria) string {
	if criteria.Calibration.Layer == "" {
		return "judge"
	}
	return criteria.Calibration.Layer
}

func printRunSummary(out io.Writer, run internaldojo.RunOutcome) {
	fmt.Fprintf(out, "calls: %d made, %d failed, %d skipped by the budget; session %s\n", run.Summary.Calls, run.Summary.Failures, run.Summary.Skipped, run.Session) //nolint:errcheck // best-effort summary line
	fmt.Fprintf(out, "measured provider usage: %d input and %d output tokens (reported by the provider)\n", run.Summary.InputTokens, run.Summary.OutputTokens)       //nolint:errcheck // best-effort summary line
	if run.Record != nil {
		fmt.Fprintf(out, "calibration: %s status=%s model=%s\n", run.Record.ID, run.Record.Status, run.Record.Model) //nolint:errcheck // best-effort summary line
	}
}
