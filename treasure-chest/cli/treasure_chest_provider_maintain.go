package treasurecli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
	"github.com/spf13/cobra"
)

func runTreasureChestProviderMaintain(cmd *cobra.Command, opts treasureChestProviderMaintainOptions) error {
	root, err := resolveTreasureChestActionRoot(cmd, "treasure-chest provider maintain")
	if err != nil {
		return err
	}
	provider, err := newTreasureChestCLIProvider(opts.treasureChestProviderOptions)
	if err != nil {
		return fmt.Errorf("treasure-chest provider maintain: %w; verify the explicit provider binding", err)
	}
	result, err := maintainTreasureChestProvider(cmd, provider, opts, root)
	if err != nil {
		return err
	}
	return renderMaintenanceResult(cmd.OutOrStdout(), opts.Format, result)
}

func maintainTreasureChestProvider(cmd *cobra.Command, provider jeweler.CLIProvider, opts treasureChestProviderMaintainOptions, root string) (jeweler.MaintenanceResult, error) {
	result, err := (jeweler.Broker{Provider: provider}).Maintain(cmd.Context(), jeweler.MaintenanceRequest{
		Principal:      opts.Principal,
		Authorization:  opts.Authorization,
		Operation:      opts.Operation,
		Root:           root,
		CorrelationID:  opts.Correlation,
		IdempotencyKey: opts.Idempotency,
	})
	if err != nil {
		return jeweler.MaintenanceResult{}, fmt.Errorf("treasure-chest provider maintain: %w; no local fallback was attempted", err)
	}
	return result, nil
}

func renderMaintenanceResult(w io.Writer, format string, result jeweler.MaintenanceResult) error {
	switch format {
	case outputFormatJSON:
		if err := json.NewEncoder(w).Encode(result); err != nil {
			return fmt.Errorf("render maintenance result JSON: %w", err)
		}
		return nil
	case "", outputFormatTable:
		return renderMaintenanceTable(w, result)
	default:
		return fmt.Errorf("treasure-chest provider maintain: unknown --format %q (want table or json)", format)
	}
}

func renderMaintenanceTable(w io.Writer, result jeweler.MaintenanceResult) error {
	if _, err := fmt.Fprintf(w, "operation: %s\nreadiness: %s (%s)\n", result.Operation, result.Readiness.Status, result.Readiness.Reason); err != nil {
		return fmt.Errorf("render maintenance result: %w", err)
	}
	if strings.TrimSpace(result.Detail) == "" {
		return nil
	}
	if _, err := fmt.Fprintf(w, "detail:    %s\n", result.Detail); err != nil {
		return fmt.Errorf("render maintenance detail: %w", err)
	}
	return nil
}
