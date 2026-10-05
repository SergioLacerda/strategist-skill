package dojo

import (
	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/spf13/cobra"
)

func newPromote(deps Dependencies) *cobra.Command {
	return newLabelCommand(deps, labelSpec{
		use:   "promote <case>",
		short: "Promote a reviewed package to golden (a human act)",
		long: `Record that a package is a golden positive for the corpus of <case>: a reviewer and an oracle (why it
is good) are required, and the promotion is bound to the package's content digest, so editing the package
voids it. The DOJO agent never runs this command.`,
		verb: "promote", noteFlag: "oracle", noteUsage: "why this package is good, in one sentence",
		record: func(basePath string, entry internaldojo.CorpusEntry, pkg, reviewer, note, at string) error {
			return internaldojo.AppendPromotion(basePath, internaldojo.Promotion{Digest: entry.Digest, Package: pkg, ReviewedBy: reviewer, Oracle: note, At: at})
		},
		done: "promoted %s (%s) to golden, reviewed by %s\n",
	})
}
