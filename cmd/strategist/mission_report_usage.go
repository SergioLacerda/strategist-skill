package main

import (
	"fmt"
	"path/filepath"
	"strings"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/spf13/cobra"
)

// validateMissionReportUsageOptions checks the flags this command treats as
// required and range-validates the token counts. Cobra's Int64Var flag type
// already rejects a non-numeric --tokens-in/--tokens-out at parse time
// (returns "invalid argument ... strconv.ParseInt" before RunE is ever
// called), so this only needs to catch: the flag never passed at all, and a
// value that parsed fine but is out of range (negative).
func validateMissionReportUsageOptions(cmd *cobra.Command, opts missionadapter.ReportUsageOptions) error {
	var problems []string
	if err := validateMissionID(opts.MissionID); err != nil {
		problems = append(problems, err.Error())
	}
	problems = append(problems, validateTokenFlag(cmd, "tokens-in", opts.TokensIn)...)
	problems = append(problems, validateTokenFlag(cmd, "tokens-out", opts.TokensOut)...)
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return nil
}

// validateTokenFlag reports a "required" problem when --name was never set,
// or a "must be >= 0" problem when an explicitly-set value is negative. cmd
// may be nil in unit tests that call validateMissionReportUsageOptions
// directly against a bare opts struct — treated as "flag not set".
func validateTokenFlag(cmd *cobra.Command, name string, value int64) []string {
	changed := cmd != nil && cmd.Flags().Changed(name)
	switch {
	case !changed:
		return []string{fmt.Sprintf("--%s is required", name)}
	case value < 0:
		return []string{fmt.Sprintf("--%s must be >= 0, got %d", name, value)}
	default:
		return nil
	}
}

// missionIDKnown reports whether id matches an existing mission artifact
// under any of the three canonical analysis directories, mirroring
// internal/install.GenerateMissionID's own collision check (that function
// cannot be reused directly: it is unexported and lives in a package this
// task is not scoped to touch).
func missionIDKnown(basePath, id string) bool {
	for _, sub := range []string{"pending", "refined", "archived"} {
		matches, err := filepath.Glob(filepath.Join(basePath, sub, id+"*"))
		if err == nil && len(matches) > 0 {
			return true
		}
	}
	return false
}
