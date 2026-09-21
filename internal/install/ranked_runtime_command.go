package install

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimeenv"
)

// runRankedRuntimeCommand is injectable so install tests can exercise the
// bootstrap contract without depending on a host OpenSpec installation.
var runRankedRuntimeCommand = func(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	// The executable and arguments are supplied by the trusted runtime
	// contract; provider roots are validated before reaching this adapter.
	// All Ranked launchers are absolute: either a Strategist-materialized Node
	// or the host Node resolved during installation. Neither path resolves an
	// OpenSpec executable through PATH.
	cmd, err := rankedRuntimeCommand(ctx, dir, name, args...)
	if err != nil {
		return nil, fmt.Errorf("prepare provider command: %w", err)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// rankedRuntimeBootstrapError replaces the opaque exec lookup failure with the
// cataloged, actionable diagnostic while keeping the cause in the chain.
func rankedRuntimeBootstrapError(providerID string, err error) error {
	var missing *runtimeenv.ExecutableNotFoundError
	if errors.As(err, &missing) {
		return fmt.Errorf("ranked runtime provider %q: reason=%s: %s: %w", providerID, domain.ReasonRankedRuntimeExecutableMissing, domain.RankedRuntimeExecutableMissingMessage(providerID, missing.Name), err)
	}
	return fmt.Errorf("ranked runtime provider %q: %w", providerID, err)
}

// rankedRuntimeCommand fails closed on a non-absolute launcher. Every Ranked
// launcher is absolute — a Strategist-materialized Node or the host Node
// resolved at install time — so a bare name means the resolver is broken. It is
// never degraded into a PATH lookup: that is the drift this runtime exists to
// prevent, and it would surface as a silently working install on any machine
// that happens to carry an `openspec` executable.
func rankedRuntimeCommand(ctx context.Context, dir, name string, args ...string) (*exec.Cmd, error) {
	if !filepath.IsAbs(name) {
		return nil, fmt.Errorf("prepare ranked runtime command %q: launcher must be an absolute path resolved by the ranked runtime resolver, not a name resolved from PATH", name)
	}
	cmd, err := runtimeenv.PrivateCommand(ctx, dir, name, args...)
	if err != nil {
		return nil, fmt.Errorf("prepare ranked runtime command %q: %w", name, err)
	}
	return cmd, nil
}
