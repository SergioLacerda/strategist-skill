package dojo

import (
	"context"
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Plan is what a run would send: the exact numbers of the dry run.
type Plan struct {
	// Bytes is the projected state size per repetition, summed over every case.
	Bytes int
	// Denied counts cases the operator's data policy would not let through.
	Denied int
	// Fields are the contract fields the data policy allows.
	Fields []string
}

// CalibrationDependencies supplies the provider-facing concerns of the
// calibration commands at the composition edge, so this package never imports
// the integration packages.
type CalibrationDependencies struct {
	Reader  internaldojo.FieldReader
	Judge   func(strategistRoot, profile string) (calibration.Judge, error)
	Current func(strategistRoot string) float64
	Profile func(profile, boundary string) (id, version string)
	Plan    func(strategistRoot, profile, boundary string, cases []calibration.Case) (Plan, error)
	Signals func(strategistRoot, missionID string) ([]internaldojo.MissionSignal, error)
	// Policy runs the integration layer: the approval policy against scripted answers, no call.
	Policy func(now time.Time) []domain.DojoCheckItem
	// Endpoint is the provider endpoint a run would call, for the consent term.
	Endpoint func(strategistRoot string) string
	// LivePlan is the number of calls and bytes of the live smoke check (synthetic text only).
	LivePlan func() (calls, bytes int)
	// PairJudge asks the atomic questions of one requirement pair.
	PairJudge func(strategistRoot string) (calibration.PairJudge, error)
	// PairProfile is the atomic question profile id and version.
	PairProfile func() (id, version string)
	// Live runs the live smoke check against the real provider. Only a run with consent reaches it.
	Live func(ctx context.Context, strategistRoot string) ([]domain.DojoCheckItem, error)
}

func (d CalibrationDependencies) ready() error {
	if d.Reader == nil || d.Judge == nil || d.Current == nil || d.Profile == nil || d.Plan == nil || d.Signals == nil || d.Policy == nil || d.Endpoint == nil || d.LivePlan == nil || d.Live == nil || d.PairJudge == nil || d.PairProfile == nil {
		return fmt.Errorf("dojo: calibration dependencies are not configured")
	}
	return nil
}

// caseContext is everything a calibration command resolves before it acts.
type caseContext struct {
	strategistRoot string
	basePath       string
	criteria       domain.DojoCriteria
	paths          internaldojo.CasePaths
}

func loadCase(deps Dependencies, root, name string) (caseContext, error) {
	strategistRoot, basePath, err := resolveRoots(deps, root)
	if err != nil {
		return caseContext{}, err
	}
	if err := deps.Calibration.ready(); err != nil {
		return caseContext{}, err
	}
	scenario, err := internaldojo.NewStoragePaths(basePath, name)
	if err != nil {
		return caseContext{}, fmt.Errorf("dojo: %w", err)
	}
	criteria, err := internaldojo.LoadCriteria(scenario.ScenarioDir)
	if err != nil {
		return caseContext{}, fmt.Errorf("dojo: %w", err)
	}
	if criteria.Kind != domain.KindJudgeCalibration {
		return caseContext{}, fmt.Errorf("dojo: %q is a pipeline scenario, not a %s case", name, domain.KindJudgeCalibration)
	}
	paths, err := internaldojo.NewCasePaths(basePath, name, criteria.Calibration.Output)
	if err != nil {
		return caseContext{}, fmt.Errorf("dojo: %w", err)
	}
	return caseContext{strategistRoot: strategistRoot, basePath: basePath, criteria: criteria, paths: paths}, nil
}
