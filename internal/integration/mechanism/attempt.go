package mechanism

import (
	"slices"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
)

// attempt holds the state of one call's preconditions. Each check returns the
// state that fails it, or the empty state, and records the typed error.
type attempt struct {
	m       *Mechanism
	binding integration.Binding
	request integration.Request
	adapter integration.Adapter
	err     error
}

func (a *attempt) correlation() integration.Correlation {
	correlation := a.request.Correlation
	correlation.Provider = a.binding.Provider
	return correlation
}

func (a *attempt) fail(state integration.State, reason string) integration.State {
	a.err = integration.NewError(state, a.correlation(), reason)
	return state
}

func (a *attempt) provider() (config.Provider, bool) {
	provider, ok := a.m.opts.Config.Providers[a.binding.Provider]
	return provider, ok
}

func (a *attempt) checkBinding() integration.State {
	if a.m.opts.Registry == nil {
		return a.fail(integration.StateBindingIntegrity, "no adapter registry is configured")
	}
	adapter, err := a.m.opts.Registry.Resolve(a.binding)
	if err != nil {
		state, _ := integration.StateOf(err)
		a.err = integration.WithCorrelation(err, a.correlation())
		return state
	}
	a.adapter = adapter
	return ""
}

func (a *attempt) checkEnabled() integration.State {
	if provider, ok := a.provider(); !ok || !provider.Enabled {
		return integration.StateDisabled
	}
	return ""
}

func (a *attempt) checkCapability() integration.State {
	provider, _ := a.provider()
	if !slices.Contains(provider.AllowedCapabilities, a.binding.Capability) {
		return a.fail(integration.StateUnsupportedCapability, "capability is not allowed by the operator configuration")
	}
	return ""
}

func (a *attempt) checkCredential() integration.State {
	provider, _ := a.provider()
	if _, err := credential.Resolve(provider.CredentialRef, a.m.opts.Env); err != nil {
		a.err = integration.WithCorrelation(err, a.correlation())
		return integration.StateCredentialMissing
	}
	return ""
}

func (a *attempt) checkDataPolicy() integration.State {
	provider, _ := a.provider()
	if len(a.request.State) > provider.DataPolicy.MaxStateBytes {
		return a.fail(integration.StateDataPolicyDenied, "state exceeds the data policy size limit")
	}
	return ""
}

func (a *attempt) checkBreaker() integration.State {
	if a.m.opts.Breaker == nil {
		return ""
	}
	if state := a.m.opts.Breaker.Check(); state != "" {
		return a.fail(state, "circuit is open after repeated provider failures")
	}
	return ""
}
