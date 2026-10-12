package calibration

import (
	"context"
	"fmt"
	"time"
)

// PairVerdict is what a judge returned for one pair: the provider's own
// probability that each dimension holds, or a typed failure.
type PairVerdict struct {
	Scores       map[string]float64
	Model        string
	Latency      time.Duration
	InputTokens  int
	OutputTokens int
	State        string
}

// PairJudge is the port of a system evaluated on requirement-to-task pairs.
type PairJudge interface {
	JudgePair(ctx context.Context, pair Pair) PairVerdict
}

// FuncPairJudge adapts a function to the port, for scripted judges and tests.
type FuncPairJudge func(Pair) PairVerdict

// JudgePair calls the function.
func (f FuncPairJudge) JudgePair(_ context.Context, p Pair) PairVerdict { return f(p) }

// PairObservation is one judge call about one pair. It carries identifiers, the
// pair's oracle and the numbers returned, and no text of the pair.
type PairObservation struct {
	Pair           string             `json:"pair"`
	Parent         string             `json:"parent"`
	Family         string             `json:"family"`
	HeldOut        bool               `json:"held_out,omitempty"`
	Session        string             `json:"session,omitempty"`
	Run            int                `json:"run"`
	Oracle         map[string]bool    `json:"oracle"`
	Scores         map[string]float64 `json:"scores,omitempty"`
	Model          string             `json:"model,omitempty"`
	Profile        string             `json:"profile,omitempty"`
	ProfileVersion string             `json:"profile_version,omitempty"`
	LatencyMS      int64              `json:"latency_ms,omitempty"`
	InputTokens    int                `json:"input_tokens,omitempty"`
	OutputTokens   int                `json:"output_tokens,omitempty"`
	State          string             `json:"state,omitempty"`
}

// Answered reports whether the call produced usable scores for every dimension of its oracle.
func (o PairObservation) Answered() bool {
	if o.State != "" {
		return false
	}
	for dim := range o.Oracle {
		if _, ok := o.Scores[dim]; !ok {
			return false
		}
	}
	return len(o.Scores) > 0
}

// PairRunOptions bound one run over pairs.
type PairRunOptions struct {
	Repetitions    int
	Budget         int
	Profile        string
	ProfileVersion string
	Session        string
	HeldOut        func(parent string) bool
	Sink           func(PairObservation) error
}

// RunPairs evaluates every pair Repetitions times, sequentially and in a stable
// order, under a call budget.
func RunPairs(ctx context.Context, pairs []Pair, judge PairJudge, opts PairRunOptions) (Summary, error) {
	var summary Summary
	ordered := SortedPairs(pairs)
	for run := 1; run <= max(opts.Repetitions, 1); run++ {
		if err := pairRound(ctx, ordered, run, judge, opts, &summary); err != nil {
			return summary, err
		}
	}
	return summary, nil
}

func pairRound(ctx context.Context, ordered []Pair, run int, judge PairJudge, opts PairRunOptions, summary *Summary) error {
	for _, p := range ordered {
		if opts.Budget > 0 && summary.Calls >= opts.Budget {
			summary.Skipped++
			continue
		}
		if err := observePairChecked(ctx, p, run, judge, opts, summary); err != nil {
			return err
		}
	}
	return nil
}

func observePairChecked(ctx context.Context, p Pair, run int, judge PairJudge, opts PairRunOptions, summary *Summary) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("evaluation interrupted: %w", err)
	}
	return observePair(ctx, p, run, judge, opts, summary)
}

func observePair(ctx context.Context, p Pair, run int, judge PairJudge, opts PairRunOptions, summary *Summary) error {
	v := judge.JudgePair(ctx, p)
	summary.Calls++
	summary.InputTokens += v.InputTokens
	summary.OutputTokens += v.OutputTokens
	if v.State != "" {
		summary.Failures++
	}
	if v.Model != "" {
		summary.Model = v.Model
	}
	if opts.Sink == nil {
		return nil
	}
	obs := PairObservation{Pair: p.ID, Parent: p.Parent, Family: p.Family, Session: opts.Session, Run: run, Oracle: p.Oracle, Scores: v.Scores, Model: v.Model,
		Profile: opts.Profile, ProfileVersion: opts.ProfileVersion, LatencyMS: v.Latency.Milliseconds(), InputTokens: v.InputTokens, OutputTokens: v.OutputTokens, State: v.State}
	if opts.HeldOut != nil {
		obs.HeldOut = opts.HeldOut(p.Parent)
	}
	return opts.Sink(obs)
}
