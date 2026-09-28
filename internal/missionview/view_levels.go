package missionview

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
)

func buildJourney(reg domain.RoleRegistry, providers map[string]string) []JourneyEntry {
	roles := reg.Roles()
	out := make([]JourneyEntry, 0, len(roles)+1)
	for _, role := range roles {
		if phase, ok := reg.PhaseOf("gate"); ok && role.Phase == phase+1 {
			out = append(out, JourneyEntry{Kind: "gate", ID: "gate", Phase: phase})
		}
		entry := JourneyEntry{Kind: "role", ID: role.ID, Phase: role.Phase}
		if role.Slot != "" {
			entry.Provider = providers[role.Slot]
		}
		out = append(out, entry)
	}
	return out
}

func buildLevels(reg domain.RoleRegistry, records []leveling.Record, run string) LevelingSection {
	byRole := groupLevelRecords(records)
	section := LevelingSection{Availability: Available, Selection: selection(run)}
	for _, role := range reg.Roles() {
		section.Roles = append(section.Roles, levelRole(role.ID, byRole[role.ID], run))
	}
	// The ledger was readable but holds nothing for this mission: every role
	// is unresolved, which is not the same as "available".
	if len(records) == 0 {
		section.Availability = Unknown
	}
	return section
}

func groupLevelRecords(records []leveling.Record) map[string][]leveling.Record {
	byRole := make(map[string][]leveling.Record)
	for _, record := range records {
		byRole[record.Role] = append(byRole[record.Role], record)
	}
	return byRole
}

func levelRole(role string, records []leveling.Record, run string) LevelingRole {
	history := append([]leveling.Record(nil), records...)
	return LevelingRole{Role: role, History: history, Effective: effectiveRecord(history, run)}
}

func effectiveRecord(records []leveling.Record, run string) *leveling.Record {
	for i := len(records) - 1; i >= 0; i-- {
		if run == "" || records[i].Run == run {
			record := records[i]
			return &record
		}
	}
	return nil
}
