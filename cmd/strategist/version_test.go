package main

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- version ---

func TestDisplayVersion(t *testing.T) {
	cases := map[string]string{
		"1.0.18":                   "V1.0.18",
		"v1.0.18":                  "V1.0.18",
		"V1.0.18":                  "V1.0.18",
		"v1.0.18-3-gabc1234":       "V1.0.18+",
		"v1.0.18-3-gabc1234-dirty": "V1.0.18+",
		"v1.0.18-dirty":            "V1.0.18+",
		"1.0.18-dirty":             "V1.0.18+",
		"v1.0.18-rc1":              "Vdev",
		"dev":                      "Vdev",
		"":                         "Vdev",
		"1.2.3-test":               "Vdev",
	}
	for raw, want := range cases {
		assert.Equal(t, want, displayVersion(raw), "raw=%q", raw)
	}
}

func TestVersionCmd_PrintsSingleCleanLine(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "1.0.18"

	out := captureStdout(t, func() {
		versionCmd.Run(versionCmd, nil)
	})
	assert.Equal(t, "V1.0.18\n", out)
}

func TestVersionCmd_TelemetryIsDebugLevel(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "1.2.3-test"

	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })

	_ = captureStdout(t, func() { versionCmd.Run(versionCmd, nil) })
	assert.Empty(t, buf.String(), "version must not log at Info level")

	buf.Reset()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	_ = captureStdout(t, func() { versionCmd.Run(versionCmd, nil) })
	out := buf.String()
	assert.Contains(t, out, "strategist.component=version")
	assert.Contains(t, out, "strategist.version=1.2.3-test")
}

func TestVersionIsHumanStatusCommand(t *testing.T) {
	assert.True(t, isHumanStatusCommand(versionCmd))
}

// --- compile ---
