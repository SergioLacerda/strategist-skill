package main

import (
	"context"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func main() {
	cfg := telemetry.FromEnv()
	shutdown, err := telemetry.Init(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Strategist] WARN: telemetry init failed: %v\n", err)
	} else {
		defer shutdown(context.Background()) //nolint:errcheck // best-effort shutdown
	}
	execute()
}
