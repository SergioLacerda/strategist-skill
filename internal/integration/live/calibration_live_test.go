//go:build live

package live

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/stretchr/testify/require"
)

func envInt(name string, fallback int) int {
	if value, err := strconv.Atoi(os.Getenv(name)); err == nil && value > 0 {
		return value
	}
	return fallback
}

// TestLiveCalibrationPilotOnSyntheticPackages produces the first calibration
// record against the real provider. It sends only synthetic packages, so it
// needs no consent, and it reports measured values only: no threshold is claimed
// unless the fitness gates, the sample size and the held-out check all pass.
func TestLiveCalibrationPilotOnSyntheticPackages(t *testing.T) {
	r := newRig(t)
	consumer, _ := r.consumer(t)
	judge := handoffconsumer.Judge{Mechanism: consumer.Mechanism, Provider: r.provider, Name: "jev", Version: "1"}

	base := t.TempDir()
	golden := make([]string, 0, len(dojo.SyntheticTopics))
	_, err := dojo.WriteSyntheticPackages(filepath.Join(base, "done"), len(dojo.SyntheticTopics))
	require.NoError(t, err)
	cal := domain.DojoCalibration{Boundary: calibration.BoundaryPackage, Seed: 1, Repetitions: envInt("LIVE_CALIBRATION_REPETITIONS", 2),
		RiskPolicy: domain.DojoRiskPolicy{MaxFalseApproval: 0.05, MaxFalseRejection: 0.15}, Corpus: domain.DojoCorpus{HeldOutPercent: 25}}
	corpus, err := dojo.BuildCorpus(base, cal, handoffconsumer.Reader{})
	require.NoError(t, err)
	for _, e := range corpus.Manifest.Entries {
		raw, marshalErr := json.Marshal(dojo.Promotion{Digest: e.Digest, Package: e.Name, ReviewedBy: "live-pilot", Oracle: "synthetic fixture, well formed by construction"})
		require.NoError(t, marshalErr)
		golden = append(golden, string(raw))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(base, "dojo", dojo.DirCorpus), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(base, "dojo", dojo.DirCorpus, "golden.jsonl"), []byte(strings.Join(golden, "\n")+"\n"), 0o644))
	corpus, err = dojo.BuildCorpus(base, cal, handoffconsumer.Reader{})
	require.NoError(t, err)

	paths, err := dojo.NewCasePaths(base, "live-pilot", "")
	require.NoError(t, err)
	out, err := dojo.RunCalibration(context.Background(), dojo.RunInput{
		BasePath: base, Criteria: domain.DojoCriteria{Scenario: "live-pilot", Kind: domain.KindJudgeCalibration, Calibration: &cal}, Paths: paths, Corpus: corpus,
		Judge: judge, Profile: handoffconsumer.ProfileID, ProfileVersion: handoffconsumer.ProfileVersion(calibration.BoundaryPackage),
		Repetitions: cal.Repetitions, Budget: envInt("LIVE_CALIBRATION_BUDGET", 300), Current: 0.80, Now: time.Now(),
	})
	require.NoError(t, err)
	reportMeasured(t, out)

	raw, err := os.ReadFile(paths.ObservationsPath)
	require.NoError(t, err)
	require.False(t, r.leaks(string(raw)), "observations never carry the credential")
	require.NotContains(t, string(raw), "docs/dojo", "observations never carry package content")
	require.NotNil(t, out.Record)
	require.NoError(t, out.Record.Validate())
}

func reportMeasured(t *testing.T, out dojo.RunOutcome) {
	t.Helper()
	r := out.Check.Report
	t.Logf("calls=%d failures=%d skipped=%d model=%s", out.Summary.Calls, out.Summary.Failures, out.Summary.Skipped, out.Check.Model)
	t.Logf("status=%s failing_gate=%q notes=%v", r.Status, r.FailingGate, r.Notes)
	t.Logf("counts=%+v auc=%.3f ece=%.3f brier=%.3f", r.Counts, r.AUC, r.ECE, r.Brier)
	for _, gate := range r.Gates {
		t.Logf("gate %s: %s value=%.3f limit=%.3f (%s)", gate.Name, gate.Status, gate.Value, gate.Limit, gate.Detail)
	}
	at := r.AtCurrent
	t.Logf("at the default threshold %.2f: false approval %d/%d (%.1f%%, 95%% %.1f-%.1f%%), false rejection %d/%d (%.1f%%)", at.Threshold,
		at.FalseApproval.K, at.FalseApproval.N, 100*at.FalseApproval.Value, 100*at.FalseApproval.Low, 100*at.FalseApproval.High, at.FalseRejection.K, at.FalseRejection.N, 100*at.FalseRejection.Value)
	means := meanScores(out.Check.Observations)
	for _, op := range r.Operators {
		t.Logf("operator %-42s band=%-5s responded %d/%d insensitive=%t mean P(accept) %.2f (golden parents %.2f)", op.Operator, op.Band, op.Sensitivity.K, op.Sensitivity.N, op.Insensitive(), means[op.Operator], means[""])
	}
}

// meanScores is the mean probability of accept per operator; the key "" holds the
// golden parents. It prints distributions, never content.
func meanScores(observations []calibration.Observation) map[string]float64 {
	sums, counts := map[string]float64{}, map[string]int{}
	for _, stat := range calibration.Stats(observations, calibration.Filter{}) {
		if stat.Answered() == 0 {
			continue
		}
		key := stat.Operator
		sums[key] += stat.Score()
		counts[key]++
	}
	out := map[string]float64{}
	for key, sum := range sums {
		out[key] = sum / float64(counts[key])
	}
	return out
}
