package treasure

// StatusRow is the merged runtime status for one treasure chest across all truth layers.
type StatusRow struct {
	ID           string
	Path         string
	Scope        []string
	TrustTier    string
	ReviewedBy   string
	LastReviewed string
	Configured   bool
	Governed     bool
	Indexed      bool
	Compiled     bool
	Freshness    string
	Drift        []string
	SourceGrade  string
	ReuseValue   string
	OpenGaps     []string
	JewelCount   int
}

// MergeChestRows merges active, governed, indexed, compiled, and jewel data into status rows.
func MergeChestRows(
	active []ActiveChest,
	governed map[string]GovernedChest,
	indexed map[string]bool,
	compiledIDs map[string]bool,
	jewels map[string][]Jewel,
) []StatusRow {
	seen := make(map[string]bool)
	rows := activeStatusRows(active, governed, indexed, compiledIDs, jewels, seen)
	rows = append(rows, governedOnlyStatusRows(governed, indexed, compiledIDs, jewels, seen)...)
	rows = append(rows, indexedOnlyStatusRows(indexed, compiledIDs, seen)...)
	return rows
}

func activeStatusRows(
	active []ActiveChest,
	governed map[string]GovernedChest,
	indexed map[string]bool,
	compiledIDs map[string]bool,
	jewels map[string][]Jewel,
	seen map[string]bool,
) []StatusRow {
	rows := make([]StatusRow, 0, len(active))
	for _, ac := range active {
		row := activeStatusRow(ac, indexed, compiledIDs, jewels)
		if gc, ok := governed[ac.ID]; ok {
			applyGovernedStatus(&row, gc)
		}
		rows = append(rows, finalizeStatusRow(row))
		seen[ac.ID] = true
	}
	return rows
}

func activeStatusRow(ac ActiveChest, indexed, compiledIDs map[string]bool, jewels map[string][]Jewel) StatusRow {
	return StatusRow{
		ID:         ac.ID,
		Path:       ac.Path,
		Scope:      ac.Scope,
		Configured: true,
		Indexed:    indexed[ac.ID],
		Compiled:   compiledIDs[ac.ID],
		JewelCount: NonDeprecatedJewelCount(jewels[ac.ID]),
	}
}

func governedOnlyStatusRows(
	governed map[string]GovernedChest,
	indexed map[string]bool,
	compiledIDs map[string]bool,
	jewels map[string][]Jewel,
	seen map[string]bool,
) []StatusRow {
	var rows []StatusRow
	for id, gc := range governed {
		if seen[id] {
			continue
		}
		row := governedOnlyStatusRow(id, gc, indexed, compiledIDs, jewels)
		rows = append(rows, finalizeStatusRow(row))
		seen[id] = true
	}
	return rows
}

func governedOnlyStatusRow(
	id string,
	gc GovernedChest,
	indexed map[string]bool,
	compiledIDs map[string]bool,
	jewels map[string][]Jewel,
) StatusRow {
	row := StatusRow{
		ID:         id,
		Path:       gc.Path,
		Indexed:    indexed[id],
		Compiled:   compiledIDs[id],
		JewelCount: NonDeprecatedJewelCount(jewels[id]),
	}
	applyGovernedStatus(&row, gc)
	return row
}

func indexedOnlyStatusRows(indexed, compiledIDs map[string]bool, seen map[string]bool) []StatusRow {
	var rows []StatusRow
	for id := range indexed {
		if seen[id] {
			continue
		}
		row := StatusRow{
			ID:        id,
			Indexed:   true,
			Compiled:  compiledIDs[id],
			Freshness: "unknown",
		}
		rows = append(rows, finalizeStatusRow(row))
		seen[id] = true
	}
	return rows
}

func applyGovernedStatus(row *StatusRow, gc GovernedChest) {
	row.Governed = true
	row.TrustTier = gc.Trust.Tier
	row.ReviewedBy = gc.Trust.ReviewedBy
	row.LastReviewed = gc.Trust.LastReviewed
	row.SourceGrade = gc.Grade.SourceGrade
	row.ReuseValue = gc.Grade.ReuseValue
	row.OpenGaps = gc.OpenGaps
}

func finalizeStatusRow(row StatusRow) StatusRow {
	if row.Freshness == "" {
		row.Freshness = DeriveFreshness(row)
	}
	row.Drift = DeriveDrift(row)
	return row
}

// DeriveFreshness computes the freshness label for a merged chest row.
func DeriveFreshness(r StatusRow) string {
	if r.LastReviewed != "" {
		return "fresh"
	}
	return "unknown"
}

// DeriveDrift reports missing or unscoped truth-layer relationships for a row.
func DeriveDrift(r StatusRow) []string {
	var d []string
	if r.Configured && !r.Governed {
		d = append(d, "missing_governance")
	}
	if (r.Configured || r.Governed) && !r.Indexed {
		d = append(d, "missing_index")
	}
	if !r.Configured && (r.Governed || r.Indexed) {
		d = append(d, "unscoped")
	}
	return d
}

// FilterRowsByScope filters rows to a Strategist slot scope, treating "all" as a match.
func FilterRowsByScope(rows []StatusRow, value string) []StatusRow {
	if value == "" {
		return rows
	}
	out := make([]StatusRow, 0, len(rows))
	for _, r := range rows {
		for _, s := range r.Scope {
			if s == value || s == "all" {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

// HistoricalCount counts rows with historical/lower-trust tiers.
func HistoricalCount(rows []StatusRow) int {
	n := 0
	for _, r := range rows {
		if r.TrustTier == "T2" || r.TrustTier == "T3" {
			n++
		}
	}
	return n
}
