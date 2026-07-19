package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

var compileRoot string

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile all skill artifacts from a .strategist/ root",
	RunE:  runCompile,
}

func runCompile(cmd *cobra.Command, _ []string) (retErr error) {
	if compileRoot == "" {
		if err := defaultCompileRoot(); err != nil {
			return err
		}
	}

	ctx := commandContext(cmd)
	markCompileRun(ctx)
	ctx, span := startCompileSpan(ctx)
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	addMissionLines(ctx, 1)
	slog.InfoContext(ctx, "[Strategist] compile running",
		telemetry.AttrComponent, "compile",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
		telemetry.AttrTarget, compileRoot,
	)

	indexPath := filepath.Join(compileRoot, "knowledge.index.yaml")
	c := compile.Compiler{}
	if err := c.CompileAll(compileRoot, indexPath); err != nil {
		return fmt.Errorf("compile: compile all: %w", err)
	}

	addMissionLines(ctx, 2)
	slog.InfoContext(ctx, "[Strategist] compile complete",
		telemetry.AttrComponent, "compile",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
		telemetry.AttrTarget, compileRoot,
	)
	fmt.Printf("[Strategist] compile complete → %s/.compiled/\n", compileRoot)

	refreshCompileAwareness(ctx)

	return nil
}

func defaultCompileRoot() error {
	compileRoot = ".strategist"
	return requireStrategistDir()
}

func markCompileRun(ctx context.Context) {
	if run := telemetry.MissionRunFromContext(ctx); run != nil {
		run.MarkRanger()
	}
}

func startCompileSpan(ctx context.Context) (context.Context, trace.Span) {
	return telemetry.Tracer().Start(ctx, "strategist.compile",
		trace.WithAttributes(
			attribute.String(telemetry.AttrComponent, "compile"),
			attribute.String(telemetry.AttrRuntimeMode, "cli"),
			attribute.String(telemetry.AttrOutputProfile, "default"),
			attribute.String(telemetry.AttrTarget, compileRoot),
		),
	)
}

func refreshCompileAwareness(ctx context.Context) {
	projectRoot := filepath.Dir(compileRoot)
	tplBytes, err := embedpkg.Extractor{}.ReadFile("templates/agent-protocol.md")
	if err != nil {
		slog.WarnContext(ctx, "[Strategist] agent-protocol template missing", "error", err)
		tplBytes = nil
	}
	if protocolOK := compile.RefreshAgentAwareness(compileRoot, projectRoot, Version, tplBytes); protocolOK {
		fmt.Printf("[Strategist] agent-protocol → %s/agent-protocol.md\n", compileRoot)
	}
	fmt.Printf("[Strategist] agent-awareness pass complete → %s\n", projectRoot)
}

func init() {
	compileCmd.Flags().StringVar(&compileRoot, "root", "", "path to .strategist/ root (default: .strategist)")
}
