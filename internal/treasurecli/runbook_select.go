package treasurecli

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
	"github.com/spf13/cobra"
)

// runbookChestID is the registered chest id for docs/runbooks in
// treasure-chests.yaml — the constant chest_id value every selection row
// carries, mirroring selected_runbooks_hint's schema
// (handoff-ranger-to-archivist.schema.yaml).
const runbookChestID = "runbooks"

// runbookSidecarGlob is the same directory/pattern
// internal/treasure.ScanRunbookDirectory's sibling mechanism scans for
// Potions, but for the typed sidecar (*.runbook.yaml) instead of the
// canonical markdown — see docs/runbooks/README.md.
const runbookSidecarGlob = "docs/runbooks/*.runbook.yaml"

type runbookSelectOptions struct {
	Signals []string
	Format  string
	Root    string
}

// runbookCmd is the parent for Strategist's runbook-corpus subcommands.
// Only `select` exists today — see
// .analysis/refined/20260805-runbook-mechanism-activation/design.md
// Decision 1 for why list/show are deliberately not built yet.
var runbookCmd = &cobra.Command{
	Use:   "runbook",
	Short: "Operate on the typed docs/runbooks/*.runbook.yaml corpus",
}

var runbookSelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select applicable runbooks for the given mission signals",
	Long: `Scores docs/runbooks/*.runbook.yaml sidecars against mission signals via
internal/runbook.Select(), bounded by internal/runbook.DefaultSelectionPolicy()
(at most one primary, at most two supporting runbooks, each with a non-empty
match reason). This is the select_runbook ability's concrete implementation
(see .strategist/roles/ranger.yaml#canonical.select_runbook and
contracts/narrative/03-discovery.md § Retrieval Cascade, stage 6).

Prints an empty-result message (exit 0) when no runbook sidecar exists yet,
or when none match the given signals — neither case is an error.`,
	Args: cobra.NoArgs,
}

func init() {
	selectOpts := runbookSelectOptions{Format: outputFormatTable}
	runbookCmd.PersistentFlags().StringVar(&selectOpts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	runbookSelectCmd.Flags().StringArray("signal", nil, "mission signal to match against runbook applies_when entries (repeatable; at least one required)")
	runbookSelectCmd.Flags().StringVar(&selectOpts.Format, flagFormat, outputFormatTable, "output format: table or json")
	runbookSelectCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runRunbookSelect(cmd, args, selectOpts)
	}

	runbookCmd.AddCommand(runbookSelectCmd)
}

// Register attaches this package's top-level commands (treasure-chest, runbook)
// onto root. Called once from cmd/strategist's own init(), mirroring how every
// other feature area self-registers onto rootCmd — see
// cmd/strategist/treasurecli_register.go.
func Register(root *cobra.Command) {
	root.AddCommand(treasureChestCmd)
	root.AddCommand(runbookCmd)
}

func runRunbookSelect(cmd *cobra.Command, _ []string, opts runbookSelectOptions) error {
	if run := telemetryRunFromCmd(cmd); run != nil {
		run.SetSilent()
	}
	opts.Format = stringFlag(cmd, flagFormat, opts.Format)
	signals, err := cmd.Flags().GetStringArray("signal")
	if err != nil {
		return fmt.Errorf("runbook select: read --signal: %w", err)
	}
	if len(signals) == 0 {
		return fmt.Errorf("runbook select: at least one --signal is required")
	}

	strategistDir, projectRoot, err := resolveRunbookActionRoot(cmd, opts.Root)
	if err != nil {
		return err
	}

	candidates, sourceDocByID, err := loadRunbookSidecars(strategistDir, projectRoot)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		fmt.Println("[Strategist] runbook select: no docs/runbooks/*.runbook.yaml sidecars found")
		return nil
	}

	selections, rejections, err := runbook.Select(candidates, runbook.MissionSignals(signals), runbook.DefaultSelectionPolicy())
	if err != nil {
		return fmt.Errorf("runbook select: %w", err)
	}
	printRunbookRejections(rejections)
	if len(selections) == 0 {
		fmt.Println("[Strategist] runbook select: no runbook matched the given signals")
		return nil
	}

	rows := make([]runbookSelectionRow, 0, len(selections))
	for _, s := range selections {
		rows = append(rows, runbookSelectionRow{
			RunbookID: s.RunbookID,
			Role:      string(s.Role),
			ChestID:   runbookChestID,
			Ref:       sourceDocByID[s.RunbookID],
			Reason:    s.Reason,
		})
	}
	sortRunbookSelectionRows(rows)

	switch opts.Format {
	case "", outputFormatTable:
		return renderRunbookSelectionTable(rows)
	case outputFormatJSON:
		return renderRunbookSelectionJSON(rows)
	default:
		return fmt.Errorf("runbook select: unknown --format %q (want table or json)", opts.Format)
	}
}

// printRunbookRejections logs one line per rejected candidate to stderr —
// auditability (design.md item 4) without disturbing --format json's stdout
// contract, which callers parse as a bare selection array (see
// runbook_select_test.go's TestRunbookSelect_MatchAssignsPrimaryAndSupportingWithReason).
func printRunbookRejections(rejections []runbook.Rejection) {
	for _, r := range rejections {
		fmt.Fprintf(os.Stderr, "[Strategist] runbook select: rejected %s reason=%s\n", r.RunbookID, r.Reason)
	}
}
