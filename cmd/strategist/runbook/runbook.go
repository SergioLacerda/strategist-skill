// Package runbook owns the generic Strategist runbook command. It consumes
// runbook projections through JEWELER and has no knowledge of Treasure Chest's
// implementation package or local sidecar selector.
package runbook

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
	"github.com/SergioLacerda/strategist-skill/internal/runtime/catalog"
	"github.com/spf13/cobra"
)

const providerExecutableEnv = jeweler.ExecutableEnv

type options struct {
	root, executable, requester, mission, correlation, format string
}

type selectionRow struct {
	RunbookID string `json:"runbook_id"`
	Role      string `json:"role"`
	ChestID   string `json:"chest_id"`
	Ref       string `json:"ref"`
	Reason    string `json:"reason"`
}

// Register attaches only the provider-mediated runbook capability.
func Register(root *cobra.Command) {
	opts := options{requester: "ranger", format: "table"}
	parent := &cobra.Command{Use: "runbook", Short: "Query provider-backed runbooks for mission signals"}
	selectCmd := &cobra.Command{
		Use:   "select",
		Short: "Select applicable runbooks for mission signals",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return runSelect(cmd, opts, args) },
	}
	parent.PersistentFlags().StringVar(&opts.root, "root", "", "path to .strategist/ root (default: current workspace)")
	selectCmd.Flags().StringArray("signal", nil, "mission signal to match (repeatable; at least one required)")
	selectCmd.Flags().StringVar(&opts.format, "format", opts.format, "output format: table or json")
	selectCmd.Flags().StringVar(&opts.executable, "provider-executable", "", "explicit TREASURE-CHEST-TMP executable")
	selectCmd.Flags().StringVar(&opts.requester, "requester", opts.requester, "authorized consumer role: ranger or archivist")
	selectCmd.Flags().StringVar(&opts.mission, "mission", "", "mission identifier (required) — runbooks are selected only inside a mission")
	selectCmd.Flags().StringVar(&opts.correlation, "correlation", "", "correlation identifier")
	parent.AddCommand(selectCmd)
	root.AddCommand(parent)
}

func runSelect(cmd *cobra.Command, opts options, _ []string) error {
	signals, err := cmd.Flags().GetStringArray("signal")
	if err != nil {
		return fmt.Errorf("runbook select: read --signal: %w", err)
	}
	if len(signals) == 0 {
		return fmt.Errorf("runbook select: at least one --signal is required")
	}
	strategistDir, err := resolveRoot(opts.root)
	if err != nil {
		return err
	}
	projection, err := selectProjection(cmd, opts, strategistDir, signals)
	if err != nil {
		return err
	}
	return render(cmd, projection, opts.format)
}

func selectProjection(cmd *cobra.Command, opts options, strategistDir string, signals []string) (jeweler.Projection, error) {
	executable := opts.executable
	if executable == "" {
		executable = os.Getenv(providerExecutableEnv)
	}
	if executable == "" {
		return jeweler.Projection{}, fmt.Errorf("runbook select: JEWELER provider binding is missing; pass --provider-executable or set %s", providerExecutableEnv)
	}
	mission := opts.mission
	if mission == "" {
		return jeweler.Projection{}, fmt.Errorf("runbook select: mission identifier is required for JEWELER")
	}
	correlation := correlationID(opts.correlation, mission)
	provider, err := newProvider(strategistDir, executable)
	if err != nil {
		return jeweler.Projection{}, fmt.Errorf("runbook select: JEWELER binding: %w", err)
	}
	projection, err := (jeweler.Broker{Provider: provider}).SelectRunbooks(cmd.Context(), jeweler.RunbookSelectionRequest{
		QueryRequest: jeweler.QueryRequest{Requester: opts.requester, MissionID: mission, CorrelationID: correlation, Root: strategistDir, Limit: 3},
		Signals:      signals,
	})
	if err != nil {
		return jeweler.Projection{}, fmt.Errorf("runbook select: JEWELER: %w", err)
	}
	return projection, nil
}

func correlationID(value, mission string) string {
	if value == "" {
		return mission
	}
	return value
}

func resolveRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Clean(explicit), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("runbook select: get cwd: %w", err)
	}
	if info, statErr := os.Stat(filepath.Join(cwd, ".strategist")); statErr == nil && info.IsDir() {
		return filepath.Join(cwd, ".strategist"), nil
	}
	return cwd, nil
}

func newProvider(strategistDir, executable string) (jeweler.Provider, error) {
	raw, err := os.ReadFile(filepath.Join(strategistDir, "plugins", "catalog.yaml")) //nolint:gosec // resolved workspace root
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
	provider, err := jeweler.NewCLIProviderFromCompiledWeapon(weapon, executable)
	if err != nil {
		return nil, fmt.Errorf("create JEWELER provider: %w", err)
	}
	return provider, nil
}
