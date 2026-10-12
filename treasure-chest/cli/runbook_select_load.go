package treasurecli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
	"github.com/SergioLacerda/strategist-skill/treasure-chest"
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
// under projectRoot and returns the ones that may be offered together with the
// diagnostics of the ones that were skipped or only warn. One inconsistent
// file never fails the corpus: a skip is reported (stderr, one line each) and
// never silent — a silently ignored sidecar is worse than a failing one (see
// docs/runbooks/README.md). An error is returned only when the corpus cannot
// be enumerated. Unreadable chest trust metadata degrades to "no trust signal"
// with a trust_unavailable warning (it is optional metadata, not a runbook).
//
// Every candidate's Trust is populated from treasure-chests.yaml's
// "runbooks" chest entry (governed[runbookChestID].Trust.Tier) — trust is
// chest-level metadata, not part of the sidecar schema, so ParseSidecar
// never sets it (see runbook.Runbook.Trust's doc comment). A missing
// or unreadable treasure-chests.yaml leaves Trust at its zero value, which
// select_runbook.Select's MinTrust check treats as "no trust signal, do not
// filter".
func loadRunbookSidecars(strategistDir, projectRoot string) (runbookCorpus, error) {
	roots, rootDiags := runbookRoots(strategistDir)
	matches, err := globRunbookSidecars(projectRoot, roots)
	if err != nil {
		return runbookCorpus{}, err
	}

	runbookTrust, trustDiags := loadRunbookTrust(strategistDir)

	parsed, diags := parseSidecars(matches, projectRoot)
	diags = append(diags, rootDiags...)
	diags = append(diags, trustDiags...)
	kept, dupDiags := resolveDuplicateIDs(parsed)
	diags = append(diags, dupDiags...)
	sortDiagnostics(diags)

	corpus := runbookCorpus{
		Candidates:    make([]runbook.Runbook, 0, len(kept)),
		SourceDocByID: make(map[string]string, len(kept)),
		Diagnostics:   diags,
	}
	for _, p := range kept {
		p.rb.Trust = runbookTrust
		corpus.Candidates = append(corpus.Candidates, p.rb)
		corpus.SourceDocByID[p.rb.RunbookID] = p.rb.SourceDoc
	}
	return corpus, nil
}

// parseSidecars reads, parses and consistency-checks every match. Files that
// cannot be read, do not parse or carry a skip finding become diagnostics;
// warn findings are recorded and the sidecar is kept.
func parseSidecars(matches []rankedSidecar, projectRoot string) ([]parsedSidecar, []corpusDiagnostic) {
	parsed := make([]parsedSidecar, 0, len(matches))
	var diags []corpusDiagnostic
	for _, match := range matches {
		rel := relSidecarPath(projectRoot, match.path)
		data, readErr := os.ReadFile(match.path) //nolint:gosec // path comes from a controlled glob under the resolved project root
		if readErr != nil {
			diags = append(diags, corpusDiagnostic{Path: rel, Code: diagReadFailed, Severity: runbook.SeveritySkip, Detail: firstLine(readErr)})
			continue
		}
		rb, parseErr := runbook.ParseSidecar(data)
		if parseErr != nil {
			diags = append(diags, corpusDiagnostic{Path: rel, Code: diagSidecarInvalid, Severity: runbook.SeveritySkip, Detail: firstLine(parseErr)})
			continue
		}
		findings := runbook.CheckSidecarConsistency(rel, rb, projectRoot)
		diags = append(diags, findingDiagnostics(rel, findings)...)
		if hasSkipFinding(findings) {
			continue
		}
		parsed = append(parsed, parsedSidecar{relPath: rel, stem: strings.TrimSuffix(filepath.Base(rel), ".runbook.yaml"), rank: match.rank, rb: rb})
	}
	return parsed, diags
}

func findingDiagnostics(rel string, findings []runbook.Finding) []corpusDiagnostic {
	out := make([]corpusDiagnostic, 0, len(findings))
	for _, f := range findings {
		out = append(out, corpusDiagnostic{Path: rel, Code: f.Code, Severity: f.Severity, Detail: f.Detail})
	}
	return out
}

func hasSkipFinding(findings []runbook.Finding) bool {
	for _, f := range findings {
		if f.Severity == runbook.SeveritySkip {
			return true
		}
	}
	return false
}

// loadRunbookTrust reads the trust tier of the "runbooks" chest. The tier is
// optional metadata: a missing file already yields none, and an unreadable one
// degrades to none with a visible trust_unavailable warning instead of
// stopping the selection.
func loadRunbookTrust(strategistDir string) (string, []corpusDiagnostic) {
	governed, err := treasure.LoadGoverned(strategistDir)
	if err != nil {
		return "", []corpusDiagnostic{{
			Path: "treasure-chests.yaml", Code: diagTrustUnavailable,
			Severity: runbook.SeverityWarn, Detail: firstLine(err),
		}}
	}
	return governed[runbookChestID].Trust.Tier, nil
}
