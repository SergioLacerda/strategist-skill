// Package i18n provides wizard string bundles and bilingual protocol constants.
package i18n

// PT-BR reserved words used as agent match tokens and YAML config values.
// These are NOT user-visible strings — they are part of the bilingual protocol
// between the user, active.yaml, and the agent. Do not translate.
const (
	// Gate responses — typed by the user, matched by the agent.
	ReservedGateYes               = "sim"
	ReservedGateNo                = "nao"
	ReservedGateAccept            = "concordo"
	ReservedGateRevisionRequested = "faltou"
	ReservedGateReject            = "pedi_outra_coisa"

	// Quick draw triggers — typed by the user, matched by the agent.
	ReservedQuickDrawPT = "saque rapido"
	ReservedQuickDrawEN = "quick draw"
)
