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
	// An absolute name is a Strategist-materialized private runtime and is
	// never resolved through PATH; a bare name is the host/custom lookup.
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
		return fmt.Errorf("ranked runtime provider %q: %s: %w", providerID, domain.RankedRuntimeExecutableMissingMessage(providerID, missing.Name), err)
	}
	return fmt.Errorf("ranked runtime provider %q: %w", providerID, err)
}

func rankedRuntimeCommand(ctx context.Context, dir, name string, args ...string) (*exec.Cmd, error) {
	build := runtimeenv.Command
	if filepath.IsAbs(name) {
		build = runtimeenv.PrivateCommand
	}
	cmd, err := build(ctx, dir, name, args...)
	if err != nil {
		return nil, fmt.Errorf("prepare ranked runtime command %q: %w", name, err)
	}
	return cmd, nil
}
