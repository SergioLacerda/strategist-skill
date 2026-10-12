package treasurecli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/cli/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/runbook"
)

const (
	// legacyRunbooksDir is the compatibility root; it is always read.
	legacyRunbooksDir = "docs/runbooks"
	// diagRootsUnavailable reports an active.yaml that cannot be read, so only
	// the legacy root is used.
	diagRootsUnavailable = "runbooks_root_unavailable"
)

// rankedSidecar is one sidecar path with the rank of the root it came from;
// rank 0 is the configured root and wins a duplicate runbook_id.
type rankedSidecar struct {
	path string
	rank int
}

// runbookRoots returns the project-relative directories to read: the root
// configured by `runbooks.canonical_path` first, then the legacy root as a
// fallback. A missing active.yaml means only the legacy root; an unreadable one
// does the same and says so.
func runbookRoots(strategistDir string) ([]string, []corpusDiagnostic) {
	roots := []string{legacyRunbooksDir}
	cfg, err := cliutil.LoadActiveConfig(strategistDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return roots, nil
		}
		return roots, []corpusDiagnostic{{
			Path: "active.yaml", Code: diagRootsUnavailable, Severity: runbook.SeverityWarn,
			Detail: firstLine(err),
		}}
	}
	canonical := filepath.ToSlash(filepath.Clean(cfg.Runbooks.CanonicalPath))
	if cfg.Runbooks.CanonicalPath == "" || canonical == legacyRunbooksDir || !filepath.IsLocal(canonical) {
		return roots, nil
	}
	return []string{canonical, legacyRunbooksDir}, nil
}

// globRunbookSidecars lists the sidecars of every root, configured root first;
// the paths of one root are sorted.
func globRunbookSidecars(projectRoot string, roots []string) ([]rankedSidecar, error) {
	var out []rankedSidecar
	for rank, root := range roots {
		matches, err := filepath.Glob(filepath.Join(projectRoot, filepath.FromSlash(root), "*.runbook.yaml"))
		if err != nil {
			return nil, fmt.Errorf("runbook select: glob %s: %w", root, err)
		}
		sort.Strings(matches)
		for _, match := range matches {
			out = append(out, rankedSidecar{path: match, rank: rank})
		}
	}
	return out, nil
}
