package main

import (
	"context"
	"log/slog"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func main() {
	cfg := telemetry.FromEnv()
	shutdown, err := telemetry.Init(cfg)
	if err != nil {
		slog.Warn("[Strategist] telemetry init failed",
			"error", err,
			telemetry.AttrComponent, "main",
			telemetry.AttrRuntimeMode, "cli",
			telemetry.AttrOutputProfile, "default",
		)
	} else {
		defer shutdown(context.Background()) //nolint:errcheck // best-effort shutdown
	}
	execute()
}

func init() {
	runtimepayload.RegisterOpenSpec(embed.DefaultsFS())
}
