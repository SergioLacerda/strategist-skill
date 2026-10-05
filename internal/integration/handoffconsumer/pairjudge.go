package handoffconsumer

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
)

// PairProfileID names the experimental atomic question profile: one noul per
// dimension of a requirement-to-task pair, each about one relation between two
// named parts of the state. The pre-check does not use it; only the DOJO pair
// layer does, to measure whether such questions are answered reliably.
const PairProfileID = "atomic-v1"

// PairQuestions returns the atomic questions, in the order of PairDimensions.
func PairQuestions() []integration.Question {
	return []integration.Question{
		{Key: calibration.DimKind, Kind: integration.KindNoul, Instructions: "Does `target.declared_kind` correctly classify the work described in `target.action`? " +
			"`documentation` means writing or editing documentation only; `implementation` means changing executable behavior or code. Yes when the declared kind matches the action; no when it does not."},
		{Key: calibration.DimObligation, Kind: integration.KindNoul, Instructions: "Does `target.statement` retain the obligation stated in `requirement.text` as mandatory, without making compliance optional, conditional or best effort? " +
			"Yes when the same obligation remains mandatory; no when it is omitted, contradicted or weakened."},
		{Key: calibration.DimVerification, Kind: integration.KindNoul, Instructions: "Does `target.verification` check the specific behavior described in `target.action`? " +
			"Yes when carrying out the verification would show that this behavior holds; no when it checks something unrelated, is vague, or is absent."},
	}
}

// PairProfileVersion is a digest of the exact atomic questions.
func PairProfileVersion() string {
	raw, err := json.Marshal(PairQuestions())
	if err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))[:8]
}

// PairJudge asks the atomic questions of one pair through the Integration
// Mechanism and reports the provider's own noul probabilities, dimension by dimension.
type PairJudge struct {
	Mechanism *mechanism.Mechanism
	Provider  config.Provider
	Name      string
	Version   string
	Now       func() time.Time
}

func (j PairJudge) binding() integration.Binding {
	return integration.Binding{Provider: j.Name, Version: j.Version, Model: j.Provider.Model, Consumer: "handoff", Capability: integration.CapHandoffValidate}
}

// JudgePair projects the pair under the state size limit and asks the provider.
func (j PairJudge) JudgePair(ctx context.Context, pair calibration.Pair) calibration.PairVerdict {
	state := pair.State()
	if len(state) > j.Provider.DataPolicy.MaxStateBytes {
		return calibration.PairVerdict{State: string(integration.StateDataPolicyDenied)}
	}
	started := j.now()
	outcome := j.Mechanism.Call(ctx, j.binding(), integration.Request{
		Correlation: integration.Correlation{MissionID: "dojo-pair-lab", Consumer: "handoff"},
		Capability:  integration.CapHandoffValidate, State: state, Questions: PairQuestions(),
	})
	return pairVerdictOf(outcome, j.now().Sub(started))
}

func pairVerdictOf(outcome mechanism.Outcome, latency time.Duration) calibration.PairVerdict {
	if outcome.Result == nil {
		state := outcome.Decision.Reason
		if state == "" {
			state = integration.StateUnavailable
		}
		return calibration.PairVerdict{State: string(state)}
	}
	scores := map[string]float64{}
	for _, q := range PairQuestions() {
		answer, ok := outcome.Result.Answers[q.Key]
		if !ok || answer.Kind != integration.KindNoul {
			return calibration.PairVerdict{State: string(integration.StateInvalidResponse), Model: outcome.Result.Model}
		}
		scores[q.Key] = answer.Noul
	}
	return calibration.PairVerdict{Scores: scores, Model: outcome.Result.Model, Latency: latency,
		InputTokens: outcome.Result.Usage.InputTokens, OutputTokens: outcome.Result.Usage.OutputTokens}
}

func (j PairJudge) now() time.Time {
	if j.Now != nil {
		return j.Now()
	}
	return time.Now()
}
