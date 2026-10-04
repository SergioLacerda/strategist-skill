package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// validateAmendFlags checks the amend flag combination before any path is
// resolved: the two value flags belong to --amend, and an amendment never consumes
// a pending analysis.
func validateAmendFlags(cmd *cobra.Command, opts NormalizeOptions) error {
	values := map[string]string{"--amends": opts.Amends, "--authorization-ref": opts.AuthorizationRef}
	if !opts.Amend {
		return requireOnlyWithAmend(map[string]string{
			"--amends": opts.Amends, "--authorization-ref": opts.AuthorizationRef,
			"--reason": opts.Reason, "--supersedes-mission-id": opts.SupersedesMissionID,
		})
	}
	if cmd.Flags().Changed("pending-analysis") {
		return fmt.Errorf("--pending-analysis cannot be used with --amend: an amendment never consumes a pending analysis")
	}
	return requireAmendValues(values)
}

// requireOnlyWithAmend rejects a value flag given without --amend.
func requireOnlyWithAmend(values map[string]string) error {
	for name, value := range values {
		if value != "" {
			return fmt.Errorf("%s requires --amend", name)
		}
	}
	return nil
}

// requireAmendValues rejects a missing value flag under --amend.
func requireAmendValues(values map[string]string) error {
	for name, value := range values {
		if value == "" {
			return fmt.Errorf("--amend requires %s", name)
		}
	}
	return nil
}

// runAmend applies an amendment through refinement.AmendOpenSpec.
func runAmend(cmd *cobra.Command, deps NormalizeDependencies, opts NormalizeOptions, basePath, runtimeRoot string) error {
	persisted, label, err := loadAmendContext(deps, opts)
	if err != nil {
		return err
	}
	result, err := application.ApplyOpenSpecAmendment(application.AmendOpenSpecRequest{
		MissionID: opts.MissionID, BasePath: basePath, RuntimeRoot: runtimeRoot, ChangeID: opts.ChangeID,
		Amends: opts.Amends, AuthorizationRef: opts.AuthorizationRef, GateLabel: label, PersistedStatus: persisted, Reason: opts.Reason, SupersedesMissionID: opts.SupersedesMissionID,
	})
	if err != nil {
		return fmt.Errorf("mission normalize-openspec: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "[Strategist] phase=archivist status=amended mission_id=%s amendment=%03d\n", opts.MissionID, result.Amendment); err != nil {
		return fmt.Errorf("mission normalize-openspec: write output: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s provider_change_id=%s refined=%s amendment=%03d status=%s\n", opts.MissionID, result.ProviderChangeID, result.RefinedPath, result.Amendment, result.Status); err != nil {
		return fmt.Errorf("mission normalize-openspec: write output: %w", err)
	}
	return nil
}

func loadAmendContext(deps NormalizeDependencies, opts NormalizeOptions) (domain.MissionEngineStatus, string, error) {
	if deps.LoadMission == nil {
		return domain.MissionEngineStatus{}, "", fmt.Errorf("mission normalize-openspec: persisted FSM state loader is unavailable")
	}
	persisted, err := deps.LoadMission(opts)
	if err != nil {
		return domain.MissionEngineStatus{}, "", fmt.Errorf("mission normalize-openspec: read persisted FSM state: %w", err)
	}
	label, err := loadAmendGateLabel(deps, opts)
	if err != nil {
		return domain.MissionEngineStatus{}, "", err
	}
	return persisted, label, nil
}

func loadAmendGateLabel(deps NormalizeDependencies, opts NormalizeOptions) (string, error) {
	if deps.GateLabel == nil {
		return "", nil
	}
	label, err := deps.GateLabel(opts)
	if err != nil {
		return "", fmt.Errorf("mission normalize-openspec: read gate label: %w", err)
	}
	return label, nil
}
