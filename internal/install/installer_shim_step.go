package install

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

type installTransaction struct {
	strategistDir string
	existedBefore bool
	createdPaths  []string
}

func newInstallTransaction(strategistDir string) *installTransaction {
	return &installTransaction{
		strategistDir: strategistDir,
		existedBefore: runtimefs.Exists(strategistDir),
	}
}

func (tx *installTransaction) record(paths ...string) {
	tx.createdPaths = append(tx.createdPaths, paths...)
}

// rollback restores the workspace to its pre-install state on failure.
// On a fresh install, Strategist owns the entire tree and removes it wholesale.
// On a re-install over an existing tree, pre-existing content is never deleted:
// only entries tracked during this transaction are rolled back.
func (tx *installTransaction) rollback(ctx context.Context) {
	if !tx.existedBefore {
		if err := os.RemoveAll(tx.strategistDir); err != nil {
			slog.ErrorContext(ctx, "[Strategist] rollback failed — manual cleanup required",
				"path", tx.strategistDir,
				"error", err,
				"hint", fmt.Sprintf("rm -rf %s", tx.strategistDir),
				telemetry.AttrComponent, "install",
				telemetry.AttrRuntimeMode, "cli",
				telemetry.AttrOutputProfile, "default",
				telemetry.AttrTarget, tx.strategistDir,
			)
		}
	}
	// strategistDir (when freshly removed above) already covers everything under
	// it; the remaining manifest entries are outside it (e.g. target/.gitignore)
	// or, on the existing-tree path, the specific entries added this run.
	rollbackManifest(ctx, tx.createdPaths)
}

// runShimStep writes the SKILL.md shim per cfg (default path, --shim-path, or
// skipped entirely for --no-shim), and returns the paths to track for rollback
// (dir before file, so reverse-order rollback removes the file first).
func (s Service) runShimStep(ctx context.Context, cfg domain.InstallConfig) ([]string, error) {
	if cfg.NoShim {
		slog.InfoContext(ctx, "[Strategist] install shim-step skipped (--no-shim)",
			telemetry.AttrComponent, "install",
		)
		return nil, nil
	}
	slog.InfoContext(ctx, "[Strategist] install shim-step",
		telemetry.AttrComponent, "install",
		telemetry.AttrTarget, telemetry.SanitizePath(cfg.Target),
	)
	shimPath, err := s.installShimStep(ctx, cfg.Target, cfg.ShimPath)
	if err != nil {
		return nil, err
	}
	return []string{filepath.Dir(shimPath), shimPath}, nil
}

// installShimStep reads target/.strategist/SKILL.md and writes the full shim.
// shimPathOverride, if non-empty, writes the shim there instead of the default
// ~/.claude/skills/strategist/SKILL.md location (see --shim-path).
// Returns the shim path for rollback tracking.
func (s Service) installShimStep(ctx context.Context, target, shimPathOverride string) (string, error) {
	skillContent, err := s.readLocalSKILLMD(ctx, target)
	if err != nil {
		return "", fmt.Errorf("install: read SKILL.md: %w", err)
	}
	shimPath, err := s.resolveShimPath(shimPathOverride)
	if err != nil {
		return "", fmt.Errorf("install: resolve shim path: %w", err)
	}
	if err := s.installShimFor(target, skillContent, shimPathOverride); err != nil {
		return "", fmt.Errorf("install: shim: %w", err)
	}
	return shimPath, nil
}

// readLocalSKILLMD reads SKILL.md from the embedded FS.
// .strategist/ is write-only — we never read back from it.
func (s Service) readLocalSKILLMD(ctx context.Context, _ string) (string, error) {
	data, err := s.Extractor.ReadFile(skillMDName)
	if err != nil {
		return "", fmt.Errorf("read embedded SKILL.md: %w", err)
	}
	slog.InfoContext(ctx, "[Strategist] SKILL.md read from embedded FS",
		telemetry.AttrComponent, "install",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
	)
	return string(data), nil
}

// installShimFor installs the shim, using ShimHomeDir if set (for tests) or
// shimPathOverride if set (--shim-path).
func (s Service) installShimFor(target, skillContent, shimPathOverride string) error {
	skillRoot := ""
	if target != "" {
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("resolve target: %w", err)
		}
		skillRoot = filepath.Join(absTarget, strategistDirName)
	}
	if shimPathOverride != "" {
		return installShimToPath(shimPathOverride, skillContent, skillRoot)
	}
	if s.ShimHomeDir != "" {
		return installShimTo(s.ShimHomeDir, skillContent, skillRoot)
	}
	return installShim(target)
}

// resolveShimPath returns the path of the SKILL.md shim that will be installed,
// without actually installing it. Used to track the shim in the rollback manifest.
func (s Service) resolveShimPath(shimPathOverride string) (string, error) {
	if shimPathOverride != "" {
		return shimPathOverride, nil
	}
	homeDir := s.ShimHomeDir
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home dir: %w", err)
		}
	}
	return defaultShimPath(homeDir), nil
}

// rollbackManifest removes created paths in reverse order (best-effort).
// Non-empty directories are silently skipped — Remove only removes empty dirs.
func rollbackManifest(ctx context.Context, manifest []string) {
	for i := len(manifest) - 1; i >= 0; i-- {
		if err := os.Remove(manifest[i]); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.WarnContext(ctx, "[Strategist] rollback remove failed",
				"path", manifest[i],
				"error", err,
				telemetry.AttrComponent, "install",
				telemetry.AttrRuntimeMode, "cli",
				telemetry.AttrOutputProfile, "default",
				telemetry.AttrTarget, manifest[i],
			)
		}
	}
	slog.WarnContext(ctx, "[Strategist] install rolled back",
		"workspace", "restored",
		telemetry.AttrComponent, "install",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
	)
}
