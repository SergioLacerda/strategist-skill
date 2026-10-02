package dojo

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ArtifactName identifies a current dojo state artifact.
type ArtifactName string

const (
	// ArtifactCriteria is the user-authored scenario input.
	ArtifactCriteria ArtifactName = "criteria"
	// ArtifactEmitLog is replaceable diagnostic execution evidence.
	ArtifactEmitLog ArtifactName = "emit_log"
	// ArtifactResult is retained latest-run evidence.
	ArtifactResult ArtifactName = "result"
	// ArtifactLesson is durable failure learning.
	ArtifactLesson ArtifactName = "lesson"
	// ArtifactHistory is durable aggregate run history.
	ArtifactHistory ArtifactName = "history"
)

// Durability records the lifecycle promised by the dojo ownership contract.
type Durability string

const (
	// DurabilityDurable identifies state expected to survive workspace lifecycle events.
	DurabilityDurable Durability = "durable"
	// DurabilityRetained identifies derived state retained for inspection.
	DurabilityRetained Durability = "retained"
	// DurabilityReplaceable identifies evidence that may be recreated on a later run.
	DurabilityReplaceable Durability = "replaceable"
)

// ArtifactPolicy describes ownership, backup, and recovery expectations without
// changing the physical public paths used by existing workspaces.
type ArtifactPolicy struct {
	Name              ArtifactName
	Durability        Durability
	UserValue         string
	Owner             string
	BackupExpectation string
	Recovery          string
}

var artifactPolicies = []ArtifactPolicy{
	{
		Name:              ArtifactCriteria,
		Durability:        DurabilityDurable,
		UserValue:         "user-authored scenario input",
		Owner:             "<base_path>/dojo",
		BackupExpectation: "included with the workspace backup",
		Recovery:          "restore from the workspace backup; do not recreate silently",
	},
	{
		Name:              ArtifactEmitLog,
		Durability:        DurabilityReplaceable,
		UserValue:         "diagnostic execution evidence",
		Owner:             "<base_path>/dojo",
		BackupExpectation: "optional diagnostic evidence in the workspace backup",
		Recovery:          "recreate on a later run; a missing log is reported by the checker",
	},
	{
		Name:              ArtifactResult,
		Durability:        DurabilityRetained,
		UserValue:         "latest derived run evidence",
		Owner:             "<base_path>/dojo",
		BackupExpectation: "retained with the workspace when inspection history matters",
		Recovery:          "replace atomically on the next successful persistence attempt",
	},
	{
		Name:              ArtifactLesson,
		Durability:        DurabilityDurable,
		UserValue:         "user-facing failure learning",
		Owner:             "<base_path>/dojo",
		BackupExpectation: "included with the workspace backup",
		Recovery:          "preserve until a later failed run replaces it or an operator restores it",
	},
	{
		Name:              ArtifactHistory,
		Durability:        DurabilityDurable,
		UserValue:         "aggregate trend and learning history",
		Owner:             "<base_path>/dojo",
		BackupExpectation: "included with the workspace backup",
		Recovery:          "preserve valid lines and report malformed lines before cleanup",
	},
}

// ArtifactPolicies returns the ownership contract as a defensive copy.
func ArtifactPolicies() []ArtifactPolicy {
	return append([]ArtifactPolicy(nil), artifactPolicies...)
}

// StoragePaths is the stable path identity for one configured dojo scenario.
// ScenarioID is deliberately rooted at the configured dojo domain rather than
// at the current working directory or generated .strategist runtime.
type StoragePaths struct {
	BasePath    string
	DojoRoot    string
	Scenario    string
	ScenarioID  string
	ScenarioDir string
	LastRunDir  string
	EmitLogPath string
	ResultPath  string
	LessonPath  string
	HistoryPath string
}

// NewStoragePaths resolves all public dojo paths and rejects path-like scenario
// names so state cannot escape the configured dojo domain.
func NewStoragePaths(basePath, scenario string) (StoragePaths, error) {
	if strings.TrimSpace(basePath) == "" {
		return StoragePaths{}, fmt.Errorf("dojo: base path is empty")
	}
	if !validScenarioName(scenario) {
		return StoragePaths{}, fmt.Errorf("dojo: invalid scenario name %q", scenario)
	}

	basePath = filepath.Clean(basePath)
	dojoRoot := filepath.Join(basePath, "dojo")
	lastRunDir := filepath.Join(dojoRoot, ".last-run", scenario)
	return StoragePaths{
		BasePath:    basePath,
		DojoRoot:    dojoRoot,
		Scenario:    scenario,
		ScenarioID:  filepath.ToSlash(filepath.Join("dojo", scenario)),
		ScenarioDir: filepath.Join(dojoRoot, scenario),
		LastRunDir:  lastRunDir,
		EmitLogPath: filepath.Join(lastRunDir, "emit.log"),
		ResultPath:  filepath.Join(lastRunDir, "result.json"),
		LessonPath:  filepath.Join(lastRunDir, "lesson.md"),
		HistoryPath: filepath.Join(dojoRoot, ".history.jsonl"),
	}, nil
}

func validScenarioName(scenario string) bool {
	if scenario == "" || scenario == "." || scenario == ".." || filepath.IsAbs(scenario) {
		return false
	}
	if strings.ContainsAny(scenario, "/\\\x00") {
		return false
	}
	return filepath.Clean(scenario) == scenario
}
