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
	var out strings.Builder
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		findings, err := refinement.CheckPackagePortability(filepath.Join(refinedRoot, entry.Name()))
		if err != nil {
			continue // package without analysis.md: not a portability concern
		}
		for _, f := range findings {
			fmt.Fprintf(&out, "  ⚠ package_portability: %s: %s\n", f.Path, f.Reason)
		}
	}
	if out.Len() > 0 {
		if _, err := io.WriteString(w, out.String()); err != nil {
			return // advisory output only; a failed write must not affect check
		}
	}
}
