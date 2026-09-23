package telemetry

import "path/filepath"

// InitiativeEventHistoryRelPath is the diagnostic event stream for
// INITIATIVE. It is separate from both the LEVELING ledger and the
// initiative advice/result ledger.
const InitiativeEventHistoryRelPath = "memory/initiative-events.jsonl"

// InitiativeEventHistoryPath returns the INITIATIVE telemetry path under a
// Strategist runtime root.
func InitiativeEventHistoryPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(InitiativeEventHistoryRelPath))
}
