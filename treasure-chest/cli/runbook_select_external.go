package treasurecli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
	"github.com/SergioLacerda/strategist-skill/internal/runtime/catalog"
	"github.com/spf13/cobra"
)

var newRunbookProvider = func(strategistDir, executable string) (jeweler.Provider, error) {
	raw, err := os.ReadFile(filepath.Join(strategistDir, "plugins", "catalog.yaml")) //nolint:gosec // runtime root is explicitly resolved by the command
	if err != nil {
		return nil, fmt.Errorf("read ROSTER catalog: %w", err)
	}
	registry, err := catalog.ParseCompiledRegistryCatalog(raw)
	if err != nil {
		return nil, fmt.Errorf("parse ROSTER catalog: %w", err)
	}
	identity := jeweler.AdmittedTreasureChestTMPIdentity()
	weapon, ok := registry.Weapon(identity.Package, identity.Version)
	if !ok {
		return nil, fmt.Errorf("ROSTER catalog has no admitted Weapon %s@%s", identity.Package, identity.Version)
	}
	return jeweler.NewCLIProviderFromCompiledWeapon(weapon, executable)
}

func runExternalRunbookSelect(cmd *cobra.Command, opts runbookSelectOptions, strategistDir string, signals []string) error {
	executable, err := resolveRunbookProviderExecutable(opts)
	if err != nil {
		return err
	}
	missionID, correlationID, err := resolveRunbookMission(cmd, opts)
	if err != nil {
		return err
	}
	provider, err := newRunbookProvider(strategistDir, executable)
	if err != nil {
		return fmt.Errorf("runbook select: JEWELER binding: %w", err)
	}
	projection, err := selectExternalRunbooks(cmd.Context(), provider, opts, strategistDir, missionID, correlationID, signals)
	if err != nil {
		return fmt.Errorf("runbook select: JEWELER: %w", err)
	}
	return renderExternalRunbookProjection(projection, opts.Format)
}

func resolveRunbookProviderExecutable(opts runbookSelectOptions) (string, error) {
	if opts.ProviderExecutable != "" {
		return opts.ProviderExecutable, nil
	}
	if executable := os.Getenv(treasureChestExecutableEnv); executable != "" {
		return executable, nil
	}
	return "", fmt.Errorf("runbook select: JEWELER provider binding is missing; pass --provider-executable or set %s", treasureChestExecutableEnv)
}

func resolveRunbookMission(cmd *cobra.Command, opts runbookSelectOptions) (string, string, error) {
	missionID := opts.MissionID
	if missionID == "" {
		if run := telemetryRunFromCmd(cmd); run != nil {
			missionID = run.Snapshot().MissionID
		}
	}
	if missionID == "" {
		return "", "", fmt.Errorf("runbook select: mission identifier is required for JEWELER")
	}
	correlationID := opts.CorrelationID
	if correlationID == "" {
		correlationID = missionID
	}
	return missionID, correlationID, nil
}

func selectExternalRunbooks(ctx context.Context, provider jeweler.Provider, opts runbookSelectOptions, root, missionID, correlationID string, signals []string) (jeweler.Projection, error) {
	projection, err := (jeweler.Broker{Provider: provider}).SelectRunbooks(ctx, jeweler.RunbookSelectionRequest{
		QueryRequest: jeweler.QueryRequest{
			Requester:     opts.Requester,
			MissionID:     missionID,
			CorrelationID: correlationID,
			Root:          root,
			Limit:         3,
		},
		Signals: signals,
	})
	if err != nil {
		return jeweler.Projection{}, fmt.Errorf("select runbooks through JEWELER: %w", err)
	}
	return projection, nil
}
