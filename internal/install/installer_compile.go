package install

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func (s Service) compileAfterInstall(ctx context.Context, cfg domain.InstallConfig, strategistDir string) error {
	kiPath := filepath.Join(strategistDir, "knowledge.index.yaml")
	if compileErr := s.Compiler.CompileAll(strategistDir, kiPath); compileErr != nil {
		if cfg.StrictCompile {
			return fmt.Errorf("install: strict compile: %w", compileErr)
		}
		slog.WarnContext(ctx, "[Strategist] compile warning",
			"error", compileErr,
			telemetry.AttrComponent, "install",
			telemetry.AttrRuntimeMode, "cli",
			telemetry.AttrOutputProfile, "default",
			telemetry.AttrTarget, strategistDir,
		)
	}
	return nil
}
