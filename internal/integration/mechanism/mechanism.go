// Package mechanism is the single Integration Mechanism: for every call it
// re-evaluates binding, enablement, capability, credential, data policy and
// provider health, picks the path deterministically (JEV preferred, main as the
// fallback) and records why. It grants no authority: a provider answer is data
// that a consumer's local rule may use, and nothing here decides a Gate.
package mechanism

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/policy"
)

// Options wires the mechanism. Breaker may be nil to disable circuit breaking.
type Options struct {
	Config   config.File
	Registry *integration.Registry
	Breaker  *policy.Breaker
	Env      credential.Env
}

// Mechanism serves integration calls.
type Mechanism struct {
	opts Options
}

// New builds the mechanism.
func New(opts Options) *Mechanism { return &Mechanism{opts: opts} }

// Outcome is the auditable result of one call. Result is set only when the
// preferred path served the call and its answer validated. Err is the typed
// reason when it did not (nil when the call was served or the provider was
// simply disabled).
type Outcome struct {
	Decision  policy.Decision
	Result    *integration.Result
	Attempted bool
	Err       error
}

// Call evaluates the preconditions for this attempt and, when the preferred
// path is allowed, calls the adapter once. Any failure after that selects the
// fallback path; nothing is retried here.
func (m *Mechanism) Call(ctx context.Context, binding integration.Binding, request integration.Request) Outcome {
	a := &attempt{m: m, binding: binding, request: request}
	state := policy.Evaluate(a.checkEnabled, a.checkBinding, a.checkCapability, a.checkCredential, a.checkDataPolicy, a.checkBreaker)
	decision := policy.Select(state, binding)
	if decision.Effective != policy.PathJEV {
		return Outcome{Decision: decision, Err: a.err}
	}
	return m.serve(ctx, a)
}

func (m *Mechanism) serve(ctx context.Context, a *attempt) Outcome {
	result, err := a.adapter.Call(ctx, a.request)
	if m.opts.Breaker != nil {
		m.opts.Breaker.Record(err)
	}
	if err != nil {
		state, ok := integration.StateOf(err)
		if !ok {
			state = integration.StateUnavailable
		}
		return Outcome{Decision: policy.Select(state, a.binding), Attempted: true, Err: err}
	}
	resolved := a.binding
	resolved.Model = result.Model
	return Outcome{Decision: policy.Select("", resolved), Result: &result, Attempted: true}
}
