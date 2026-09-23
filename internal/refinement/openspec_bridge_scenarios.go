package refinement

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// deltaHeader starts the part of an OpenSpec spec that states requirements and
// scenarios; the Purpose section before it is prose the design already covers.
var deltaHeader = regexp.MustCompile(`(?m)^## (ADDED|MODIFIED|REMOVED|RENAMED) Requirements`)

// withAcceptanceScenarios appends the change's requirements and scenarios to
// design.md. The four-file package shape is kept, but the testable acceptance
// criteria the provider wrote in specs/ are no longer lost on normalization.
func withAcceptanceScenarios(design []byte, changeDir string) ([]byte, error) {
	blocks, err := specDeltaBlocks(filepath.Join(changeDir, "specs"))
	if err != nil || len(blocks) == 0 {
		return design, err
	}
	var out bytes.Buffer
	out.Write(bytes.TrimRight(design, "\n"))
	out.WriteString("\n\n## Acceptance scenarios\n\nCarried from the OpenSpec change's specs by normalization.\n")
	for _, block := range blocks {
		out.WriteString(block)
	}
	return out.Bytes(), nil
}

// specDeltaBlocks returns one block per spec file with requirement deltas, in
// capability path order so the output is deterministic.
func specDeltaBlocks(specsDir string) ([]string, error) {
	paths, err := specFiles(specsDir)
	if err != nil {
		return nil, err
	}
	var blocks []string
	for _, path := range paths {
		raw, err := os.ReadFile(path) //nolint:gosec // path is under the validated change directory
		if err != nil {
			return nil, fmt.Errorf("openspec bridge: read %s: %w", path, err)
		}
		loc := deltaHeader.FindIndex(raw)
		if loc == nil {
			continue
		}
		capability := filepath.ToSlash(filepath.Dir(mustRel(specsDir, path)))
		blocks = append(blocks, fmt.Sprintf("\n### `%s`\n\n%s\n", capability, demoteHeadings(strings.TrimSpace(string(raw[loc[0]:])))))
	}
	return blocks, nil
}

func specFiles(specsDir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(specsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && d.Name() == "spec.md" {
			paths = append(paths, path)
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("openspec bridge: list specs: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func mustRel(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}

// demoteHeadings shifts markdown headings two levels down so a spec's `##`
// sections nest under the capability heading inside design.md.
func demoteHeadings(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			lines[i] = "##" + line
		}
	}
	return strings.Join(lines, "\n")
}

// archiveChange moves a normalized change out of the active change list,
// following OpenSpec's `changes/archive/<date>-<id>` layout. It is a plain move:
// provider spec deltas are never merged into the runtime's specs.
func archiveChange(runtimeRoot, changeDir, changeID string) error {
	archiveDir := filepath.Join(runtimeRoot, "changes", "archive")
	if err := os.MkdirAll(archiveDir, 0o750); err != nil {
		return fmt.Errorf("openspec bridge: create archive: %w", err)
	}
	target := filepath.Join(archiveDir, time.Now().UTC().Format("2006-01-02")+"-"+changeID)
	if err := os.Rename(changeDir, target); err != nil {
		return fmt.Errorf("openspec bridge: archive change %s: %w", changeID, err)
	}
	return nil
}
