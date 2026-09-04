package treasurecli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
)

// resolveRunbookActionRoot resolves the workspace project root (the parent
// of .strategist/) — docs/runbooks lives at the project root, not inside
// .strategist/, so this mirrors scanRegisteredChestsForPotions' own
// root/projectRoot split rather than resolveTreasureChestActionRoot's
// .strategist-scoped helper (runbook is not a treasure-chest subcommand).
func resolveRunbookActionRoot(cmd *cobra.Command, explicitRoot string) (strategistDir, projectRoot string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("runbook select: get cwd: %w", err)
	}
	strategistDir, projectRoot, err = resolveStrategistRoot(stringFlag(cmd, flagRoot, explicitRoot), cwd)
	if err != nil {
		return "", "", fmt.Errorf("runbook select: %w", err)
	}
	return strategistDir, projectRoot, nil
}

// loadRunbookSidecars parses every docs/runbooks/*.runbook.yaml sidecar
// under projectRoot. A parse failure is a hard error naming the offending
// file — no silent skip (see docs/runbooks/README.md's own authoring
// guidance: an honestly under-specified sidecar is better than a silently
// wrong one; a silently *ignored* one is worse than either).
//
// Every candidate's Trust is populated from treasure-chests.yaml's
// "runbooks" chest entry (governed[runbookChestID].Trust.Tier) — trust is
// chest-level metadata, not part of the sidecar schema, so ParseSidecar
// never sets it (see runbook.Runbook.Trust's doc comment). A missing or
// unreadable treasure-chests.yaml leaves Trust at its zero value, which
// runbook.Select's MinTrust check treats as "no trust signal, do not
// filter" — the same non-blocking posture LoadGoverned itself takes on a
// missing file.
func loadRunbookSidecars(strategistDir, projectRoot string) ([]runbook.Runbook, map[string]string, error) {
	matches, err := filepath.Glob(filepath.Join(projectRoot, runbookSidecarGlob))
	if err != nil {
		return nil, nil, fmt.Errorf("runbook select: glob %s: %w", runbookSidecarGlob, err)
	}
	sort.Strings(matches)

	governed, err := treasure.LoadGoverned(strategistDir)
	if err != nil {
		return nil, nil, fmt.Errorf("runbook select: load treasure-chests.yaml: %w", err)
	}
	runbookTrust := governed[runbookChestID].Trust.Tier

	candidates := make([]runbook.Runbook, 0, len(matches))
	sourceDocByID := make(map[string]string, len(matches))
	for _, path := range matches {
		data, readErr := os.ReadFile(path) //nolint:gosec // path comes from a controlled glob under the resolved project root
		if readErr != nil {
			return nil, nil, fmt.Errorf("runbook select: read %s: %w", path, readErr)
		}
		rb, parseErr := runbook.ParseSidecar(data)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("runbook select: parse %s: %w", path, parseErr)
		}
		rb.Trust = runbookTrust
		candidates = append(candidates, rb)
		sourceDocByID[rb.RunbookID] = rb.SourceDoc
	}
	return candidates, sourceDocByID, nil
}
