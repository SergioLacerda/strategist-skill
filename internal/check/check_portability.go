package check

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/refinement"
)

// reportRefinedPortability runs the advisory CheckPackagePortability over each
// package under <worktree>/<base_path>/refined and writes one warning per
// finding to w. It never contributes to the check's error count, so old
// packages are surfaced for review without changing the exit code.
func reportRefinedPortability(w io.Writer, strategistRoot, basePath string) {
	if basePath == "" {
		return
	}
	refinedRoot := filepath.Join(filepath.Dir(strategistRoot), basePath, "refined")
	entries, err := os.ReadDir(refinedRoot)
	if err != nil {
		return
	}
	report := formatRefinedPortabilityFindings(refinedRoot, entries)
	if report == "" {
		return
	}
	if _, err := io.WriteString(w, report); err != nil {
		return // advisory output only; a failed write must not affect check
	}
}

// formatRefinedPortabilityFindings renders one warning line per portability
// finding across every package directory in entries, skipping non-directory
// entries. Split out of reportRefinedPortability to keep cognitive complexity
// within budget.
func formatRefinedPortabilityFindings(refinedRoot string, entries []os.DirEntry) string {
	var out strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			writePackagePortabilityFindings(&out, filepath.Join(refinedRoot, entry.Name()))
		}
	}
	return out.String()
}

// writePackagePortabilityFindings appends one warning line per finding for a
// single refined package directory. A package without analysis.md is not a
// portability concern and is silently skipped.
func writePackagePortabilityFindings(out *strings.Builder, pkgDir string) {
	findings, err := refinement.CheckPackagePortability(pkgDir)
	if err != nil {
		return
	}
	for _, f := range findings {
		fmt.Fprintf(out, "  ⚠ package_portability: %s: %s\n", f.Path, f.Reason)
	}
}
