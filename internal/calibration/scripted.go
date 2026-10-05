package calibration

import "context"

// FuncJudge adapts a function to the Judge port. It is the scripted judge of the
// integration layer and of tests: its answers are known, so every metric can be
// checked by hand and the Strategist policy can be exercised with no provider.
type FuncJudge func(boundary string, fields Fields) Verdict

// Judge calls the function.
func (f FuncJudge) Judge(_ context.Context, boundary string, fields Fields) Verdict {
	return f(boundary, fields)
}

// ScoredVerdict builds an answered verdict from a probability of accept, with the
// aggregate choice following the usual rule (accept when the probability is at
// least one half).
func ScoredVerdict(pAccept float64) Verdict {
	decision := "reject"
	if pAccept >= 0.5 {
		decision = "accept"
	}
	return Verdict{
		Decision: decision, PAccept: pAccept, Confidence: pAccept, Model: "scripted-1",
		Probabilities: Distribution{"accept": pAccept, "reject": 1 - pAccept},
	}
}
