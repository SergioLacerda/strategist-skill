package dojo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// RunInput is one calibration run: the case, its corpus, the judge and the
// bounds. Resolve defaults before calling.
type RunInput struct {
	BasePath       string
	Criteria       domain.DojoCriteria
	Paths          CasePaths
	Corpus         Corpus
	Judge          calibration.Judge
	Profile        string
	ProfileVersion string
	Repetitions    int
	Budget         int
	// BoundaryMargin, when positive, repeats the cases whose score is within the
	// margin of Current, where the decision is least certain.
	BoundaryMargin float64
	Current        float64
	Now            time.Time
}

// RunOutcome is what a run produced.
type RunOutcome struct {
	Cases   []calibration.Case
	Summary calibration.Summary
	Session string
	Check   CheckOutcome
	Record  *calibration.Record
}

// SessionID names a run by its start time.
func SessionID(now time.Time) string { return "s-" + now.UTC().Format("20060102T150405Z") }

// RunCalibration generates the cases, calls the judge under the budget, keeps
// every raw observation, evaluates the session and publishes a calibration
// record bound to its provenance. Persisting the result and the lesson is left
// to the caller so a persistence failure never changes a verdict.
func RunCalibration(ctx context.Context, in RunInput) (RunOutcome, error) {
	cal := *in.Criteria.Calibration
	cases := BuildCases(in.Corpus, cal)
	if len(cases) == 0 {
		return RunOutcome{}, errors.New("dojo: the corpus produced no cases")
	}
	if err := WriteManifest(in.Paths, in.Corpus.Manifest); err != nil {
		return RunOutcome{}, err
	}
	out := RunOutcome{Cases: cases, Session: SessionID(in.Now)}
	var err error
	if out.Summary, err = execute(ctx, in, cases, out.Session, 1, in.Repetitions, in.Budget); err != nil {
		return out, err
	}
	if err := repeatBoundary(ctx, in, cases, &out); err != nil {
		return out, err
	}
	return finish(in, cal, &out)
}

func execute(ctx context.Context, in RunInput, cases []calibration.Case, session string, start, repetitions, budget int) (calibration.Summary, error) {
	var appendErr error
	summary, err := calibration.Run(ctx, cases, in.Judge, calibration.RunOptions{
		Repetitions: repetitions, Budget: budget, Profile: in.Profile, ProfileVersion: in.ProfileVersion, Session: session, StartRun: start, Now: Now, Local: calibration.LocalFlags,
		Sink: func(obs calibration.Observation) error {
			if appendErr != nil {
				return appendErr
			}
			appendErr = AppendObservation(in.Paths.ObservationsPath, obs)
			return appendErr
		},
	})
	if err != nil {
		return summary, fmt.Errorf("dojo: calibration run: %w", err)
	}
	return summary, nil
}

// Now is the clock of the calibration flows; tests replace it.
var Now = time.Now

func repeatBoundary(ctx context.Context, in RunInput, cases []calibration.Case, out *RunOutcome) error {
	budget, ok := boundaryBudget(in, out.Summary.Calls)
	if in.BoundaryMargin <= 0 || !ok {
		return nil
	}
	subset, err := nearCases(in, cases, out.Session)
	if err != nil || len(subset) == 0 {
		return err
	}
	extra, err := execute(ctx, in, subset, out.Session, in.Repetitions+1, 2, budget)
	out.Summary.Calls += extra.Calls
	out.Summary.Failures += extra.Failures
	return err
}

// boundaryBudget is what is left of the call budget; false when none is left.
func boundaryBudget(in RunInput, used int) (int, bool) {
	if in.Budget <= 0 {
		return 0, true
	}
	left := in.Budget - used
	return left, left > 0
}

func nearCases(in RunInput, cases []calibration.Case, session string) ([]calibration.Case, error) {
	observed, _, err := ReadObservations(in.Paths.ObservationsPath)
	if err != nil {
		return nil, err
	}
	near := map[string]bool{}
	for _, id := range calibration.BoundaryCases(calibration.SessionOf(observed, session), in.Current, in.BoundaryMargin) {
		near[id] = true
	}
	var subset []calibration.Case
	for _, c := range cases {
		if near[c.ID] {
			subset = append(subset, c)
		}
	}
	return subset, nil
}

func finish(in RunInput, cal domain.DojoCalibration, out *RunOutcome) (RunOutcome, error) {
	check, err := CheckCalibration(CheckInput{Criteria: in.Criteria, Paths: in.Paths, Current: in.Current, ProfileVersion: in.ProfileVersion, Session: out.Session})
	if err != nil {
		return *out, err
	}
	out.Check = check
	if check.Model == "" {
		return *out, errors.New("dojo: no call was answered, so there is nothing to calibrate; see the observations for the failing states")
	}
	record, err := calibration.NewRecord(check.Report, calibration.Meta{
		Provider: providerOf(cal), Model: check.Model, Profile: in.Profile, ProfileVersion: in.ProfileVersion, CorpusDigest: in.Corpus.Manifest.Digest,
		CatalogVersion: calibration.CatalogVersion(), Operators: cal.Operators, Seed: cal.Seed, Repetitions: in.Repetitions, Observations: calibration.DigestOf(check.Observations), Now: in.Now,
	})
	if err != nil {
		return *out, fmt.Errorf("dojo: calibration record: %w", err)
	}
	if err := SaveCalibration(in.BasePath, record); err != nil {
		return *out, err
	}
	out.Record = &record
	out.Check.Result.Calibration.RecordID = record.ID
	return *out, nil
}

func providerOf(cal domain.DojoCalibration) string {
	if cal.Provider == "" {
		return "jev"
	}
	return cal.Provider
}
