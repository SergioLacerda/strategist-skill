// Package dojo implements the offline criteria checker for strategist dojo scenarios.
package dojo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// LoadCriteria reads and parses a criteria.yaml for the given scenario directory.
func LoadCriteria(scenarioDir string) (domain.DojoCriteria, error) {
	raw, err := os.ReadFile(filepath.Join(scenarioDir, "criteria.yaml"))
	if err != nil {
		return domain.DojoCriteria{}, fmt.Errorf("dojo: read criteria.yaml: %w", err)
	}
	var c domain.DojoCriteria
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return domain.DojoCriteria{}, fmt.Errorf("dojo: parse criteria.yaml: %w", err)
	}
	return c, nil
}

// CheckFiles validates the files_created section of criteria against the run directory.
func CheckFiles(criteria domain.DojoCriteria, basePath string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	runDir := filepath.Join(basePath, criteria.RunDir)

	for _, fc := range criteria.FilesCreated {
		full := filepath.Join(runDir, fc.Path)
		exists := fileExists(full)
		items = append(items, domain.DojoCheckItem{
			Label:  fmt.Sprintf("files_created %s", fc.Path),
			Passed: exists,
			Detail: ifFail(exists, "file not found: "+full),
		})
		if !exists {
			continue
		}

		content, err := os.ReadFile(full)
		if err != nil {
			items = append(items, domain.DojoCheckItem{
				Label:  fmt.Sprintf("files_created %s (read)", fc.Path),
				Passed: false,
				Detail: err.Error(),
			})
			continue
		}
		text := string(content)

		for _, section := range fc.RequiredSections {
			found := strings.Contains(text, section)
			items = append(items, domain.DojoCheckItem{
				Label:  fmt.Sprintf("section %q in %s", section, fc.Path),
				Passed: found,
				Detail: ifFail(found, fmt.Sprintf("section %q not found", section)),
			})
		}
		for _, needle := range fc.MustContain {
			found := strings.Contains(text, needle)
			items = append(items, domain.DojoCheckItem{
				Label:  fmt.Sprintf("must_contain %q in %s", needle, fc.Path),
				Passed: found,
				Detail: ifFail(found, fmt.Sprintf("%q not found in file", needle)),
			})
		}
		for _, needle := range fc.MustNotContain {
			found := strings.Contains(text, needle)
			items = append(items, domain.DojoCheckItem{
				Label:  fmt.Sprintf("must_not_contain %q in %s", needle, fc.Path),
				Passed: !found,
				Detail: ifFail(!found, fmt.Sprintf("%q must not appear in file", needle)),
			})
		}
	}
	return items
}

// CheckEmitLog validates the emit_log section of criteria against a captured log file.
// logPath is the path to the emit.log written during an LLM run.
// If logPath does not exist and filesOnly is true, emit checks are skipped.
func CheckEmitLog(criteria domain.DojoCriteria, logPath string, filesOnly bool) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	if len(criteria.EmitLog.MustContain) == 0 && len(criteria.EmitLog.MustNotContain) == 0 {
		return items
	}

	if !fileExists(logPath) {
		if filesOnly {
			return items
		}
		for _, key := range criteria.EmitLog.MustContain {
			items = append(items, domain.DojoCheckItem{
				Label:  fmt.Sprintf("emit %s", key),
				Passed: false,
				Detail: "emit.log not found — run the LLM scenario first",
			})
		}
		return items
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		return []domain.DojoCheckItem{{
			Label:  "emit_log read",
			Passed: false,
			Detail: err.Error(),
		}}
	}
	log := string(raw)

	for _, key := range criteria.EmitLog.MustContain {
		found := strings.Contains(log, key)
		items = append(items, domain.DojoCheckItem{
			Label:  fmt.Sprintf("emit %s", key),
			Passed: found,
			Detail: ifFail(found, fmt.Sprintf("emit key %q not found in log", key)),
		})
	}
	for _, key := range criteria.EmitLog.MustNotContain {
		found := strings.Contains(log, key)
		items = append(items, domain.DojoCheckItem{
			Label:  fmt.Sprintf("emit %s must NOT appear", key),
			Passed: !found,
			Detail: ifFail(!found, fmt.Sprintf("emit key %q must not appear in log", key)),
		})
	}
	return items
}

// CheckManifests validates the manifest_checks section of criteria.
// strategistDir is the path to the .strategist/ directory.
func CheckManifests(criteria domain.DojoCriteria, strategistDir string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem

	for _, mc := range criteria.ManifestChecks {
		manifestPath := filepath.Join(strategistDir, "skills", mc.ExpectedProvider, "skill.yaml")
		exists := fileExists(manifestPath)
		items = append(items, domain.DojoCheckItem{
			Label:  fmt.Sprintf("manifest %s/skills/%s/skill.yaml", mc.Slot, mc.ExpectedProvider),
			Passed: exists == mc.ManifestExists,
			Detail: ifFail(exists == mc.ManifestExists, fmt.Sprintf("manifest_exists=%v but got %v", mc.ManifestExists, exists)),
		})
		if !exists {
			continue
		}

		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			items = append(items, domain.DojoCheckItem{
				Label:  fmt.Sprintf("manifest %s read", mc.ExpectedProvider),
				Passed: false,
				Detail: err.Error(),
			})
			continue
		}
		text := string(raw)
		for _, field := range mc.FieldsPresent {
			found := strings.Contains(text, field+":")
			items = append(items, domain.DojoCheckItem{
				Label:  fmt.Sprintf("manifest field %q in skills/%s/skill.yaml", field, mc.ExpectedProvider),
				Passed: found,
				Detail: ifFail(found, fmt.Sprintf("field %q not found in manifest", field)),
			})
		}
	}
	return items
}

// CheckTiming validates wall-time performance from a timing_criteria block.
// It reads total_wall_time_ms=<value> from the emit log.
// If timing_criteria is nil, returns empty (no check performed).
func CheckTiming(criteria domain.DojoCriteria, logPath string) []domain.DojoCheckItem {
	if criteria.TimingCriteria == nil {
		return nil
	}
	tc := criteria.TimingCriteria

	if !fileExists(logPath) {
		return []domain.DojoCheckItem{{
			Label:  "timing total_wall_time_ms",
			Passed: false,
			Detail: "emit.log not found — run the LLM scenario first",
		}}
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		return []domain.DojoCheckItem{{
			Label:  "timing total_wall_time_ms",
			Passed: false,
			Detail: err.Error(),
		}}
	}

	log := string(raw)
	const field = "total_wall_time_ms="
	idx := strings.Index(log, field)
	if idx < 0 {
		return []domain.DojoCheckItem{{
			Label:  "timing total_wall_time_ms",
			Passed: false,
			Detail: "total_wall_time_ms not found in emit.log",
		}}
	}

	rest := log[idx+len(field):]
	end := strings.IndexAny(rest, " \t\n\r")
	if end < 0 {
		end = len(rest)
	}
	valStr := rest[:end]
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return []domain.DojoCheckItem{{
			Label:  "timing total_wall_time_ms",
			Passed: false,
			Detail: fmt.Sprintf("cannot parse total_wall_time_ms=%q: %v", valStr, err),
		}}
	}

	passed := val <= tc.MaxWallTimeMs
	return []domain.DojoCheckItem{{
		Label:  "timing total_wall_time_ms",
		Passed: passed,
		Detail: ifFail(passed, fmt.Sprintf("wall time %d ms exceeds max %d ms", val, tc.MaxWallTimeMs)),
	}}
}

// Run executes all checks for a scenario and returns the aggregated result.
func Run(criteria domain.DojoCriteria, basePath, strategistDir, emitLogPath string, filesOnly bool) domain.DojoCheckResult {
	result := domain.DojoCheckResult{Scenario: criteria.Scenario}
	result.Items = append(result.Items, CheckFiles(criteria, basePath)...)
	result.Items = append(result.Items, CheckEmitLog(criteria, emitLogPath, filesOnly)...)
	result.Items = append(result.Items, CheckManifests(criteria, strategistDir)...)
	result.Items = append(result.Items, CheckTiming(criteria, emitLogPath)...)
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
