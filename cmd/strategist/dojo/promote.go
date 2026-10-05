package dojo

import (
	"fmt"
	"time"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/spf13/cobra"
)

func newPromote(deps Dependencies) *cobra.Command {
	var root, name, reviewer, oracle string
	cmd := &cobra.Command{
		Use:   "promote <case>",
		Short: "Promote a reviewed package to golden (a human act)",
		Long: `Record that a package is a golden positive for the corpus of <case>: a reviewer and an oracle (why it
is good) are required, and the promotion is bound to the package's content digest, so editing the package
voids it. The DOJO agent never runs this command.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPromote(cmd, deps, root, args[0], name, reviewer, oracle)
		},
	}
	f := cmd.Flags()
	f.StringVar(&root, "root", "", rootFlagUsage)
	f.StringVar(&name, "package", "", "package name to promote (required)")
	f.StringVar(&reviewer, "reviewed-by", "", "name of the human reviewer (required)")
	f.StringVar(&oracle, "oracle", "", "why this package is good, in one sentence (required)")
	for _, flag := range []string{"package", "reviewed-by", "oracle"} {
		_ = cmd.MarkFlagRequired(flag) //nolint:errcheck // flags are defined above
	}
	return cmd
}

func runPromote(cmd *cobra.Command, deps Dependencies, root, caseName, pkg, reviewer, oracle string) error {
	cc, err := loadCase(deps, root, caseName)
	if err != nil {
		return err
	}
	corpus, err := internaldojo.BuildCorpus(cc.basePath, *cc.criteria.Calibration, deps.Calibration.Reader)
	if err != nil {
		return fmt.Errorf("dojo promote: %w", err)
	}
	for _, entry := range corpus.Manifest.Entries {
		if entry.Name != pkg {
			continue
		}
		promotion := internaldojo.Promotion{Digest: entry.Digest, Package: pkg, ReviewedBy: reviewer, Oracle: oracle, At: time.Now().UTC().Format(time.RFC3339)}
		if err := internaldojo.AppendPromotion(cc.basePath, promotion); err != nil {
			return fmt.Errorf("dojo promote: %w", err)
		}
		return say(cmd.OutOrStdout(), "promoted %s (%s) to golden, reviewed by %s\n", pkg, entry.Digest, reviewer)
	}
	return fmt.Errorf("dojo promote: package %q is not in the corpus of case %q", pkg, caseName)
}
