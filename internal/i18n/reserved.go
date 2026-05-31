// Package i18n provides wizard string bundles and bilingual protocol constants.
package i18n

// PT-BR reserved words used as agent match tokens and YAML config values.
// These are NOT user-visible strings — they are part of the bilingual protocol
// between the user, active.yaml, and the agent. Do not translate.
const (
	// MissionMode values — written to active.yaml, matched by the agent.
	ReservedMissionModeAnalysis        = "analise"
	ReservedMissionModeRevisedDelivery = "entrega_revisada"
	ReservedMissionModeExecDelivery    = "entrega_executada"
	ReservedDoneScopeDelivery          = "entrega"

	// Gate responses — typed by the user, matched by the agent.
	ReservedGateYes = "sim"
	ReservedGateNo  = "nao"

	// Quick draw triggers — typed by the user, matched by the agent.
	ReservedQuickDrawPT = "saque rapido"
	ReservedQuickDrawEN = "quick draw"
)
