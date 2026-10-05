package live

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/probe"
)

// WellFormed is a synthetic Ranger handoff that states a concrete objective,
// evidenced facts, explicit uncertainties and an actionable focus. It contains
// nothing of any mission.
var WellFormed = calibration.Fields{
	"objective": "Decide whether the approval screen shows how well each handoff was judged, and whether the confidence figure is carried between handoffs, documents and telemetry.",
	"known_facts": "- F-01: The approval prompt is written by the agent from a contract template; no code renders it. Source: the gate contract.\n" +
		"- F-02: The confidence ledger had 59 records at read time, none for the critic or the mission-quality evaluation. Source: the confidence ledger.\n" +
		"- F-03: The provider confidence is stored only on the handoff outcome and in the call ledger. Source: the handoff outcome type.",
	"uncertainties": "- U-01: Whether the author wants the critic score and the mission-quality result on the same line is not known.\n" +
		"- U-02: Whether other telemetry consumers exist was not exhaustively searched.",
	"recommended_refinement_focus": "Refine three deliverables: the gate display block, a read-only report that feeds it, and the telemetry attributes; keep every code change as an implementation handoff.",
}

// Poor is a synthetic, vague handoff a conformance check should not approve.
var Poor = calibration.Fields{
	"objective":                    "stuff",
	"known_facts":                  "things are fine probably",
	"recommended_refinement_focus": "do it",
}

// Check is one item of the smoke check.
type Check struct {
	Name   string
	Passed bool
	Detail string
}

// Env is what the smoke check needs, injected so it can run against a simulated
// provider with no network.
type Env struct {
	// Probe sends the synthetic probe once.
	Probe func(ctx context.Context) probe.Record
	// Judge asks the handoff pre-check questions of one boundary.
	Judge calibration.Judge
	// Secret returns the credential, only to prove it never appears in a detail.
	// May be nil.
	Secret func() string
}

// Calls is the number of provider calls one smoke check makes: the probe and one
// pre-check per synthetic handoff.
const Calls = 3

// Bytes is the size of the synthetic handoff text one smoke check sends.
func Bytes() int {
	total := 0
	for _, fields := range []calibration.Fields{WellFormed, Poor} {
		for _, value := range fields {
			total += len(value)
		}
	}
	return total + len(probe.SyntheticState)
}

var resolvedModel = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// Smoke runs the provider smoke check: the probe authenticates and answers in the
// documented shape, a well-formed and a vague handoff are each answered, the vague
// one scores lower, and no detail contains the credential. It reports the
// provider's own numbers and claims no threshold.
func Smoke(ctx context.Context, env Env) []Check {
	record := env.Probe(ctx)
	checks := []Check{
		{"live probe authenticates and answers in the documented shape", record.State == "ok", "state " + record.State},
		{"live probe reports the model that served the call", resolvedModel.MatchString(record.Model), "model " + record.Model},
		{"live probe reports usage", record.InputTokens > 0, fmt.Sprintf("input_tokens=%d output_tokens=%d latency_ms=%d", record.InputTokens, record.OutputTokens, record.LatencyMS)},
	}
	good := env.Judge.Judge(ctx, calibration.BoundaryRanger, WellFormed)
	poor := env.Judge.Judge(ctx, calibration.BoundaryRanger, Poor)
	checks = append(checks,
		Check{"live precheck answers a well-formed handoff", good.State == "", answerDetail(good)},
		Check{"live precheck answers a vague handoff", poor.State == "", answerDetail(poor)},
		Check{"live precheck scores the vague handoff below the well-formed one", good.State == "" && poor.State == "" && poor.PAccept < good.PAccept,
			fmt.Sprintf("P(accept) well-formed %.2f, vague %.2f", good.PAccept, poor.PAccept)},
	)
	return append(checks, credentialCheck(env, checks))
}

func answerDetail(v calibration.Verdict) string {
	if v.State != "" {
		return "state " + v.State
	}
	return fmt.Sprintf("decision %s, P(accept) %.2f, model %s, latency %dms", v.Decision, v.PAccept, v.Model, v.Latency.Milliseconds())
}

func credentialCheck(env Env, checks []Check) Check {
	if env.Secret == nil {
		return Check{"live report never contains the credential", true, "not checked: no secret supplied"}
	}
	secret := env.Secret()
	for _, c := range checks {
		if secret != "" && strings.Contains(c.Name+c.Detail, secret) {
			return Check{"live report never contains the credential", false, "a detail contains the credential"}
		}
	}
	return Check{"live report never contains the credential", true, "no detail contains it"}
}
