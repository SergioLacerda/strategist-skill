package mission

import "github.com/SergioLacerda/strategist-skill/internal/feats/initiative"

// InitiativeRoleEntry is the input for the current Scout role-entry
// consultation. INITIATIVE never resolves or changes LEVELING facts.
type InitiativeRoleEntry struct {
	MissionID string
	Role      string
	RunID     string
	Observed  initiative.Observation
	Leveling  *initiative.LevelingResolution
}
