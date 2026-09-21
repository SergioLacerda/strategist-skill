package main

import (
	"bytes"
	"log/slog"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// --- version --build ---

func TestFormatBuildInfoShowsCommitPlatformAndEmbeddedOpenSpec(t *testing.T) {
	settings := map[string]string{"vcs.revision": "895d45b1c2e3f4a5b6c7d8e9f0a1b2c3d4e5f607", "vcs.modified": "false"}
	payload := &buildPayload{OpenSpec: "1.13.0"}

	got := formatBuildInfo(settings, "windows", "amd64", payload)

	assert.Equal(t, []string{
		"commit: 895d45b1c2e3",
		"platform: windows/amd64",
		"runtime: embedded OpenSpec 1.13.0; host Node >=20.19.0 required",
	}, got)
}

func TestFormatBuildInfoFlagsDirtyTreesAndMissingInformation(t *testing.T) {
	dirty := formatBuildInfo(map[string]string{"vcs.revision": "895d45b1c2e3f4a5", "vcs.modified": "true"}, "linux", "arm64", nil)
	assert.Equal(t, "commit: 895d45b1c2e3 (modified)", dirty[0])
	assert.Equal(t, "runtime: embedded OpenSpec bundle; host Node >=20.19.0 required", dirty[2])

	unknown := formatBuildInfo(nil, "linux", "amd64", nil)
	assert.Equal(t, "commit: unknown", unknown[0])
}

func TestVersionCmd_BuildFlagAddsLinesButDefaultStaysOneLine(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "1.0.18"

	t.Cleanup(func() { _ = versionCmd.Flags().Set("build", "false") })
	require.NoError(t, versionCmd.Flags().Set("build", "true"))
	out := captureStdout(t, func() { versionCmd.Run(versionCmd, nil) })
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Equal(t, "V1.0.18", lines[0])
	assert.GreaterOrEqual(t, len(lines), 4)
	assert.Contains(t, out, "platform: "+runtime.GOOS+"/"+runtime.GOARCH)

	require.NoError(t, versionCmd.Flags().Set("build", "false"))
	assert.Equal(t, "V1.0.18\n", captureStdout(t, func() { versionCmd.Run(versionCmd, nil) }))
}

func TestEmbeddedPayloadSummarizesTheEmbeddedOpenSpec(t *testing.T) {
	got := embeddedPayload()
	require.NotNil(t, got)
	assert.Equal(t, "1.13.0", got.OpenSpec)
}
