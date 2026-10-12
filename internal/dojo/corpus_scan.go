package dojo

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

type scanned struct {
	entry  CorpusEntry
	fields calibration.Fields
	dir    string
}

func scanRoots(basePath string, cal domain.DojoCalibration, reader FieldReader) ([]scanned, error) {
	var out []scanned
	for _, root := range corpusRoots(cal) {
		found, err := scanRoot(basePath, root, cal, reader)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].entry.Path < out[j].entry.Path })
	return out, nil
}

func corpusRoots(cal domain.DojoCalibration) []string {
	if len(cal.Corpus.Roots) > 0 {
		return cal.Corpus.Roots
	}
	if cal.Boundary == calibration.BoundaryRanger {
		return []string{"pending"}
	}
	return []string{"refined", "done"}
}

func scanRoot(basePath, root string, cal domain.DojoCalibration, reader FieldReader) ([]scanned, error) {
	dir := filepath.Join(basePath, filepath.Clean(root))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("dojo: read corpus root %s: %w", dir, err)
	}
	var out []scanned
	for _, e := range entries {
		if item, ok := scanEntry(basePath, root, dir, e, cal, reader); ok {
			out = append(out, item)
		}
	}
	return out, nil
}

func scanEntry(basePath, root, dir string, e os.DirEntry, cal domain.DojoCalibration, reader FieldReader) (scanned, bool) {
	name := e.Name()
	if strings.HasPrefix(name, ".") || (len(cal.Corpus.Include) > 0 && !slices.Contains(cal.Corpus.Include, strings.TrimSuffix(name, "-analysis.md"))) {
		return scanned{}, false
	}
	path := filepath.Join(dir, name)
	var fields calibration.Fields
	var err error
	if cal.Boundary == calibration.BoundaryRanger {
		fields, err = readRanger(path, e, reader)
	} else {
		fields, err = readPackage(path, e, root, reader)
	}
	if err != nil || len(fields) == 0 {
		return scanned{}, false
	}
	rel, _ := filepath.Rel(basePath, path) //nolint:errcheck // both paths are under basePath
	return scanned{entry: CorpusEntry{Name: strings.TrimSuffix(name, "-analysis.md"), Path: filepath.ToSlash(rel), Status: filepath.Base(root), Digest: calibration.Digest(fields), Amendments: countAmendments(path)}, fields: fields, dir: path}, true
}

func readRanger(path string, e os.DirEntry, reader FieldReader) (calibration.Fields, error) {
	if e.IsDir() || !strings.HasSuffix(path, "-analysis.md") {
		return nil, os.ErrNotExist
	}
	fields, err := reader.Ranger(path)
	if err != nil {
		return nil, fmt.Errorf("dojo: read ranger artifact %s: %w", path, err)
	}
	return fields, nil
}

var canonicalFiles = []string{"analysis.md", "proposal.md", "design.md", "tasks.md"}

// readPackage reads a package that has all four canonical files and, outside the
// done root, a gate-accepted analysis: a refined package not yet accepted has no label.
func readPackage(path string, e os.DirEntry, root string, reader FieldReader) (calibration.Fields, error) {
	if !e.IsDir() || !hasCanonicalFiles(path) {
		return nil, os.ErrNotExist
	}
	if filepath.Base(root) == "refined" && missionStatus(filepath.Join(path, "analysis.md")) != "gate_analysis_accepted" {
		return nil, os.ErrNotExist
	}
	fields, err := reader.Package(filepath.Join(path, "tasks.md"), filepath.Join(path, "analysis.md"))
	if err != nil {
		return nil, fmt.Errorf("dojo: read package %s: %w", path, err)
	}
	return fields, nil
}

func hasCanonicalFiles(dir string) bool {
	for _, name := range canonicalFiles {
		if info, err := os.Stat(filepath.Join(dir, name)); err != nil || info.IsDir() {
			return false
		}
	}
	return true
}

func missionStatus(analysisPath string) string {
	raw, err := os.ReadFile(analysisPath) //nolint:gosec // G304: a package analysis under the corpus root
	if err != nil {
		return ""
	}
	rest, ok := strings.CutPrefix(string(raw), "---\n")
	if !ok {
		return ""
	}
	header, _, _ := strings.Cut(rest, "\n---")
	var front struct {
		Status string `yaml:"mission_status"`
	}
	if yaml.Unmarshal([]byte(header), &front) != nil {
		return ""
	}
	return front.Status
}

func countAmendments(pkg string) int {
	entries, err := os.ReadDir(filepath.Join(pkg, ".amendments"))
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			n++
		}
	}
	return n
}
