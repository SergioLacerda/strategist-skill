package dojo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// pairsFile is the document of a pair-layer case's fixtures.
type pairsFile struct {
	Pairs []calibration.Pair `yaml:"pairs"`
}

// LoadPairs reads and validates the pair fixtures. A duplicate id, an invalid
// pair or a pair that names its own label is refused, never skipped.
func LoadPairs(path string) ([]calibration.Pair, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: the pairs file of a case under the dojo storage domain
	if err != nil {
		return nil, fmt.Errorf("dojo: read pairs %s: %w", path, err)
	}
	var doc pairsFile
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("dojo: parse pairs %s: %w", path, err)
	}
	if err := validatePairs(doc.Pairs); err != nil {
		return nil, fmt.Errorf("dojo: pairs %s: %w", path, err)
	}
	return calibration.SortedPairs(doc.Pairs), nil
}

func validatePairs(pairs []calibration.Pair) error {
	if len(pairs) == 0 {
		return errors.New("no pairs")
	}
	seen := map[string]bool{}
	var problems []error
	for _, p := range pairs {
		if err := p.Validate(); err != nil {
			problems = append(problems, fmt.Errorf("pair %q: %w", p.ID, err))
		}
		if seen[p.ID] {
			problems = append(problems, fmt.Errorf("duplicate pair id %q", p.ID))
		}
		seen[p.ID] = true
	}
	return errors.Join(problems...)
}

// WritePairs writes the fixtures file; it refuses to replace an existing one.
func WritePairs(path string, pairs []calibration.Pair) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("dojo: %s already exists; fixtures are never overwritten", path)
	}
	raw, err := yaml.Marshal(pairsFile{Pairs: pairs})
	if err != nil {
		return fmt.Errorf("dojo: encode pairs: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // G301: dojo storage domain
		return fmt.Errorf("dojo: create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil { //nolint:gosec // G306: dojo storage domain
		return fmt.Errorf("dojo: write %s: %w", path, err)
	}
	return nil
}

// PairHeldOutPercent resolves the final-split share of a pair case.
func PairHeldOutPercent(cal domain.DojoCalibration) int { return HeldOutPercent(cal) }

// PairRunInput is one run of a pair-layer case.
type PairRunInput struct {
	Paths          CasePaths
	Criteria       domain.DojoCriteria
	Judge          calibration.PairJudge
	Pairs          []calibration.Pair
	Profile        string
	ProfileVersion string
	Repetitions    int
	Budget         int
}

// PairOutcome is what a pair run produced.
type PairOutcome struct {
	Summary calibration.Summary
	Session string
	Check   PairCheck
}

// RunPairs runs the pairs, keeps every raw observation and evaluates the session.
func RunPairs(ctx context.Context, in PairRunInput, session string) (PairOutcome, error) {
	cal := *in.Criteria.Calibration
	summary, err := calibration.RunPairs(ctx, in.Pairs, in.Judge, calibration.PairRunOptions{
		Repetitions: in.Repetitions, Budget: in.Budget, Profile: in.Profile, ProfileVersion: in.ProfileVersion, Session: session,
		HeldOut: func(parent string) bool {
			return calibration.PairHeldOut(parent, cal.Corpus.Salt, PairHeldOutPercent(cal))
		},
		Sink: func(o calibration.PairObservation) error { return appendJSONL(in.Paths.PairObservationsPath, o) },
	})
	if err != nil {
		return PairOutcome{Summary: summary, Session: session}, fmt.Errorf("dojo: pair run: %w", err)
	}
	check, err := CheckPairs(in.Criteria, in.Paths, session)
	return PairOutcome{Summary: summary, Session: session, Check: check}, err
}

// PairCounts summarizes the fixtures by family, for the dry run.
func PairCounts(pairs []calibration.Pair) map[string]int {
	out := map[string]int{}
	for _, p := range pairs {
		out[p.Family]++
	}
	return out
}

// PairFamilies lists the families present in sorted order.
func PairFamilies(pairs []calibration.Pair) []string {
	var names []string
	for family := range PairCounts(pairs) {
		names = append(names, family)
	}
	sort.Strings(names)
	return names
}
