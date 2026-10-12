package calibration

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Verdict is what a judge returned for one projection.
type Verdict struct {
	Decision      string
	Probabilities Distribution
	PAccept       float64
	Confidence    float64
	// Scores are the per-criterion noul answers of an atomic profile, empty otherwise.
	Scores  map[string]float64
	Model   string
	Latency time.Duration
	// InputTokens and OutputTokens are the usage the provider reported for the call.
	InputTokens  int
	OutputTokens int
	// State is the typed failure when the judge could not answer ("" when it did).
	State string
}

// Judge is the port of a system under evaluation. The JEV handoff pre-check is
// the first adapter; a scripted judge serves the integration layer and tests.
type Judge interface {
	Judge(ctx context.Context, boundary string, fields Fields) Verdict
}

// RunOptions bound one evaluation run.
type RunOptions struct {
	Repetitions    int
	Budget         int
	Profile        string
	ProfileVersion string
	// Session groups the observations of one run; StartRun is the number of the
	// first repetition (1 by default), so extra repetitions continue a session.
	Session  string
	StartRun int
	// Local evaluates the local checks of a case; nil disables them. It runs once
	// per case and its flags are copied to every observation of that case.
	Local func(boundary string, fields Fields) []string
	// Sink receives every observation as it is made.
	Sink func(Observation) error
	Now  func() time.Time
}

// Summary reports what a run did and did not do.
type Summary struct {
	Calls    int
	Failures int
	// Skipped is the number of planned calls not made because the budget ended.
	Skipped int
	Model   string
	// InputTokens and OutputTokens are the measured usage the provider reported.
	InputTokens  int
	OutputTokens int
}

// Plan is the number of calls a run would make, for the dry run.
func Plan(cases []Case, repetitions int) int {
	return len(cases) * max(repetitions, 1)
}

// Run evaluates every case Repetitions times, sequentially and in a stable
// order, under a call budget. It stops cleanly when the budget is spent.
func Run(ctx context.Context, cases []Case, judge Judge, opts RunOptions) (Summary, error) {
	ordered := append([]Case(nil), cases...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	var summary Summary
	first := max(opts.StartRun, 1)
	for run := first; run < first+max(opts.Repetitions, 1); run++ {
		if err := runRound(ctx, ordered, run, judge, opts, &summary); err != nil {
			return summary, err
		}
	}
	return summary, nil
}

func runRound(ctx context.Context, ordered []Case, run int, judge Judge, opts RunOptions, summary *Summary) error {
	for _, c := range ordered {
		if opts.Budget > 0 && summary.Calls >= opts.Budget {
			summary.Skipped++
			continue
		}
		if err := observeOne(ctx, c, run, judge, opts, summary); err != nil {
			return err
		}
	}
	return nil
}

func observeOne(ctx context.Context, c Case, run int, judge Judge, opts RunOptions, summary *Summary) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("evaluation interrupted: %w", err)
	}
	return observe(ctx, c, run, judge, opts, summary)
}

func observe(ctx context.Context, c Case, run int, judge Judge, opts RunOptions, summary *Summary) error {
	verdict := judge.Judge(ctx, c.Boundary, c.Fields)
	summary.Calls++
	if verdict.State != "" {
		summary.Failures++
	}
	if verdict.Model != "" {
		summary.Model = verdict.Model
	}
	summary.InputTokens += verdict.InputTokens
	summary.OutputTokens += verdict.OutputTokens
	if opts.Sink == nil {
		return nil
	}
	obs := newObservation(c, run, verdict, opts)
	if opts.Local != nil {
		obs.Local, obs.LocalFlags = true, opts.Local(c.Boundary, c.Fields)
	}
	return opts.Sink(obs)
}

func newObservation(c Case, run int, v Verdict, opts RunOptions) Observation {
	obs := Observation{
		CaseID: c.ID, ParentID: c.ParentID, Kind: c.Kind, Operator: c.Operator, Stage: c.Stage, Band: c.Band, Level: c.Level,
		Expect: c.Expect, Golden: c.Golden, HeldOut: c.HeldOut, Session: opts.Session, Run: run, Decision: v.Decision,
		Probabilities: v.Probabilities, PAccept: v.PAccept, Scores: v.Scores, Confidence: v.Confidence, Model: v.Model,
		Profile: opts.Profile, ProfileVersion: opts.ProfileVersion, LatencyMS: v.Latency.Milliseconds(), State: v.State, InputTokens: v.InputTokens, OutputTokens: v.OutputTokens,
	}
	if opts.Now != nil {
		obs.At = opts.Now().UTC().Format(time.RFC3339)
	}
	return obs
}
