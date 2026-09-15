package main

import (
	"context"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// pluginsCmd is the parent for Strategist's plugin-enforcement CLI surface.
var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Inspect and evaluate Strategist plugin enforcement",
}

type pluginsEvaluateWriteOptions struct {
	Root   string
	Target string
}

var pluginsEvaluateWriteCmd = &cobra.Command{
	Use:   "evaluate-write",
	Short: "Report whether a write to --target is enforceably allowed under the active connector",
	Long: `Classifies --target against the mission's active.yaml base_path
(internal/plugins/policy.ClassifyWriteTarget), then checks that
classification against the native runtime connector's own enforcement
observation (internal/plugins/policy.EvaluateWrite).

This is a scriptable, deterministic tool an agent embodying Sniper can
invoke before writing, rather than relying only on the write_scope/
approved_scope prose in roles/sniper.yaml — see
.analysis/refined/20260830-skill-gaps-triage/analysis.md Cluster 1 (K03) and
.analysis/refined/20260830-skill-gaps-followup/tasks.md item A4 for why a
CLI-exposed check, not internal Go interception, is the right shape here:
Sniper is a parent-agent-embodied native role, not a Go writer this binary
can intercept.

Exits non-zero when the write is not enforceably allowed, so callers can
gate on it directly.`,
}

func runPluginsEvaluateWrite(opts pluginsEvaluateWriteOptions) error {
	if opts.Target == "" {
		return fmt.Errorf("plugins evaluate-write: --target is required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("plugins evaluate-write: %w", err)
	}
	strategistRoot, _, err := cliutil.ResolveStrategistRoot(opts.Root, cwd)
	if err != nil {
		return fmt.Errorf("plugins evaluate-write: %w", err)
	}
	basePath, err := readActiveBasePath(strategistRoot)
	if err != nil {
		return fmt.Errorf("plugins evaluate-write: %w", err)
	}

	connector := connectors.NativeRuntimeConnector{
		ConnectorID:           "strategist-native",
		ConnectorAPIVersion:   "strategist-connector-api/1",
		EnforcementObservable: true,
	}
	observation := connector.Observe(context.Background(), domain.InstalledInstance{ID: "sniper", State: "active"})

	// opts.Target is classified verbatim, relative to the project root
	// (e.g. "docs/foo.md", ".analysis/refined/x/tasks.md", "internal/foo.go")
	// — the same frame active.yaml's own base_path is written in
	// (policy.ClassifyWriteTarget's "docs/" prefix check is a literal,
	// project-root-relative comparison, not basePath-relative).
	decision := policy.EvaluateWrite(basePath, opts.Target, observation.Enforcement)
	printPluginsEvaluateWriteResult(decision)
	if !decision.Allowed {
		return fmt.Errorf("plugins evaluate-write: denied (permission=%s)", decision.Permission)
	}
	return nil
}

// readActiveBasePath reads active.yaml's raw base_path (e.g. ".analysis"),
// unresolved against the project root — policy.ClassifyWriteTarget expects
// basePath and targetPath in the same relative frame, and callers of this
// command pass --target relative to the project root.
func readActiveBasePath(strategistRoot string) (string, error) {
	activeYAMLPath, err := runtimefs.SafeJoin(strategistRoot, "active.yaml")
	if err != nil {
		return "", fmt.Errorf("resolve active.yaml path: %w", err)
	}
	raw, err := os.ReadFile(activeYAMLPath) //nolint:gosec // G304: path validated by runtimefs.SafeJoin, confined to strategistRoot
	if err != nil {
		return "", fmt.Errorf("read active.yaml: %w", err)
	}
	var cfg domain.ActiveConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("parse active.yaml: %w", err)
	}
	if cfg.BasePath == "" {
		return "", fmt.Errorf("active.yaml: base_path is empty")
	}
	return cfg.BasePath, nil
}

func printPluginsEvaluateWriteResult(decision policy.WriteDecision) {
	status := "denied"
	if decision.Allowed {
		status = "allowed"
	}
	fmt.Printf("write=%s permission=%s reason=%s\n", status, decision.Permission, decision.Reason)
}

func init() {
	opts := pluginsEvaluateWriteOptions{}
	pluginsEvaluateWriteCmd.Flags().StringVar(&opts.Root, flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	pluginsEvaluateWriteCmd.Flags().StringVar(&opts.Target, "target", "", "candidate write path to evaluate, relative to the project root (required)")
	pluginsEvaluateWriteCmd.RunE = func(_ *cobra.Command, _ []string) error {
		return runPluginsEvaluateWrite(opts)
	}
	pluginsCmd.AddCommand(pluginsEvaluateWriteCmd)
	rootCmd.AddCommand(pluginsCmd)
}
