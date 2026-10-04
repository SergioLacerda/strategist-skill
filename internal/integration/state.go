package integration

// State is the closed vocabulary of integration outcomes and diagnoses.
type State string

// Integration states. Every failure path maps to exactly one of them.
const (
	StateDisabled              State = "disabled"
	StateCredentialMissing     State = "credential_missing" //nolint:gosec // G101: a state name, not a credential
	StateAuthenticationFailed  State = "authentication_failed"
	StateForbidden             State = "forbidden"
	StateUnsupportedCapability State = "unsupported_capability"
	StateIncompatibleContract  State = "incompatible_contract"
	StateTimeout               State = "timeout"
	StateUnavailable           State = "unavailable"
	StateRateLimited           State = "rate_limited"
	StateInvalidResponse       State = "invalid_response"
	StateBindingIntegrity      State = "binding_integrity_failed"
	StateDataPolicyDenied      State = "data_policy_denied"
)

// States lists every state in a stable order.
func States() []State {
	return []State{
		StateDisabled, StateCredentialMissing, StateAuthenticationFailed, StateForbidden,
		StateUnsupportedCapability, StateIncompatibleContract, StateTimeout, StateUnavailable,
		StateRateLimited, StateInvalidResponse, StateBindingIntegrity, StateDataPolicyDenied,
	}
}
