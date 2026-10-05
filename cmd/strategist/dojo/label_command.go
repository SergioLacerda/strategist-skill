package dojo

import (
	"fmt"
	"time"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/spf13/cobra"
)

// labelSpec describes a command that a person uses to label a package of a case's corpus:
// it needs the package, a reviewer and one sentence of justification, and records a decision
// bound to the package's content digest.
type labelSpec struct {
	use, short, long, verb string
	// noteFlag is the justification flag ("oracle" for a promotion, "reason" for a demotion).
	noteFlag, noteUsage string
	record              func(basePath string, entry internaldojo.CorpusEntry, pkg, reviewer, note, at string) error
	done                string
}

func newLabelCommand(deps Dependencies, spec labelSpec) *cobra.Command {
	var root, name, reviewer, note string
	cmd := &cobra.Command{
		Use: spec.use, Short: spec.short, Long: spec.long, Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabel(cmd, deps, spec, labelRequest{root: root, caseName: args[0], pkg: name, reviewer: reviewer, note: note})
		},
	}
	f := cmd.Flags()
	f.StringVar(&root, "root", "", rootFlagUsage)
	f.StringVar(&name, "package", "", "package name to "+spec.verb+" (required)")
	f.StringVar(&reviewer, "reviewed-by", "", "name of the human reviewer (required)")
	f.StringVar(&note, spec.noteFlag, "", spec.noteUsage+" (required)")
	for _, flag := range []string{"package", "reviewed-by", spec.noteFlag} {
		_ = cmd.MarkFlagRequired(flag) //nolint:errcheck // flags are defined above
	}
	return cmd
}

type labelRequest struct{ root, caseName, pkg, reviewer, note string }

func runLabel(cmd *cobra.Command, deps Dependencies, spec labelSpec, req labelRequest) error {
	cc, entry, err := packageInCorpus(deps, req.root, req.caseName, req.pkg, spec.verb)
	if err != nil {
		return err
	}
	if err := spec.record(cc.basePath, entry, req.pkg, req.reviewer, req.note, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("dojo %s: %w", spec.verb, err)
	}
	return say(cmd.OutOrStdout(), spec.done, req.pkg, entry.Digest, req.reviewer)
}

// packageInCorpus finds a package in the corpus of a case, for the commands that label it.
func packageInCorpus(deps Dependencies, root, caseName, pkg, verb string) (caseContext, internaldojo.CorpusEntry, error) {
	cc, err := loadCase(deps, root, caseName)
	if err != nil {
		return caseContext{}, internaldojo.CorpusEntry{}, err
	}
	corpus, err := internaldojo.BuildCorpus(cc.basePath, *cc.criteria.Calibration, deps.Calibration.Reader)
	if err != nil {
		return caseContext{}, internaldojo.CorpusEntry{}, fmt.Errorf("dojo %s: %w", verb, err)
	}
	for _, entry := range corpus.Manifest.Entries {
		if entry.Name == pkg {
			return cc, entry, nil
		}
	}
	return caseContext{}, internaldojo.CorpusEntry{}, fmt.Errorf("dojo %s: package %q is not in the corpus of case %q", verb, pkg, caseName)
}
