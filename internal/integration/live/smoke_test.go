package live

import (
	"context"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/probe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func okProbe(context.Context) probe.Record {
	return probe.Record{State: "ok", Model: "jev-1.13.0", InputTokens: 12, OutputTokens: 3, LatencyMS: 400}
}

func scoring(good, poor float64) calibration.Judge {
	return calibration.FuncJudge(func(_ string, f calibration.Fields) calibration.Verdict {
		v := calibration.ScoredVerdict(good)
		if f["objective"] == Poor["objective"] {
			v = calibration.ScoredVerdict(poor)
		}
		v.Latency = 10 * time.Millisecond
		return v
	})
}

func passed(checks []Check) bool {
	for _, c := range checks {
		if !c.Passed {
			return false
		}
	}
	return true
}

func TestSmokePassesAgainstASimulatedProvider(t *testing.T) {
	checks := Smoke(context.Background(), Env{Probe: okProbe, Judge: scoring(0.86, 0.05), Secret: func() string { return "sk-secret" }})
	require.Len(t, checks, Calls+4)
	assert.True(t, passed(checks), "%+v", checks)
}

func TestSmokeNamesWhatFails(t *testing.T) {
	failing := func(context.Context) probe.Record { return probe.Record{State: "authentication_failed"} }
	checks := Smoke(context.Background(), Env{Probe: failing, Judge: scoring(0.2, 0.6)})
	byName := map[string]Check{}
	for _, c := range checks {
		byName[c.Name] = c
	}
	assert.False(t, byName["live probe authenticates and answers in the documented shape"].Passed)
	assert.False(t, byName["live probe reports the model that served the call"].Passed)
	assert.False(t, byName["live precheck scores the vague handoff below the well-formed one"].Passed)
	assert.True(t, byName["live report never contains the credential"].Passed, "an unchecked secret is reported as such")
}

func TestSmokeFailsWhenTheJudgeCannotAnswer(t *testing.T) {
	down := calibration.FuncJudge(func(string, calibration.Fields) calibration.Verdict { return calibration.Verdict{State: "timeout"} })
	checks := Smoke(context.Background(), Env{Probe: okProbe, Judge: down})
	assert.False(t, passed(checks))
	assert.Contains(t, checks[3].Detail, "timeout")
}

func TestSmokeNeverLetsTheCredentialIntoItsReport(t *testing.T) {
	leaky := func(context.Context) probe.Record { return probe.Record{State: "sk-secret", Model: "m"} }
	checks := Smoke(context.Background(), Env{Probe: leaky, Judge: scoring(0.8, 0.1), Secret: func() string { return "sk-secret" }})
	last := checks[len(checks)-1]
	assert.False(t, last.Passed)
	assert.NotContains(t, last.Detail, "sk-secret")
}

func TestTheSmokeSendsOnlySyntheticText(t *testing.T) {
	assert.Positive(t, Bytes())
	assert.Equal(t, 3, Calls)
}
