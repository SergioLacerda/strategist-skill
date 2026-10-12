package dojo

import (
	"fmt"
	"io"
	"strings"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/spf13/cobra"
)

func newCorpusScreen(deps Dependencies) *cobra.Command {
	var root string
	var onlyClean bool
	cmd := &cobra.Command{
		Use:   "corpus <case>",
		Short: "List the packages of a case's corpus with an offline screen (no call)",
		Long: `List the packages the corpus of <case> would use, with their label and the result of the deterministic
local checks (missing fields, untyped or unverified tasks, a task type that does not match its action). The screen reads
files only and never calls a provider. A package with no flag is a candidate for review, not a golden: only
'strategist dojo promote', a human act, makes it one.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCorpusScreen(cmd, deps, root, args[0], onlyClean)
		},
	}
	cmd.Flags().StringVar(&root, "root", "", rootFlagUsage)
	cmd.Flags().BoolVar(&onlyClean, "clean", false, "list only the packages no local check flags")
	return cmd
}

type screenRow struct {
	name, label string
	tasks       int
	flags       []string
}

func (r screenRow) clean() bool { return len(r.flags) == 0 }

func screenRows(corpus internaldojo.Corpus, boundary string) []screenRow {
	rows := make([]screenRow, 0, len(corpus.Sources))
	for i, source := range corpus.Sources {
		rows = append(rows, screenRow{
			name: source.Name, label: corpus.Manifest.Entries[i].Label,
			tasks: strings.Count(source.Fields["implementation_plan"], "- ["), flags: calibration.LocalFlags(boundary, source.Fields),
		})
	}
	return rows
}

func runCorpusScreen(cmd *cobra.Command, deps Dependencies, root, name string, onlyClean bool) error {
	cc, err := loadCase(deps, root, name)
	if err != nil {
		return err
	}
	cal := *cc.criteria.Calibration
	corpus, err := internaldojo.BuildCorpus(cc.basePath, cal, deps.Calibration.Reader)
	if err != nil {
		return fmt.Errorf("dojo corpus: %w", err)
	}
	return printScreen(cmd.OutOrStdout(), name, screenRows(corpus, cal.Boundary), onlyClean)
}

func cleanCount(rows []screenRow) int {
	n := 0
	for _, row := range rows {
		if row.clean() {
			n++
		}
	}
	return n
}

func printScreen(out io.Writer, name string, rows []screenRow, onlyClean bool) error {
	clean := cleanCount(rows)
	if err := say(out, "case %s: %d packages, %d with no local flag, %d flagged\n", name, len(rows), clean, len(rows)-clean); err != nil {
		return err
	}
	if err := printRows(out, rows, onlyClean); err != nil {
		return err
	}
	return say(out, "review a package, then: strategist dojo promote %s --package <name> --reviewed-by <you> --oracle \"<why it is good>\"\n", name)
}

func printRows(out io.Writer, rows []screenRow, onlyClean bool) error {
	for _, row := range rows {
		if onlyClean && !row.clean() {
			continue
		}
		if err := say(out, "  %-62s %-10s tasks=%-3d %s\n", row.name, row.label, row.tasks, flagText(row)); err != nil {
			return err
		}
	}
	return nil
}

func flagText(row screenRow) string {
	if row.clean() {
		return "clean"
	}
	return "flags: " + strings.Join(row.flags, ",")
}
