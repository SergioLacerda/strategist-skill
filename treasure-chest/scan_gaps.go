package treasure

import (
	"sort"
	"strings"
)

// BuildGaps returns still-pending side quests across scanned missions.
func BuildGaps(missions []ScannedMission) []Gap {
	byID := make(map[string]*Gap)
	var order []string
	for _, m := range missions {
		for _, sq := range m.SQs {
			order = appendPendingGap(byID, order, m.MissionID, sq)
		}
	}
	return sortedGaps(byID, order)
}

func appendPendingGap(byID map[string]*Gap, order []string, missionID string, sq SQEntry) []string {
	if sq.Status != "sq_pending" {
		return order
	}
	id := GapID(sq.ID)
	if g, ok := byID[id]; ok {
		g.CitedMissions = append(g.CitedMissions, missionID)
		return order
	}
	byID[id] = &Gap{
		ID:            id,
		CitedMissions: []string{missionID},
		Status:        sq.Status,
		Dependencies:  sq.Dependencies,
		GeneratedAt:   nowISO(),
	}
	return append(order, id)
}

func sortedGaps(byID map[string]*Gap, order []string) []Gap {
	sort.Strings(order)
	gaps := make([]Gap, 0, len(order))
	for _, id := range order {
		gaps = append(gaps, *byID[id])
	}
	return gaps
}

// GapID normalizes a side-quest id for gap artifact filenames and ids.
func GapID(sqID string) string {
	return strings.ToLower(sqID)
}
