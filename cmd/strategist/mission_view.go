package main

import "github.com/SergioLacerda/strategist-skill/internal/leveling"

func filterMissionLevelRecords(records []leveling.Record, mission string) []leveling.Record {
	out := records[:0:0]
	for _, record := range records {
		if record.MissionID == mission {
			out = append(out, record)
		}
	}
	return out
}
