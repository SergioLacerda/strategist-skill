package dojo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// LoadCriteria reads, parses, and validates a criteria.yaml for the given scenario directory.
func LoadCriteria(scenarioDir string) (domain.DojoCriteria, error) {
	raw, err := os.ReadFile(filepath.Join(scenarioDir, "criteria.yaml"))
	if err != nil {
		return domain.DojoCriteria{}, fmt.Errorf("dojo: read criteria.yaml: %w", err)
	}
	var c domain.DojoCriteria
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return domain.DojoCriteria{}, fmt.Errorf("dojo: parse criteria.yaml: %w", err)
	}
	if err := ValidateCriteria(c); err != nil {
		return domain.DojoCriteria{}, err
	}
	return c, nil
}

// ValidateCriteria rejects structurally invalid or unsafe scenario criteria before any
// check runs, so a malformed criteria.yaml fails loudly instead of silently checking nothing.
func ValidateCriteria(c domain.DojoCriteria) error {
	var errs []string
	if strings.TrimSpace(c.Scenario) == "" {
		errs = append(errs, "scenario is required")
	}
	if strings.TrimSpace(c.RunDir) == "" {
		errs = append(errs, "run_dir is required")
	}
	errs = append(errs, validateFileCheckPaths(c.FilesCreated)...)
	errs = append(errs, validateManifestChecks(c.ManifestChecks)...)
	errs = append(errs, validateTimingCriteria(c.TimingCriteria)...)
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("dojo: criteria invalid: %s", strings.Join(errs, "; "))
}

func validateFileCheckPaths(files []domain.DojoFileCheck) []string {
	var errs []string
	for _, fc := range files {
		if strings.TrimSpace(fc.Path) == "" {
			errs = append(errs, "files_created: path is required")
			continue
		}
		if isUnsafeRelPath(fc.Path) {
			errs = append(errs, fmt.Sprintf("files_created: path %q escapes its root", fc.Path))
		}
	}
	return errs
}

// isUnsafeRelPath reports whether rel is absolute or would resolve outside a root
// it is joined against (e.g. "../../outside.md").
func isUnsafeRelPath(rel string) bool {
	if filepath.IsAbs(rel) {
		return true
	}
	clean := filepath.Clean(rel)
	return clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

func validateManifestChecks(checks []domain.DojoManifestCheck) []string {
	var errs []string
	for _, mc := range checks {
		errs = append(errs, validateManifestCheck(mc)...)
	}
	return errs
}

func validateManifestCheck(mc domain.DojoManifestCheck) []string {
	var errs []string
	if strings.TrimSpace(mc.ExpectedProvider) == "" {
		errs = append(errs, "manifest_checks: expected_provider is required")
	}
	if hasEmptyManifestField(mc.FieldsPresent) {
		errs = append(errs, "manifest_checks: fields_present entries must be non-empty")
	}
	return errs
}

func hasEmptyManifestField(fields []string) bool {
	for _, field := range fields {
		if strings.TrimSpace(field) == "" {
			return true
		}
	}
	return false
}

func validateTimingCriteria(tc *domain.DojoTimingCriteria) []string {
	if tc == nil {
		return nil
	}
	if tc.MaxWallTimeMs <= 0 {
		return []string{"timing_criteria: max_wall_time_ms must be positive"}
	}
	return nil
}
