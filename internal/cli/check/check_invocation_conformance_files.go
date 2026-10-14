package check

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// conformanceTopLevelFiles and conformanceDirs are the agent-facing runtime
// texts an agent copies command lines from.
var (
	conformanceTopLevelFiles = []string{"SKILL.md", "agent-protocol.md", "protocol.md"}
	conformanceDirs          = []string{"contracts", "roles"}
)

func fileConformanceAdvisories(root string, tree *cobra.Command, rel string) []string {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))) //nolint:gosec // G304: fixed runtime texts under the selected root
	if err != nil {
		return nil
	}
	var advisories []string
	for _, line := range commandLines(rel, string(raw)) {
		for _, problem := range conformanceProblems(tree, line.command) {
			advisories = append(advisories, formatConformanceAdvisory(rel, line, problem))
		}
	}
	return advisories
}

func formatConformanceAdvisory(rel string, line commandLine, problem string) string {
	return fmt.Sprintf(
		"[Strategist] phase=preflight status=warn reason=invocation_nonconformant file=%s line=%d command=`%s` detail=%s (the installed binary would reject this documented command line)",
		rel, line.number, line.command, problem)
}

func appendUnique(values []string, seen map[string]bool, candidates ...string) []string {
	for _, candidate := range candidates {
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		values = append(values, candidate)
	}
	return values
}

// conformanceFiles lists the top-level texts first, then the walked
// directories in sorted order, so advisories are reported deterministically.
func conformanceFiles(root string) []string {
	files := existingConformanceFiles(root)
	var walked []string
	for _, dir := range conformanceDirs {
		walked = append(walked, walkedConformanceFiles(root, dir)...)
	}
	sort.Strings(walked)
	return append(files, walked...)
}

func existingConformanceFiles(root string) []string {
	var files []string
	for _, name := range conformanceTopLevelFiles {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			files = append(files, name)
		}
	}
	return files
}

func walkedConformanceFiles(root, dir string) []string {
	var files []string
	_ = filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error { //nolint:errcheck // a missing directory simply contributes no files
		if err != nil || d.IsDir() || !isConformanceText(path) {
			return nil //nolint:nilerr // unreadable entries are skipped, not fatal for an advisory
		}
		if rel, relErr := filepath.Rel(root, path); relErr == nil {
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	return files
}

func isConformanceText(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".yaml", ".yml":
		return true
	}
	return false
}
