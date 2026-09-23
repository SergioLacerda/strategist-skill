package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	levelingadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

const roleLevelLedger = "role-levels.jsonl"

const defaultLedgerMaxRecords = 2000

func levelingAdapterDependencies() levelingadapter.Dependencies {
	return levelingadapter.Dependencies{
		LoadPolicy:    loadLevelingPolicy,
		WorkspaceRoot: levelingWorkspaceRoot,
		LoadConfig:    readActiveLevelingConfig,
		LoadRegistry:  loadRoleRegistry,
		LedgerName:    roleLevelLedger,
		RotateMax:     defaultLedgerMaxRecords,
		EmitRoleLevel: emitRoleLevel,
	}
}

func emitRoleLevel(ctx context.Context, missionID, run string, level leveling.Level, reason string) {
	attrs := []any{
		telemetry.AttrComponent, "leveling",
		telemetry.AttrMissionID, missionID,
		telemetry.AttrRole, level.Role,
		telemetry.AttrModel, level.Model,
		telemetry.AttrEffort, level.Effort,
		telemetry.AttrLevelSource, level.Source,
	}
	if reason != "" {
		attrs = append(attrs, telemetry.AttrReason, reason)
	}
	if run != "" {
		attrs = append(attrs, telemetry.AttrRoleRun, run)
	}
	slog.InfoContext(ctx, "role_level_resolved", attrs...)
}

func loadLevelingPolicy() (leveling.Policy, string, error) {
	root, err := levelingWorkspaceRoot()
	if err != nil {
		return leveling.Policy{}, "", err
	}
	path := filepath.Join(root, "leveling.yaml")
	override, _, err := leveling.ReadOverride(path)
	if err != nil {
		return leveling.Policy{}, path, fmt.Errorf("read policy: %w", err)
	}
	defaults, err := readEmbeddedLevelingDefaults()
	if err != nil {
		return leveling.Policy{}, path, fmt.Errorf("leveling_policy_stale: read embedded defaults: %w", err)
	}
	effective, _, err := leveling.LoadAuthorized(root, defaults, override, path)
	if err != nil {
		return leveling.Policy{}, path, fmt.Errorf("leveling: load authorized policy: %w", err)
	}
	return effective.Policy, path, nil
}

func levelingWorkspaceRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("leveling: get cwd: %w", err)
	}
	root, _, err := findStrategistRoot(cwd)
	if err != nil {
		return "", fmt.Errorf("leveling: %w", err)
	}
	return root, nil
}

// readActiveLevelingConfig reads the `leveling:` block through the shared
// active.yaml loader. A missing or unreadable file, or an invalid block,
// degrades to automatic with a warning: labelling never blocks a mission.
func readActiveLevelingConfig(root string) (domain.LevelingConfig, string) {
	active, err := cliutil.LoadActiveConfig(root)
	if errors.Is(err, os.ErrNotExist) {
		return domain.LevelingConfig{}, ""
	}
	if err != nil {
		return domain.LevelingConfig{}, err.Error()
	}
	if err := active.Leveling.Validate(); err != nil {
		return domain.LevelingConfig{}, err.Error()
	}
	return active.Leveling, ""
}

// loadRoleRegistry loads the workspace's roles/*.yaml over the built-in registry.
// A broken role file degrades to the built-ins with a warning: labelling never
// blocks a mission.
func loadRoleRegistry(root string) (domain.RoleRegistry, string) {
	reg, err := domain.LoadRoleRegistry(filepath.Join(root, "roles"))
	if err != nil {
		return domain.DefaultRoleRegistry(), err.Error()
	}
	return reg, ""
}

func readEmbeddedLevelingDefaults() ([]byte, error) {
	raw, err := (embedpkg.Extractor{}).ReadFile("leveling.yaml")
	if err != nil {
		return nil, fmt.Errorf("read embedded leveling.yaml: %w", err)
	}
	return raw, nil
}
