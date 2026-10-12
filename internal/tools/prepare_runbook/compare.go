package preparerunbook

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
)

func compare(in Input, candidate runbook.Runbook) (Comparison, error) {
	if reason := runbookRootReason(in.RunbookRoot); reason != "" {
		return insufficientComparison(reason), nil
	}
	paths, err := inventoryPaths(in.RunbookRoot, maxInventoryWalk)
	if err != nil {
		return insufficientComparison("Runbook inventory could not be read"), nil
	}
	paths = prioritizePaths(paths, in.SelectedRunbooksHint)
	paths, notCompared := capInventory(paths, in.MaxInventory)
	comparison := compareInventory(paths, candidate)
	comparison.SelectedHints = append([]string(nil), in.SelectedRunbooksHint...)
	return markTruncation(comparison, notCompared), nil
}

func runbookRootReason(root string) string {
	if strings.TrimSpace(root) == "" {
		return "Runbook root is not configured"
	}
	info, err := os.Stat(root)
	if err != nil {
		return "Runbook root is not readable"
	}
	if !info.IsDir() {
		return "Runbook root is not a directory"
	}
	return ""
}

func insufficientComparison(reason string) Comparison {
	return Comparison{Classification: ComparisonInsufficient, Reason: reason}
}

func inventoryPaths(root string, maxInventory int) ([]string, error) {
	limit := maxInventory
	if limit <= 0 {
		limit = maxInventoryWalk
	}
	paths := make([]string, 0, limit)
	err := filepath.WalkDir(root, collectInventoryPath(&paths, limit))
	if err != nil {
		return nil, fmt.Errorf("walk Runbook inventory: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func collectInventoryPath(paths *[]string, limit int) fs.WalkDirFunc {
	return func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".runbook.yaml") {
			return nil
		}
		*paths = append(*paths, path)
		if len(*paths) >= limit {
			return fs.SkipDir
		}
		return nil
	}
}

func comparePath(path string, candidate runbook.Runbook) (ComparisonRef, string, string) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is bounded by configured root
	if err != nil {
		return ComparisonRef{}, "", "Runbook inventory entry could not be read"
	}
	existing, err := runbook.ParseSidecar(raw)
	if err != nil {
		return ComparisonRef{}, "", "Runbook inventory contains an invalid sidecar"
	}
	relation := relationBetween(candidate, existing)
	return ComparisonRef{RunbookID: existing.RunbookID, Path: path, Relation: relation}, relation, ""
}

func prioritizePaths(paths, hints []string) []string {
	if len(hints) == 0 {
		return paths
	}
	ordered := make([]string, 0, len(paths))
	used := make(map[string]bool, len(paths))
	for _, hint := range hints {
		ordered = appendHintMatches(ordered, paths, hint, used)
	}
	ordered = appendUnmatched(ordered, paths, used)
	return ordered
}

func appendHintMatches(ordered, paths []string, hint string, used map[string]bool) []string {
	for _, path := range paths {
		if used[path] {
			continue
		}
		if strings.Contains(path, hint) {
			ordered = append(ordered, path)
			used[path] = true
		}
	}
	return ordered
}

func appendUnmatched(ordered, paths []string, used map[string]bool) []string {
	for _, path := range paths {
		if !used[path] {
			ordered = append(ordered, path)
		}
	}
	return ordered
}

func relationBetween(a, b runbook.Runbook) string {
	if normalize(a.Objective) == normalize(b.Objective) && normalizeList(a.AppliesWhen) == normalizeList(b.AppliesWhen) {
		return "duplicate"
	}
	if runbook.SharesCanonicalSignal(runbook.CanonicalSignalsIn(strings.Join(a.AppliesWhen, " ")+" "+a.Objective), runbook.CanonicalSignalsIn(strings.Join(b.AppliesWhen, " ")+" "+b.Objective)) {
		return "extension"
	}
	return "none"
}
