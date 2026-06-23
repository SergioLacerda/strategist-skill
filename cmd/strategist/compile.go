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
		compileRoot = ".strategist"
		if err := requireStrategistDir(); err != nil {
			return err
		}
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	run := telemetry.MissionRunFromContext(ctx)
	if run != nil {
		run.MarkRanger()
	}
	ctx, span := telemetry.Tracer().Start(ctx, "strategist.compile",
		trace.WithAttributes(
			attribute.String(telemetry.AttrComponent, "compile"),
			attribute.String(telemetry.AttrRuntimeMode, "cli"),
			attribute.String(telemetry.AttrOutputProfile, "default"),
			attribute.String(telemetry.AttrTarget, compileRoot),
		),
	)
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	if run != nil {
		run.AddLines(1)
	}
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

	if run != nil {
		run.AddLines(2)
	}
	slog.InfoContext(ctx, "[Strategist] compile complete",
		telemetry.AttrComponent, "compile",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
		telemetry.AttrTarget, compileRoot,
	)
	fmt.Printf("[Strategist] compile complete → %s/.compiled/\n", compileRoot)
	return nil
}

func init() {
	compileCmd.Flags().StringVar(&compileRoot, "root", "", "path to .strategist/ root (default: .strategist)")
}
