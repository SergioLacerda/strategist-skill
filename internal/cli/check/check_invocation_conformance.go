package check

import "github.com/spf13/cobra"

// conformanceTree is the command tree of the running binary, set by runCheck
// from cmd.Root() so the invocation-conformance advisory checks agent-facing
// command lines against the commands this binary actually ships.
var conformanceTree *cobra.Command

// invocationConformanceAdvisories reports every `strategist …` command line in
// the agent-facing runtime texts that the installed binary would reject: an
// unknown command, an unknown flag, or a missing required flag. A line with no
// flag is a mention, not an invocation, and is never checked for required
// flags. The result is advisory: it never changes readiness or exit status.
func invocationConformanceAdvisories(root string, tree *cobra.Command) []string {
	if tree == nil {
		return nil
	}
	var advisories []string
	seen := make(map[string]bool)
	for _, rel := range conformanceFiles(root) {
		advisories = appendUnique(advisories, seen, fileConformanceAdvisories(root, tree, rel)...)
	}
	return advisories
}
