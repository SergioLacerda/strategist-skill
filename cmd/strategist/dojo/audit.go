package dojo

import (
	"fmt"
	"io"
	"sort"
	"time"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/spf13/cobra"
)

type auditFlags struct {
	root, record, verdict, by, note string
	sample                          int
}

func newAudit(deps Dependencies) *cobra.Command {
	var flags auditFlags
	cmd := &cobra.Command{
		Use:   "audit <case>",
		Short: "Sample generated negatives for review, or record an audit verdict",
		Long: `Without --record, print a deterministic sample of the generated negatives (operator, band, expected
criteria and the names of the fields that changed; never content) for a reviewer to check. With --record <case-id>
--verdict agree|disagree, append the verdict to the case's audit log. A disagreement excludes the case from the
metrics and appears in the lesson; an audit never relabels a case, and a stage 2 case counts only after an agree.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return runAudit(cmd, deps, flags, args[0]) },
	}
	f := cmd.Flags()
	f.StringVar(&flags.root, "root", "", rootFlagUsage)
	f.IntVar(&flags.sample, "sample", 10, "number of generated negatives to sample")
	f.StringVar(&flags.record, "record", "", "case id to record a verdict for")
	f.StringVar(&flags.verdict, "verdict", "", "agree or disagree (with --record)")
	f.StringVar(&flags.by, "by", "", "reviewer name (with --record)")
	f.StringVar(&flags.note, "note", "", "why, in one sentence (with --record)")
	return cmd
}

func runAudit(cmd *cobra.Command, deps Dependencies, flags auditFlags, name string) error {
	cc, err := loadCase(deps, flags.root, name)
	if err != nil {
		return err
	}
	if flags.record != "" {
		entry := internaldojo.AuditEntry{CaseID: flags.record, Verdict: flags.verdict, By: flags.by, Note: flags.note}
		if err := internaldojo.RecordAudit(cc.paths.AuditPath, entry, time.Now()); err != nil {
			return fmt.Errorf("dojo audit: %w", err)
		}
		return say(cmd.OutOrStdout(), "recorded %s for %s\n", flags.verdict, flags.record)
	}
	cal := *cc.criteria.Calibration
	corpus, err := internaldojo.BuildCorpus(cc.basePath, cal, deps.Calibration.Reader)
	if err != nil {
		return fmt.Errorf("dojo audit: %w", err)
	}
	cases := internaldojo.BuildCases(corpus, cal)
	return printSample(cmd.OutOrStdout(), cases, internaldojo.SampleForAudit(cases, flags.sample, cal.Seed))
}

func printSample(out io.Writer, all, sample []calibration.Case) error {
	byID := map[string]calibration.Case{}
	for _, c := range all {
		byID[c.ID] = c
	}
	for _, c := range sample {
		if err := say(out, "%s\toperator=%s stage=%d band=%s expect=%s criteria=%v changed=%v\n", c.ID, c.Operator, c.Stage, c.Band, c.Expect, c.Criteria, changedFields(byID[c.ParentID].Fields, c.Fields)); err != nil {
			return err
		}
	}
	return nil
}

// changedFields names the fields whose text differs from the parent's.
func changedFields(parent, child calibration.Fields) []string {
	var names []string
	for name, value := range child {
		if parent[name] != value {
			names = append(names, name)
		}
	}
	for name := range parent {
		if _, kept := child[name]; !kept {
			names = append(names, name+"(removed)")
		}
	}
	sort.Strings(names)
	return names
}
