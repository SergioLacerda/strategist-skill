package main

import (
	"fmt"
	"log/slog"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
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
		showBuild, err := cmd.Flags().GetBool("build")
		if err != nil {
			slog.ErrorContext(cmd.Context(), "[Strategist] version flag lookup failed", "error", err)
			return
		}
		if showBuild {
			for _, line := range formatBuildInfo(vcsSettings(), runtime.GOOS, runtime.GOARCH, embeddedPayload()) {
				fmt.Println(line)
			}
		}
	},
}

func init() {
	versionCmd.Flags().Bool("build", false, "also print the commit, platform and embedded runtime payload of this binary")
}

// buildPayload summarizes the runtime payload compiled into a binary.
type buildPayload struct{ OpenSpec, Node string }

// formatBuildInfo renders the extra `version --build` lines: enough to tell two
// binaries apart (commit, platform) and whether the private runtime is inside.
func formatBuildInfo(settings map[string]string, goos, goarch string, payload *buildPayload) []string {
	commit := "unknown"
	if rev := settings["vcs.revision"]; rev != "" {
		if len(rev) > 12 {
			rev = rev[:12]
		}
		commit = rev
		if settings["vcs.modified"] == "true" {
			commit += " (modified)"
		}
	}
	runtimeLine := "runtime payload: none (openspec resolved from PATH)"
	if payload != nil {
		runtimeLine = fmt.Sprintf("runtime payload: embedded (openspec %s, node %s)", payload.OpenSpec, payload.Node)
	}
	return []string{"commit: " + commit, "platform: " + goos + "/" + goarch, runtimeLine}
}

func vcsSettings() map[string]string {
	settings := map[string]string{}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			settings[s.Key] = s.Value
		}
	}
	return settings
}

func embeddedPayload() *buildPayload {
	embedded, ok := runtimepayload.Default()
	if !ok {
		return nil
	}
	out := &buildPayload{}
	for _, c := range embedded.Manifest.Components {
		switch c.Name {
		case "openspec":
			out.OpenSpec = c.Version
		case "node":
			out.Node = c.Version
		}
	}
	return out
}
