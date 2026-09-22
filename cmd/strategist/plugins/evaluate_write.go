package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
	"github.com/spf13/cobra"
)

// EvaluateWriteOptions are the flags of `plugins evaluate-write`.
type EvaluateWriteOptions struct {
	Root   string
	Target string
}

// NewEvaluateWrite creates `plugins evaluate-write`.
func NewEvaluateWrite() *cobra.Command {
	opts := EvaluateWriteOptions{}
	cmd := &cobra.Command{
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
	cmd.Flags().StringVar(&opts.Root, cliutil.FlagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().StringVar(&opts.Target, "target", "", "candidate write path to evaluate, relative to the project root (required)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunEvaluateWrite(cmd.OutOrStdout(), opts)
	}
	return cmd
}

// RunEvaluateWrite classifies opts.Target against the active write scope and
// the native connector's enforcement observation.
func RunEvaluateWrite(out io.Writer, opts EvaluateWriteOptions) error {
	if opts.Target == "" {
		return fmt.Errorf("plugins evaluate-write: --target is required")
	}
	strategistRoot, scope, err := activeWriteScope(opts.Root)
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
	// — the same frame active.yaml's own base_path is written in.
	decision := policy.EvaluateWriteInScope(scope, filepath.Join(filepath.Dir(strategistRoot), opts.Target), observation.Enforcement)
	status := "denied"
	if decision.Allowed {
		status = "allowed"
	}
	if _, err := fmt.Fprintf(out, "write=%s permission=%s reason=%s\n", status, decision.Permission, decision.Reason); err != nil {
		return fmt.Errorf("plugins evaluate-write: write result: %w", err)
	}
	if !decision.Allowed {
		return fmt.Errorf("plugins evaluate-write: denied (permission=%s)", decision.Permission)
	}
	return nil
}

// activeWriteScope resolves the .strategist root from rootFlag (or the current
// directory) and the write scope its active.yaml declares.
func activeWriteScope(rootFlag string) (string, policy.WriteScope, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", policy.WriteScope{}, fmt.Errorf("%w", err)
	}
	strategistRoot, _, err := cliutil.ResolveStrategistRoot(rootFlag, cwd)
	if err != nil {
		return "", policy.WriteScope{}, fmt.Errorf("%w", err)
	}
	cfg, err := cliutil.LoadActiveConfig(strategistRoot)
	if err != nil {
		return "", policy.WriteScope{}, fmt.Errorf("%w", err)
	}
	scope, err := policy.WriteScopeFromActive(strategistRoot, cfg)
	if err != nil {
		return "", policy.WriteScope{}, fmt.Errorf("%w", err)
	}
	return strategistRoot, scope, nil
}
