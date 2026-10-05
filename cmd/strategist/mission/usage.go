package mission

import (
	"encoding/json"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/missionview"
	"github.com/spf13/cobra"
)

// NewUsage builds `mission usage`: the per-phase lifecycle and token usage
// projection of one mission. It reads the same shared aggregation as
// `mission view` and never advances any state. Wave 1 reports durations only;
// token columns stay unavailable until usage observations exist.
func NewUsage(deps ViewDependencies) *cobra.Command {
	var root, missionID string
	var asJSON, details, timeline bool
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show per-phase lifecycle and token usage for one mission (read-only)",
		Long: "Show start, end, duration, runs and status per phase for one mission, with the quality of every " +
			"number stated. Unobserved values print as an unavailable marker, never as zero. The legacy " +
			"provider-reported ledger is shown on its own row, outside any total.",
	}
	f := cmd.Flags()
	f.StringVar(&root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&missionID, "mission-id", "", "mission identifier (required)")
	f.BoolVar(&asJSON, "json", false, "emit strategist-mission-usage/v1 JSON")
	f.BoolVar(&details, "details", false, "list every phase run and the legacy usage records")
	f.BoolVar(&timeline, "timeline", false, "add the merged chronology of lifecycle markers and usage observations")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunUsage(cmd, deps, root, missionID, usageFlags{asJSON: asJSON, details: details, timeline: timeline})
	}
	return cmd
}

type usageFlags struct{ asJSON, details, timeline bool }

// RunUsage renders the usage projection as human text or JSON.
func RunUsage(cmd *cobra.Command, deps ViewDependencies, rootInput, missionID string, flags usageFlags) error {
	if err := deps.RequireMissionID(missionID); err != nil {
		return err
	}
	root, _, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return fmt.Errorf("mission usage: %w", err)
	}
	_, status, err := deps.Load(root, missionID)
	if err != nil {
		return fmt.Errorf("mission usage: %w", err)
	}
	if _, err := application.QueryMission(status); err != nil {
		return fmt.Errorf("mission usage: %w", err)
	}
	sources := loadUsageSources(root, status.MissionID)
	tokens, tokensErr := readMissionTokenUsage(root, status.MissionID)
	view := missionview.BuildUsage(missionview.UsageInput{
		Status: status, Lifecycle: sources.lifecycle, LifecycleError: sources.lifecycleErr,
		Usage: sources.usage, UsageError: sources.usageErr, KnownInvocations: sources.invocationIDs,
		TokenUsage: tokens, TokenUsageError: tokensErr, WithChronology: flags.timeline,
	})
	return renderUsage(cmd, view, flags.asJSON, flags.details)
}

func renderUsage(cmd *cobra.Command, view missionview.UsageView, asJSON, details bool) error {
	if asJSON {
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(view); err != nil {
			return fmt.Errorf("mission usage: write output: %w", err)
		}
		return nil
	}
	if err := missionview.RenderUsageHuman(cmd.OutOrStdout(), view, details); err != nil {
		return fmt.Errorf("mission usage: write output: %w", err)
	}
	return nil
}
