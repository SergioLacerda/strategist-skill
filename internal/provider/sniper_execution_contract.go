package provider

// SniperExecutionContract adapts the native Sniper Role to one current-host
// execution. The payload remains authoritative; this contract narrows the
// invocation to the package accepted by the mission engine.
const SniperExecutionContract = "Execute the compiled Sniper payload once through the current-host adapter for this mission using only the accepted refined package. " +
	"Materialize only explicit documentation_target paths, never source or Git changes. Follow the claim protocol, append one materialization ledger record per target, check each completed task, write the archived report, and finish with mission_status documentation_applied. Return only the Sniper completion signal."

// SniperOutputContract is the exact success signal verified by mission complete.
const SniperOutputContract = "Return: sniper: done | report_path: <path> | mission_status: documentation_applied"
