package cliutil

import (
	"log/slog"

	"github.com/spf13/cobra"
)

// DisplayLevelAnnotation marks a command whose default log display is quieter
// than INFO. Its value is the quiet level ("warn"). The command's own
// --verbose flag lifts it.
const DisplayLevelAnnotation = "strategist.display_level"

// quietLevelWarn is the only quiet level a command can declare.
const quietLevelWarn = "warn"

// QuietByDefault annotates cmd so the root hooks show only WARN and above on
// the default log handler unless its --verbose flag is set. It changes what the
// default handler prints, never which events are emitted: a configured
// OpenTelemetry handler (slog.SetDefault) is not governed by this level and
// keeps receiving every event.
func QuietByDefault(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[DisplayLevelAnnotation] = quietLevelWarn
}

// ApplyDisplayLevel lowers the default log handler to WARN when cmd is
// annotated and not verbose, and returns the function that restores the
// previous level. The restore is a no-op for every other command, so their
// display is unchanged. It is safe to call the restore more than once.
func ApplyDisplayLevel(cmd *cobra.Command) (restore func()) {
	if cmd.Annotations[DisplayLevelAnnotation] != quietLevelWarn || isVerbose(cmd) {
		return func() {}
	}
	previous := slog.SetLogLoggerLevel(slog.LevelWarn)
	restored := false
	return func() {
		if !restored {
			slog.SetLogLoggerLevel(previous)
			restored = true
		}
	}
}

func isVerbose(cmd *cobra.Command) bool {
	flag := cmd.Flags().Lookup("verbose")
	return flag != nil && flag.Value.String() == "true"
}
