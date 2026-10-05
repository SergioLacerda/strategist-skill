package dojo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pairCase() domain.DojoCriteria {
	cal := policyCase()
	cal.Layer, cal.Boundary = "pair", calibration.BoundaryPairName
	return criteriaFor("pairs", cal)
}

func TestPairFixturesRoundTripAndAreNeverOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "case", "pairs.yaml")
	pairs := calibration.SyntheticPairs(3)
	require.NoError(t, dojo.WritePairs(path, pairs))
	loaded, err := dojo.LoadPairs(path)
	require.NoError(t, err)
	assert.Equal(t, calibration.SortedPairs(pairs), loaded)
	require.Error(t, dojo.WritePairs(path, pairs), "an existing fixtures file is never replaced")

	counts := dojo.PairCounts(loaded)
	assert.Equal(t, 3, counts[calibration.FamilyFaithful])
	assert.Contains(t, dojo.PairFamilies(loaded), calibration.FamilyWeakened)
}

func TestInvalidPairFixturesAreRefusedNotSkipped(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
		return p
	}
	_, err := dojo.LoadPairs(filepath.Join(dir, "missing.yaml"))
	require.Error(t, err)
	_, err = dojo.LoadPairs(write("empty.yaml", "pairs: []\n"))
	require.ErrorContains(t, err, "no pairs")
	_, err = dojo.LoadPairs(write("bad.yaml", "pairs: [\n"))
	require.Error(t, err)
	_, err = dojo.LoadPairs(write("invalid.yaml", "pairs:\n  - id: a\n    parent: p\n    family: f\n"))
	require.Error(t, err)

	good := calibration.SyntheticPairs(1)
	dup := append(append([]calibration.Pair(nil), good...), good[0])
	p := filepath.Join(dir, "dup.yaml")
	require.NoError(t, dojo.WritePairs(p, dup))
	_, err = dojo.LoadPairs(p)
	require.ErrorContains(t, err, "duplicate pair id")
}

func TestAPairCheckWithoutObservationsSaysWhere(t *testing.T) {
	w := newWorkspace(t)
	paths, err := dojo.NewCasePaths(w.base, "pairs", "")
	require.NoError(t, err)
	check, err := dojo.CheckPairs(pairCase(), paths, "latest")
	require.NoError(t, err)
	require.Len(t, check.Result.Items, 1)
	assert.False(t, check.Result.Passed())
	assert.Equal(t, []dojo.FailureReason{dojo.FailureNotCalibrated}, dojo.ClassifyFailures(check.Result.Items))
}

func TestAPairRunKeepsTextFreeObservationsAndEvaluatesTheSession(t *testing.T) {
	w := newWorkspace(t)
	paths, err := dojo.NewCasePaths(w.base, "pairs", "")
	require.NoError(t, err)
	judge := calibration.FuncPairJudge(func(p calibration.Pair) calibration.PairVerdict {
		scores := map[string]float64{}
		for _, d := range calibration.PairDimensions() {
			scores[d] = 0.1
			if p.Oracle[d] {
				scores[d] = 0.9
			}
		}
		return calibration.PairVerdict{Scores: scores, Model: "scripted-1"}
	})

	out, err := dojo.RunPairs(t.Context(), dojo.PairRunInput{Paths: paths, Criteria: pairCase(), Judge: judge, Pairs: calibration.SyntheticPairs(12), Profile: "atomic-v1", ProfileVersion: "abcd1234", Repetitions: 2}, "s-test")

	require.NoError(t, err)
	assert.True(t, out.Check.Result.Passed(), "%+v", out.Check.Result.Items)
	assert.Equal(t, "scripted-1", out.Check.Model)
	raw, readErr := os.ReadFile(paths.PairObservationsPath)
	require.NoError(t, readErr)
	assert.NotContains(t, string(raw), "consumers")
	observed, err := dojo.ReadPairObservations(paths.PairObservationsPath)
	require.NoError(t, err)
	assert.Len(t, observed, 144)

	require.NoError(t, os.WriteFile(paths.PairObservationsPath, append(raw, []byte("not json\n")...), 0o644))
	_, err = dojo.ReadPairObservations(paths.PairObservationsPath)
	require.ErrorContains(t, err, "malformed")
}

func TestThePairLayerAndBoundaryAreValid(t *testing.T) {
	require.NoError(t, dojo.ValidateCriteria(pairCase()))
	bad := pairCase()
	bad.Calibration.Boundary = "other"
	require.Error(t, dojo.ValidateCriteria(bad))
}
