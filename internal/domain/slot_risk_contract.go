package domain

// SlotRiskContract maps each slot to the risk_score a Weapon must declare to be
// bound to it. It is the single list both `strategist check` (at mission time) and
// `strategist provider add` (at bind time) apply, so what provider add accepts is
// never rejected by check for its risk.
var SlotRiskContract = map[string]string{
	string(SlotDiscovery):  "write_analysis",
	string(SlotRefinement): "write_analysis",
	string(SlotExecution):  "controlled",
}
