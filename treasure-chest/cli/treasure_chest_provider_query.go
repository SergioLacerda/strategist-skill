package treasurecli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
	"github.com/spf13/cobra"
)

func renderTreasureChestProviderQuery(cmd *cobra.Command, format string, projection jeweler.Projection) error {
	switch format {
	case outputFormatJSON:
		return encodeProviderQueryJSON(cmd.OutOrStdout(), projection)
	case "", outputFormatTable:
		return renderProviderQueryTable(cmd.OutOrStdout(), projection)
	default:
		return fmt.Errorf("treasure-chest provider query: unknown --format %q (want table or json)", format)
	}
}

func encodeProviderQueryJSON(w io.Writer, projection jeweler.Projection) error {
	if err := json.NewEncoder(w).Encode(projection); err != nil {
		return fmt.Errorf("render provider query JSON: %w", err)
	}
	return nil
}

func renderProviderQueryTable(w io.Writer, projection jeweler.Projection) error {
	if err := renderProviderIdentity(w, projection); err != nil {
		return err
	}
	if err := renderProviderEvidence(w, projection); err != nil {
		return err
	}
	if err := renderProviderReadiness(w, projection); err != nil {
		return err
	}
	if err := renderProviderItems(w, projection.Items); err != nil {
		return err
	}
	return renderProviderLimitations(w, projection.Limitations)
}

func renderProviderIdentity(w io.Writer, projection jeweler.Projection) error {
	if _, err := fmt.Fprintf(w, "provider:    %s@%s\n", projection.Identity.Package, projection.Identity.Version); err != nil {
		return fmt.Errorf("render provider identity: %w", err)
	}
	return nil
}

func renderProviderEvidence(w io.Writer, projection jeweler.Projection) error {
	if _, err := fmt.Fprintf(w, "api:         %s\nidentity:    %s\nprovenance:  %s\n", projection.Identity.API, projection.Identity.Digest, projection.Identity.Provenance); err != nil {
		return fmt.Errorf("render provider evidence: %w", err)
	}
	return nil
}

func renderProviderReadiness(w io.Writer, projection jeweler.Projection) error {
	if _, err := fmt.Fprintf(w, "captured_at: %s\nfreshness:   %s\nreadiness:   %s (%s)\n", projection.CapturedAt, projection.Freshness, projection.Readiness.Status, projection.Readiness.Reason); err != nil {
		return fmt.Errorf("render provider readiness: %w", err)
	}
	return nil
}

func renderProviderItems(w io.Writer, items []jeweler.Item) error {
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "  %-8s %-40s applicability=%s reason=%s\n", item.Kind, item.ID, item.Applicability, item.Reason); err != nil {
			return fmt.Errorf("render provider item: %w", err)
		}
	}
	return nil
}

func renderProviderLimitations(w io.Writer, limitations []string) error {
	for _, limitation := range limitations {
		if _, err := fmt.Fprintf(w, "limitation:  %s\n", limitation); err != nil {
			return fmt.Errorf("render provider limitation: %w", err)
		}
	}
	return nil
}
