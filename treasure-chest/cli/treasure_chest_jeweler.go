package treasurecli

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
	"github.com/spf13/cobra"
)

// treasureChestProviderOptions contains only the explicit external binding.
// Keeping it separate from treasureChestOptions prevents the compatibility
// status command from accidentally discovering or invoking a provider.
type treasureChestProviderOptions struct {
	Executable string
	Package    string
	Version    string
	API        string
	Digest     string
	Provenance string
	Format     string
}

type treasureChestProviderQueryOptions struct {
	treasureChestProviderOptions
	Requester    string
	MissionID    string
	Correlation  string
	Intent       string
	Concepts     []string
	Symbols      []string
	Limit        int
	FreshnessTTL string
}

type treasureChestProviderMaintainOptions struct {
	treasureChestProviderOptions
	Principal     string
	Authorization string
	Operation     string
	Correlation   string
	Idempotency   string
}

var treasureChestProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Use the explicitly bound external Treasure Chest provider",
	Long: `Use JEWELER as the mediated boundary for an external Treasure Chest provider.
This namespace never falls back to the in-repository implementation. The
provider executable and immutable identity evidence must be supplied explicitly
for every invocation.`,
}

var treasureChestProviderQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Run a bounded read-only query through JEWELER",
	Args:  cobra.NoArgs,
}

var treasureChestProviderMaintainCmd = &cobra.Command{
	Use:   "maintain",
	Short: "Run an explicitly authorized JEWELER maintenance operation",
	Args:  cobra.NoArgs,
}

func init() {
	queryOpts := treasureChestProviderQueryOptions{
		treasureChestProviderOptions: treasureChestProviderOptions{
			Package: "treasure-chest-tmp",
			Version: "0.1.0",
			API:     "knowledge-api/v1",
			Format:  outputFormatTable,
		},
		Limit: 20,
	}
	attachTreasureChestProviderFlags(treasureChestProviderQueryCmd, &queryOpts.treasureChestProviderOptions)
	treasureChestProviderQueryCmd.Flags().StringVar(&queryOpts.Requester, "requester", "", "authorized consumer role: ranger or archivist")
	treasureChestProviderQueryCmd.Flags().StringVar(&queryOpts.MissionID, "mission", "", "mission identifier")
	treasureChestProviderQueryCmd.Flags().StringVar(&queryOpts.Correlation, "correlation", "", "correlation identifier")
	treasureChestProviderQueryCmd.Flags().StringVar(&queryOpts.Intent, "intent", "", "high-level retrieval intent")
	treasureChestProviderQueryCmd.Flags().StringArrayVar(&queryOpts.Concepts, "concept", nil, "concept keyword to match (repeatable)")
	treasureChestProviderQueryCmd.Flags().StringArrayVar(&queryOpts.Symbols, "symbol", nil, "symbol keyword to match (repeatable)")
	treasureChestProviderQueryCmd.Flags().IntVar(&queryOpts.Limit, "limit", queryOpts.Limit, "maximum number of returned items (1-100)")
	treasureChestProviderQueryCmd.Flags().StringVar(&queryOpts.FreshnessTTL, "freshness-ttl", "", "requested freshness TTL (for example, 15m)")
	treasureChestProviderQueryCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runTreasureChestProviderQuery(cmd, queryOpts)
	}

	maintainOpts := treasureChestProviderMaintainOptions{
		treasureChestProviderOptions: treasureChestProviderOptions{
			Package: "treasure-chest-tmp",
			Version: "0.1.0",
			API:     "knowledge-api/v1",
			Format:  outputFormatTable,
		},
	}
	attachTreasureChestProviderFlags(treasureChestProviderMaintainCmd, &maintainOpts.treasureChestProviderOptions)
	treasureChestProviderMaintainCmd.Flags().StringVar(&maintainOpts.Principal, "principal", "", "maintenance principal")
	treasureChestProviderMaintainCmd.Flags().StringVar(&maintainOpts.Authorization, "authorization", "", "must equal jeweler:maintenance/v1")
	treasureChestProviderMaintainCmd.Flags().StringVar(&maintainOpts.Operation, "operation", "", "maintenance operation: index, probe, or health")
	treasureChestProviderMaintainCmd.Flags().StringVar(&maintainOpts.Correlation, "correlation", "", "correlation identifier")
	treasureChestProviderMaintainCmd.Flags().StringVar(&maintainOpts.Idempotency, "idempotency-key", "", "required for migrate and reconcile")
	treasureChestProviderMaintainCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runTreasureChestProviderMaintain(cmd, maintainOpts)
	}

	treasureChestProviderCmd.AddCommand(treasureChestProviderQueryCmd, treasureChestProviderMaintainCmd)
	treasureChestCmd.AddCommand(treasureChestProviderCmd)
}

func attachTreasureChestProviderFlags(cmd *cobra.Command, opts *treasureChestProviderOptions) {
	cmd.Flags().StringVar(&opts.Executable, "provider-executable", "", "explicit external Treasure Chest provider executable")
	cmd.Flags().StringVar(&opts.Package, "provider-package", opts.Package, "provider package identity")
	cmd.Flags().StringVar(&opts.Version, "provider-version", opts.Version, "provider version identity")
	cmd.Flags().StringVar(&opts.API, "provider-api", opts.API, "provider API identity")
	cmd.Flags().StringVar(&opts.Digest, "provider-digest", "", "immutable provider digest evidence")
	cmd.Flags().StringVar(&opts.Provenance, "provider-provenance", "", "provider provenance evidence")
	cmd.Flags().StringVar(&opts.Format, flagFormat, opts.Format, "output format: table or json")
}

func newTreasureChestCLIProvider(opts treasureChestProviderOptions) (jeweler.CLIProvider, error) {
	identity := jeweler.ProviderIdentity{
		Package:    opts.Package,
		Version:    opts.Version,
		API:        opts.API,
		Digest:     opts.Digest,
		Provenance: opts.Provenance,
	}
	provider, err := jeweler.NewCLIProviderFromBinding(jeweler.ProviderBinding{
		Role:        jeweler.RoleID,
		Slot:        jeweler.SupportSlot,
		RuntimeKind: jeweler.BindingRuntimeKind,
		HostAPI:     jeweler.BindingHostAPI,
		ConnectorID: jeweler.BindingConnectorID,
		Entrypoint:  jeweler.BindingEntrypoint,
		Executable:  opts.Executable,
		Identity:    identity,
	})
	if err != nil {
		return jeweler.CLIProvider{}, fmt.Errorf("create JEWELER provider binding: %w", err)
	}
	return provider, nil
}

func runTreasureChestProviderQuery(cmd *cobra.Command, opts treasureChestProviderQueryOptions) error {
	root, err := resolveTreasureChestActionRoot(cmd, "treasure-chest provider query")
	if err != nil {
		return err
	}
	provider, err := newTreasureChestCLIProvider(opts.treasureChestProviderOptions)
	if err != nil {
		return fmt.Errorf("treasure-chest provider query: %w; verify the explicit provider binding", err)
	}
	projection, err := (jeweler.Broker{Provider: provider}).Query(cmd.Context(), jeweler.QueryRequest{
		Requester:     opts.Requester,
		MissionID:     opts.MissionID,
		CorrelationID: opts.Correlation,
		Root:          root,
		Intent:        opts.Intent,
		Concepts:      opts.Concepts,
		Symbols:       opts.Symbols,
		Limit:         opts.Limit,
		FreshnessTTL:  opts.FreshnessTTL,
	})
	if err != nil {
		return fmt.Errorf("treasure-chest provider query: %w; verify the explicit provider binding and run its status command", err)
	}
	return renderTreasureChestProviderQuery(cmd, opts.Format, projection)
}
