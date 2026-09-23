package check

import "github.com/spf13/cobra"

var (
	checkRoot                      string
	checkStrict                    bool
	checkSimulate                  bool
	checkJSON                      bool
	checkPrintContentByLang        string
	checkPrintContentByLangPersona string
	checkConfirmChatLanguage       string
	readGitConflictedPaths         = readGitConflictedPathsFromWorktree
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Pre-mission slot validation",
	Long: `Validate that slot providers declared in active.yaml are installed and
satisfy their risk_score contracts.

Checks performed:
  - active.yaml is present and parseable
  - For each slot (discovery, refinement, execution):
      • skills/<provider>/skill.yaml exists (skill provider), OR
        roles/<provider>.yaml exists with matching slot field (native role)
      • skill providers must declare the correct risk_score for the slot contract:
        discovery/refinement → write_analysis, execution → controlled
      • native roles are accepted by slot field match; no risk_score check
  - discovery and refinement must have one valid persisted weapon binding in plugins.lock
  - role affinity and active.yaml/plugins.lock parity are validated fail-closed`,
	RunE: func(cmd *cobra.Command, _ []string) error { return runCheck(cmd) },
}
