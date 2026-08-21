package check

import "github.com/spf13/cobra"

func init() {
	checkCmd.Flags().StringVar(&checkRoot, "root", "", "path to .strategist/ root (default: .strategist)")
	checkCmd.Flags().BoolVar(&checkStrict, "strict", false, "additionally require compiled artifacts to exist and match the recorded manifest hashes")
	checkCmd.Flags().BoolVar(&checkSimulate, "simulate", false, "print a readiness report (per-slot/persona status) instead of the pass/fail banner; never invokes providers or mutates state")
	checkCmd.Flags().StringVar(&checkPrintContentByLang, "print-content-by-lang", "", "print personas.<persona>.content_by_lang.<lang> and phase_announcements.<lang> from the compiled artifact as JSON and exit; use this instead of reading persona YAML directly to resolve non-English chat templates and mission narration lines")
	checkCmd.Flags().StringVar(&checkPrintContentByLangPersona, "persona", "", "persona/mode to read with --print-content-by-lang (default: active.yaml's mode)")
}

// Register attaches this package's top-level commands (check, check-stale)
// onto root. Called once from cmd/strategist's own init(), mirroring how
// every other feature area self-registers onto rootCmd — see
// cmd/strategist/check_register.go and internal/treasurecli's own Register.
func Register(root *cobra.Command) {
	root.AddCommand(checkCmd)
	root.AddCommand(checkStaleCmd)
}
