package main

import (
	"context"
	"log/slog"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func main() {
	cfg := telemetry.FromEnv()
	shutdown, err := telemetry.Init(cfg)
	if err != nil {
		slog.Warn("[Strategist] telemetry init failed", "error", err)
	} else {
		defer shutdown(context.Background()) //nolint:errcheck // best-effort shutdown
	}
	execute()
}
