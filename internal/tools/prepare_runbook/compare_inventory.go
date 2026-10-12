package preparerunbook

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
)

// DefaultMaxInventory is the explicit number of existing sidecars compared when
// Input.MaxInventory is not set. The previous cap was an unreported 32; a
// truncated inventory is now reported (Comparison.Truncated, NotCompared) and
// cannot support a "novel" verdict.
const DefaultMaxInventory = 256

// maxInventoryWalk bounds how many sidecar paths are collected before the
// comparison cap and the hint priority are applied.
const maxInventoryWalk = 4096

// capInventory keeps the first limit paths (hint matches come first) and
// returns how many were left out.
func capInventory(paths []string, limit int) ([]string, int) {
	if limit <= 0 {
		limit = DefaultMaxInventory
	}
	if len(paths) <= limit {
		return paths, 0
	}
	return paths[:limit], len(paths) - limit
}

// markTruncation records an incomplete inventory. A duplicate or extension that
// was found stays valid evidence, but "no overlap" cannot be concluded from a
// partial inventory.
func markTruncation(c Comparison, notCompared int) Comparison {
	if notCompared == 0 {
		return c
	}
	c.Truncated = true
	c.NotCompared = notCompared
	if c.Classification == ComparisonNovel {
		c.Classification = ComparisonInsufficient
		c.Reason = fmt.Sprintf("Runbook inventory truncated: %d entries were not compared", notCompared)
	}
	return c
}

// inventoryTally accumulates the per-entry results of one comparison.
type inventoryTally struct {
	refs           []ComparisonRef
	classification ComparisonKind
	skipped        int
	compared       int
	firstReason    string
}

func (tl *inventoryTally) record(path string, candidate runbook.Runbook) {
	ref, relation, reason := comparePath(path, candidate)
	if reason != "" {
		tl.skipped++
		if tl.firstReason == "" {
			tl.firstReason = reason
		}
		return
	}
	tl.compared++
	if relation != "none" {
		tl.refs = append(tl.refs, ref)
	}
	tl.classification = nextComparison(tl.classification, relation)
}

// comparison builds the result. Only when entries exist and none could be
// compared is it insufficient.
func (tl inventoryTally) comparison(entries int) Comparison {
	if entries > 0 && tl.compared == 0 {
		c := insufficientComparison(tl.firstReason)
		c.SkippedEntries = tl.skipped
		return c
	}
	c := Comparison{Classification: tl.classification, Compared: tl.refs, Reason: comparisonReason(tl.classification), SkippedEntries: tl.skipped}
	if tl.skipped > 0 {
		c.Reason = fmt.Sprintf("%s; %d inventory entries could not be compared", c.Reason, tl.skipped)
	}
	return c
}

// compareInventory compares the candidate with every entry. An unreadable or
// invalid sidecar is skipped and counted (Comparison.SkippedEntries) instead of
// turning the whole comparison insufficient.
func compareInventory(paths []string, candidate runbook.Runbook) Comparison {
	tally := inventoryTally{refs: make([]ComparisonRef, 0, len(paths)), classification: ComparisonNovel}
	for _, path := range paths {
		tally.record(path, candidate)
	}
	return tally.comparison(len(paths))
}

func nextComparison(current ComparisonKind, relation string) ComparisonKind {
	switch {
	case relation == "duplicate":
		return ComparisonDuplicate
	case relation == "extension" && current == ComparisonNovel:
		return ComparisonExtension
	default:
		return current
	}
}
