package main

import (
	"fmt"
	"path/filepath"
	"strings"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/spf13/cobra"
)

// validateMissionReportUsageOptions checks the flags this command treats as
// required and range-validates the counts. Cobra's Int64Var already rejects a
// non-numeric value at parse time, so this catches a missing --phase/--role,
// no observed count at all, and a negative count. Counts are optional
// individually: an omitted one is "not observed", never zero.
func validateMissionReportUsageOptions(_ *cobra.Command, opts missionadapter.ReportUsageOptions) error {
	var problems []string
	if err := validateMissionID(opts.MissionID); err != nil {
		problems = append(problems, err.Error())
	}
	for flag, value := range map[string]string{"phase": opts.Phase, "role": opts.Role} {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, fmt.Sprintf("--%s is required", flag))
		}
	}
	if !opts.InSet && !opts.OutSet {
		problems = append(problems, "at least one of --tokens-in or --tokens-out is required")
	}
	problems = append(problems, negativeCountProblems(opts)...)
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return nil
}

func negativeCountProblems(opts missionadapter.ReportUsageOptions) []string {
	var problems []string
	counts := []struct {
		name  string
		set   bool
		value int64
	}{
		{"tokens-in", opts.InSet, opts.TokensIn}, {"tokens-out", opts.OutSet, opts.TokensOut},
		{"tokens-cache", opts.CacheSet, opts.TokensCache}, {"tokens-reasoning", opts.ReasonSet, opts.TokensReasoning},
	}
	for _, c := range counts {
		if c.set && c.value < 0 {
			problems = append(problems, fmt.Sprintf("--%s must be >= 0, got %d", c.name, c.value))
		}
	}
	return problems
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
