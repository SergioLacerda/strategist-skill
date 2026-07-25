// Package dojo implements the offline criteria checker for strategist dojo scenarios.
package dojo

import (
	"fmt"
	"os"
	"path/filepath"

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
