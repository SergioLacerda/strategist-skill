package treasurecli

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
)

// Diagnostic codes owned by the loader (the consistency codes live in
// internal/runbook next to the checks that produce them).
const (
	diagReadFailed       = "read_failed"
	diagSidecarInvalid   = "sidecar_invalid"
	diagDuplicateID      = "duplicate_runbook_id"
	diagTrustUnavailable = "trust_unavailable"
	runbookDiagnosticTag = "[Strategist] runbook select:"
)

// corpusDiagnostic is one skipped or suspicious sidecar. Nothing is dropped
// silently: every skip is reported with its path and a stable code.
type corpusDiagnostic struct {
	Path     string
	Code     string
	Severity string
	Detail   string
}

// runbookCorpus is the loaded corpus: the sidecars that may be offered plus
// the diagnostics of the ones that were skipped or that only warn.
type runbookCorpus struct {
	Candidates    []runbook.Runbook
	SourceDocByID map[string]string
	Diagnostics   []corpusDiagnostic
}

// counts returns how many sidecars were skipped and how many only warn.
func (c runbookCorpus) counts() (skipped, warnings int) {
	for _, d := range c.Diagnostics {
		if d.Severity == runbook.SeveritySkip {
			skipped++
		} else {
			warnings++
		}
	}
	return skipped, warnings
}

// parsedSidecar is a sidecar that was read and parsed, before duplicate
// resolution.
type parsedSidecar struct {
	relPath string
	stem    string
	rank    int // 0 is the configured root, which wins a duplicate runbook_id
	rb      runbook.Runbook
}

func relSidecarPath(projectRoot, path string) string {
	if rel, err := filepath.Rel(projectRoot, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

// firstLine keeps a diagnostic detail on one line and free of file contents:
// only the first line of the underlying error is reported.
func firstLine(err error) string {
	line, _, _ := strings.Cut(err.Error(), "\n")
	return line
}

// resolveDuplicateIDs keeps, for every duplicated runbook_id, one sidecar: the
// one from the highest-priority root (the configured root over the legacy one)
// and, among equals, the one whose file stem equals the id; when none or more
// than one qualifies it keeps none. Every other file is reported as a skip
// naming the others.
func resolveDuplicateIDs(parsed []parsedSidecar) ([]parsedSidecar, []corpusDiagnostic) {
	byID := make(map[string][]parsedSidecar, len(parsed))
	for _, p := range parsed {
		byID[p.rb.RunbookID] = append(byID[p.rb.RunbookID], p)
	}
	kept := make([]parsedSidecar, 0, len(parsed))
	var diags []corpusDiagnostic
	for _, p := range parsed {
		group := byID[p.rb.RunbookID]
		if len(group) == 1 || isDuplicateWinner(p, group) {
			kept = append(kept, p)
			continue
		}
		diags = append(diags, corpusDiagnostic{
			Path: p.relPath, Code: diagDuplicateID, Severity: runbook.SeveritySkip,
			Detail: fmt.Sprintf("runbook_id %q is also used by %s", p.rb.RunbookID, otherPaths(p, group)),
		})
	}
	return kept, diags
}

// isDuplicateWinner applies the root priority first and the stem rule second.
func isDuplicateWinner(p parsedSidecar, group []parsedSidecar) bool {
	best := bestRank(group)
	if p.rank != best {
		return false
	}
	peers := sameRank(group, best)
	return len(peers) == 1 || isStemWinner(p, peers)
}

func bestRank(group []parsedSidecar) int {
	best := group[0].rank
	for _, g := range group {
		if g.rank < best {
			best = g.rank
		}
	}
	return best
}

func sameRank(group []parsedSidecar, rank int) []parsedSidecar {
	var peers []parsedSidecar
	for _, g := range group {
		if g.rank == rank {
			peers = append(peers, g)
		}
	}
	return peers
}

func isStemWinner(p parsedSidecar, group []parsedSidecar) bool {
	if p.stem != p.rb.RunbookID {
		return false
	}
	winners := 0
	for _, g := range group {
		if g.stem == g.rb.RunbookID {
			winners++
		}
	}
	return winners == 1
}

func otherPaths(self parsedSidecar, group []parsedSidecar) string {
	var others []string
	for _, g := range group {
		if g.relPath != self.relPath {
			others = append(others, g.relPath)
		}
	}
	return strings.Join(others, ", ")
}

func sortDiagnostics(diags []corpusDiagnostic) {
	sort.SliceStable(diags, func(i, k int) bool {
		if diags[i].Path != diags[k].Path {
			return diags[i].Path < diags[k].Path
		}
		return diags[i].Code < diags[k].Code
	})
}

// printCorpusDiagnostics writes one stderr line per diagnostic and, only when
// there is at least one, a summary line. stdout is never touched, so a fully
// valid corpus keeps both streams exactly as before.
func printCorpusDiagnostics(w io.Writer, corpus runbookCorpus) {
	if len(corpus.Diagnostics) == 0 {
		return
	}
	for _, d := range corpus.Diagnostics {
		verb := "warning"
		if d.Severity == runbook.SeveritySkip {
			verb = "skipped"
		}
		fmt.Fprintf(w, "%s %s %s code=%s detail=%s\n", runbookDiagnosticTag, verb, d.Path, d.Code, d.Detail) //nolint:errcheck // diagnostics on stderr are best-effort
	}
	skipped, warnings := corpus.counts()
	fmt.Fprintf(w, "%s corpus loaded=%d skipped=%d warnings=%d\n", runbookDiagnosticTag, len(corpus.Candidates), skipped, warnings) //nolint:errcheck // diagnostics on stderr are best-effort
}
