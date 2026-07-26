// Package i18n is the only surface where localized (pt-BR) strings are allowed:
// runtime mission messages, phase announcements, and wizard install strings.
// Canonical persona YAML source stays English; pt-BR values are injected into
// content_by_lang.pt-BR and phase_announcements.pt-BR at compile time only.
package i18n

// PT-BR reserved words used as agent match tokens and YAML config values.
// These are protocol tokens, not translated display copy — they are part of
// the bilingual protocol between the user, active.yaml, and the agent. Keep
// them centralized here so localization checks can distinguish protocol
// tokens from translatable text.
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

// ReservedGateTokensPTBR returns Portuguese gate tokens accepted by the runtime protocol.
func ReservedGateTokensPTBR() []string {
	return []string{
		ReservedGateYes,
		ReservedGateNo,
		ReservedGateAccept,
		ReservedGateRevisionRequested,
		ReservedGateReject,
	}
}

// ReservedQuickDrawTokens returns Quick Draw trigger tokens accepted by the runtime protocol.
func ReservedQuickDrawTokens() []string {
	return []string{
		ReservedQuickDrawPT,
		ReservedQuickDrawEN,
	}
}
