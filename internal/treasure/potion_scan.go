package treasure

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// --- index scan extension (ask #1 / SQ-001) ---

var runbookHeadingRe = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)

// ScanRunbookDirectory scans a "runbooks"-kind chest directory (docs/runbooks/*.md) and
// builds status:proposed Potion candidates, one per runbook file. when_to_use is
// extracted from the first paragraph of the file's first "## " section (Symptom for
// diagnostic runbooks, Trigger for procedural ones) — header-extracted, not
// LLM-generated, per .analysis/refined/runbook-jewel-relevance-mechanism/design.md.
// A missing directory is not an error — it just yields no candidates.
func ScanRunbookDirectory(chestID, trustTier, dirPath string) ([]Potion, error) {
	entries, err := readRunbookDirEntries(dirPath)
	if err != nil {
		return nil, err
	}

	var candidates []Potion
	for _, entry := range entries {
		if !isRunbookCandidateFile(entry) {
			continue
		}
		candidate, err := runbookFileToPotion(chestID, trustTier, dirPath, entry.Name())
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	return candidates, nil
}

// readRunbookDirEntries reads dirPath's entries. A missing directory is not an
// error — it returns a nil slice so callers yield no candidates.
func readRunbookDirEntries(dirPath string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan runbook directory %s: %w", dirPath, err)
	}
	return entries, nil
}

func isRunbookCandidateFile(entry os.DirEntry) bool {
	if entry.IsDir() {
		return false
	}
	name := entry.Name()
	return strings.HasSuffix(name, ".md") && !strings.EqualFold(name, "README.md")
}

func runbookFileToPotion(chestID, trustTier, dirPath, fileName string) (Potion, error) {
	path := filepath.Join(dirPath, fileName)
	raw, err := os.ReadFile(path) //nolint:gosec // G304
	if err != nil {
		return Potion{}, fmt.Errorf("read %s: %w", path, err)
	}
	slug := strings.TrimSuffix(fileName, ".md")
	runbookRef := "docs/runbooks/" + fileName
	return Potion{
		ID:         "potion-" + slug,
		ChestID:    chestID,
		RunbookRef: runbookRef,
		WhenToUse:  extractRunbookWhenToUse(string(raw)),
		Trust:      trustTier,
		Status:     domain.PotionStatusProposed,
		SourceRefs: []string{runbookRef},
		ReviewedBy: "agent",
	}, nil
}

// extractRunbookWhenToUse extracts a short summary from a runbook's first "## " section
// (e.g. "Symptom" or "Trigger") — the first non-empty paragraph after that heading,
// truncated. Falls back to the H1 title when no "## " section is found.
func extractRunbookWhenToUse(content string) string {
	if loc := runbookHeadingRe.FindStringIndex(content); loc != nil {
		if para := firstParagraphAfter(content[loc[1]:]); para != "" {
			return truncateRunbookSummary(para)
		}
	}
	return truncateRunbookSummary(runbookTitleFallback(content))
}

// isParagraphBoundary reports whether trimmed (a line already stripped of
// surrounding whitespace) ends paragraph collection: either a blank line
// (which only stops collection once some content has been gathered, via
// hasContent) or a heading line, which always stops it outright.
func isParagraphBoundary(trimmed string, hasContent bool) (boundary, stopEntirely bool) {
	if trimmed == "" {
		return true, hasContent
	}
	if strings.HasPrefix(trimmed, "#") {
		return true, true
	}
	return false, false
}

func firstParagraphAfter(rest string) string {
	lines := strings.Split(rest, "\n")
	var para []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if boundary, stop := isParagraphBoundary(trimmed, len(para) > 0); boundary {
			if stop {
				break
			}
			continue
		}
		para = append(para, trimmed)
	}
	return strings.Join(para, " ")
}

func runbookTitleFallback(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimPrefix(trimmed, "# ")
		}
	}
	return ""
}

func truncateRunbookSummary(s string) string {
	const maxLen = 220
	if len(s) <= maxLen {
		return s
	}
	return strings.TrimSpace(s[:maxLen]) + "…"
}
