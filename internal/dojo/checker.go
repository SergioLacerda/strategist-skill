// Package dojo implements the offline criteria checker for strategist dojo scenarios.
package dojo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// errPathEscape marks a scenario-relative path that resolves outside its expected root.
var errPathEscape = errors.New("dojo: path escapes root")

// Run executes all checks for a scenario and returns the aggregated result.
// Log-dependent checks (emit log, timing, pipeline) are skipped when filesOnly is true.
func Run(criteria domain.DojoCriteria, basePath, strategistDir, emitLogPath string, filesOnly bool) domain.DojoCheckResult {
	result := domain.DojoCheckResult{Scenario: criteria.Scenario}
	result.Items = append(result.Items, CheckFiles(criteria, basePath)...)
	result.Items = append(result.Items, CheckEmitLog(criteria, emitLogPath, filesOnly)...)
	result.Items = append(result.Items, CheckManifests(criteria, strategistDir)...)
	result.Items = append(result.Items, CheckTiming(criteria, emitLogPath, filesOnly)...)
	result.Items = append(result.Items, CheckPipeline(criteria, emitLogPath, filesOnly)...)
	return result
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ifFail(passed bool, detail string) string {
	if passed {
		return ""
	}
	return detail
}

// newItem builds a DojoCheckItem, clearing detail on pass so callers can pass
// the failure detail unconditionally.
func newItem(label string, passed bool, detail string) domain.DojoCheckItem {
	return domain.DojoCheckItem{Label: label, Passed: passed, Detail: ifFail(passed, detail)}
}

// checkTextAssertion builds one check item for a single needle against text,
// passing when needle's presence in text matches wantPresent.
func checkTextAssertion(label, text, needle string, wantPresent bool, failDetail string) domain.DojoCheckItem {
	found := strings.Contains(text, needle)
	return newItem(label, found == wantPresent, failDetail)
}

// readFileUnderRoot reads rel resolved against root, rejecting any path that
// escapes root (e.g. via "../" traversal in scenario-controlled criteria).
// It returns the file content and the resolved absolute path for messages.
func readFileUnderRoot(root, rel string) ([]byte, string, error) {
	full := filepath.Join(root, rel)
	relCheck, err := filepath.Rel(filepath.Clean(root), full)
	if err != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(filepath.Separator)) {
		return nil, full, fmt.Errorf("%w: %q escapes root %q", errPathEscape, rel, root)
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, full, fmt.Errorf("dojo: read %s: %w", full, err)
	}
	return data, full, nil
}

// ScenarioHasCriteria reports whether name is a dojo scenario directory,
// i.e. dojoDir/name/criteria.yaml exists.
func ScenarioHasCriteria(dojoDir, name string) bool {
	return fileExists(filepath.Join(dojoDir, name, "criteria.yaml"))
}
