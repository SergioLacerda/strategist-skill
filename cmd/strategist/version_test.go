package main

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- version ---

func TestVersionCmd_PrintsVersion(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "1.2.3-test"

	out := captureStdout(t, func() {
		versionCmd.Run(versionCmd, nil)
	})
	assert.Contains(t, out, "1.2.3-test")
	assert.Contains(t, out, "strategist")
}

func TestVersionCmd_EmitsStructuredTelemetry(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "1.2.3-test"

	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })

	versionCmd.Run(versionCmd, nil)

	out := buf.String()
	assert.Contains(t, out, "strategist.component=version")
	assert.Contains(t, out, "strategist.runtime_mode=cli")
	assert.Contains(t, out, "strategist.output_profile=default")
	assert.Contains(t, out, "strategist.version=1.2.3-test")
}

// --- compile ---
