package treasurecli

import (
	"fmt"
	"os"

	selectrunbook "github.com/SergioLacerda/strategist-skill/internal/feats/select_runbook"
	"github.com/SergioLacerda/strategist-skill/internal/runbook"
	"github.com/spf13/cobra"
)

// runbookChestID is the registered chest id for docs/runbooks in
// treasure-chests.yaml — the constant chest_id value every selection row
// carries, mirroring selected_runbooks_hint's schema
// (handoff-ranger-to-archivist.schema.yaml).
const runbookChestID = "runbooks"

type runbookSelectOptions struct {
	Signals             []string
	Format              string
	Root                string
	Strict              bool
	ProviderExecutable  string
	Requester           string
	MissionID           string
	CorrelationID       string
	LocalStagingAllowed bool
}

const treasureChestExecutableEnv = "STRATEGIST_TREASURE_CHEST_EXECUTABLE"

// runbookCmd is the parent for Strategist's runbook-corpus subcommands.
// Only `select` exists today — see
// .analysis/refined/20260805-runbook-mechanism-activation/design.md
// Decision 1 for why list/show are deliberately not built yet.
var runbookCmd = &cobra.Command{
	Use:   "runbook",
	Short: "Query provider-backed runbooks for mission signals",
}

var runbookSelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select applicable runbooks for the given mission signals",
	Long: `Routes runbook selection through the JEWELER-mediated external Treasure
Chest provider. The provider returns bounded runbook/playbook candidates with
source evidence; JEWELER validates readiness, identity and item kind before
rendering the result.

The provider executable must be supplied by --provider-executable or the
explicit STRATEGIST_TREASURE_CHEST_EXECUTABLE environment binding. There is no
implicit PATH lookup or local production fallback.
This is the select_runbook ability's concrete implementation (see
.strategist/roles/ranger.yaml#canonical.select_runbook and
contracts/narrative/03-discovery.md § Retrieval Cascade, stage 6).
Prints an empty-result message (exit 0) when the admitted provider has no
matching runbook candidate. The deprecated local sidecar selector remains
available only through the hidden --allow-local-staging flag for hermetic
regression tests and migration rehearsal.`,
	Args: cobra.NoArgs,
}

func init() {
	selectOpts := runbookSelectOptions{Format: outputFormatTable}
	runbookCmd.PersistentFlags().StringVar(&selectOpts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	runbookSelectCmd.Flags().StringArray("signal", nil, "mission signal to match against runbook applies_when entries (repeatable; at least one required)")
	runbookSelectCmd.Flags().StringVar(&selectOpts.Format, flagFormat, outputFormatTable, "output format: table or json")
	runbookSelectCmd.Flags().BoolVar(&selectOpts.Strict, "strict", false, "render the selection, then exit non-zero when any sidecar was skipped or flagged")
	if err := runbookSelectCmd.Flags().MarkHidden("strict"); err != nil {
		panic(fmt.Sprintf("hide runbook select strict flag: %v", err))
	}
	runbookSelectCmd.Flags().StringVar(&selectOpts.ProviderExecutable, "provider-executable", "", "explicit external Treasure Chest provider executable (or STRATEGIST_TREASURE_CHEST_EXECUTABLE)")
	runbookSelectCmd.Flags().StringVar(&selectOpts.Requester, "requester", "ranger", "authorized consumer role: ranger or archivist")
	runbookSelectCmd.Flags().StringVar(&selectOpts.MissionID, "mission", "", "mission identifier (defaults to the ambient mission run)")
	runbookSelectCmd.Flags().StringVar(&selectOpts.CorrelationID, "correlation", "", "correlation identifier (defaults to the mission identifier)")
	// This flag exists only for hermetic regression tests and reversible staging
	// rehearsals. Production invocations must omit it and use JEWELER.
	runbookSelectCmd.Flags().BoolVar(&selectOpts.LocalStagingAllowed, "allow-local-staging", false, "explicitly use the deprecated local selector; never use in production")
	if err := runbookSelectCmd.Flags().MarkHidden("allow-local-staging"); err != nil {
		panic(fmt.Sprintf("hide runbook select local staging flag: %v", err))
	}
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
	if !opts.LocalStagingAllowed {
		return runExternalRunbookSelect(cmd, opts, strategistDir, signals)
	}

	corpus, err := loadRunbookSidecars(strategistDir, projectRoot)
	if err != nil {
		return err
	}
	printCorpusDiagnostics(os.Stderr, corpus)
	renderErr := renderCorpusSelection(corpus, signals, opts.Format)
	if renderErr != nil {
		return renderErr
	}
	return strictCorpusError(cmd, opts, corpus)
}

// renderCorpusSelection selects among the usable candidates and renders the
// result; the "nothing to select from" messages say whether sidecars exist but
// were all skipped.
func renderCorpusSelection(corpus runbookCorpus, signals []string, format string) error {
	if len(corpus.Candidates) > 0 {
		return selectAndRenderRunbooks(corpus.Candidates, corpus.SourceDocByID, signals, format)
	}
	if skipped, _ := corpus.counts(); skipped > 0 {
		fmt.Printf("[Strategist] runbook select: no usable docs/runbooks/*.runbook.yaml sidecars (skipped=%d)\n", skipped)
		return nil
	}
	fmt.Println("[Strategist] runbook select: no docs/runbooks/*.runbook.yaml sidecars found")
	return nil
}

// strictCorpusError turns any skip or warning into a non-zero exit, after the
// selection was rendered. --strict is read from the command so a rebuilt
// options struct cannot lose it.
func strictCorpusError(cmd *cobra.Command, opts runbookSelectOptions, corpus runbookCorpus) error {
	strict, err := cmd.Flags().GetBool("strict")
	if err != nil {
		strict = opts.Strict
	}
	skipped, warnings := corpus.counts()
	if strict && skipped+warnings > 0 {
		return fmt.Errorf("runbook select: strict: skipped=%d warnings=%d", skipped, warnings)
	}
	return nil
}

func selectAndRenderRunbooks(candidates []runbook.Runbook, sourceDocByID map[string]string, signals []string, format string) error {
	selections, rejections, err := selectrunbook.Select(candidates, selectrunbook.MissionSignals(signals), selectrunbook.DefaultSelectionPolicy())
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

	switch format {
	case "", outputFormatTable:
		return renderRunbookSelectionTable(rows)
	case outputFormatJSON:
		return renderRunbookSelectionJSON(rows)
	default:
		return fmt.Errorf("runbook select: unknown --format %q (want table or json)", format)
	}
}

// printRunbookRejections logs one line per rejected candidate to stderr —
// auditability (design.md item 4) without disturbing --format json's stdout
// contract, which callers parse as a bare selection array (see
// runbook_select_test.go's TestRunbookSelect_MatchAssignsPrimaryAndSupportingWithReason).
func printRunbookRejections(rejections []selectrunbook.Rejection) {
	for _, r := range rejections {
		fmt.Fprintf(os.Stderr, "[Strategist] runbook select: rejected %s reason=%s\n", r.RunbookID, r.Reason)
	}
}
