package main

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X main.Version=x.y.z".
// Release builds carry a bare semver (goreleaser); local `make build` carries
// `git describe --tags --dirty` output. Consumers other than `version` use the
// raw value; only displayVersion normalizes it.
var Version = "dev"

var (
	releaseVersionRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	// git describe shape: <tag>[-<n>-g<hash>][-dirty]; at least one suffix.
	aheadVersionRe = regexp.MustCompile(`^(\d+\.\d+\.\d+)(-\d+-g[0-9a-f]+)?(-dirty)?$`)
)

// displayVersion renders the raw build version as V1.0.18 for a release,
// V1.0.18+ for a local build ahead of (or dirty relative to) its base tag, and
// Vdev when no usable version was injected.
func displayVersion(raw string) string {
	v := strings.TrimLeft(strings.TrimSpace(raw), "vV")
	if releaseVersionRe.MatchString(v) {
		return "V" + v
	}
	if m := aheadVersionRe.FindStringSubmatch(v); m != nil {
		return "V" + m[1] + "+"
	}
	return "Vdev"
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the strategist version",
	Run: func(cmd *cobra.Command, _ []string) {
		slog.DebugContext(cmd.Context(), "[Strategist] version",
			telemetry.AttrComponent, "version",
			telemetry.AttrRuntimeMode, "cli",
			telemetry.AttrOutputProfile, "default",
			"strategist.version", Version,
		)
		fmt.Println(displayVersion(Version))
	},
}
