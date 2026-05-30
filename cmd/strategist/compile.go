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
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := telemetry.Tracer().Start(ctx, "strategist.compile",
		trace.WithAttributes(attribute.String(telemetry.AttrTarget, compileRoot)),
	)
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	slog.InfoContext(ctx, "[Strategist] compile running", "root", compileRoot)

	indexPath := filepath.Join(compileRoot, "knowledge.index.yaml")
	c := compile.Compiler{}
	if err := c.CompileAll(compileRoot, indexPath); err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	slog.InfoContext(ctx, "[Strategist] compile complete", "root", compileRoot)
	fmt.Printf("[Strategist] compile complete → %s/.compiled/\n", compileRoot)
	return nil
}

func init() {
	compileCmd.Flags().StringVar(&compileRoot, "root", "", "path to .strategist/ root (default: .strategist)")
}
