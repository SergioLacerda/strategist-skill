package dojo

import (
	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/spf13/cobra"
)

func newDemote(deps Dependencies) *cobra.Command {
	return newLabelCommand(deps, labelSpec{
		use:   "demote <case>",
		short: "Withdraw a package from the golden set (a human act)",
		long: `Record that a golden package no longer is: a reviewer and a reason (what in the package does not meet
the contract) are required. Promotions are append-only, so the demotion is its own record, bound to the package's
content digest; promoting the same content again makes it golden again. The DOJO agent never runs this command.`,
		verb: "demote", noteFlag: "reason", noteUsage: "what in the package does not meet the contract, in one sentence",
		record: func(basePath string, entry internaldojo.CorpusEntry, pkg, reviewer, note, at string) error {
			return internaldojo.AppendDemotion(basePath, internaldojo.Demotion{Digest: entry.Digest, Package: pkg, ReviewedBy: reviewer, Reason: note, At: at})
		},
		done: "demoted %s (%s), reviewed by %s: it is a candidate again\n",
	})
}
