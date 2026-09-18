// Package runtimeenv builds deterministic environments for provider-owned
// subprocesses. Provider runtimes must not inherit arbitrary host state.
package runtimeenv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Command resolves name before applying the restricted environment so the
// executable lookup still honors the operator's PATH without passing the rest
// of the host environment into the provider.
func Command(ctx context.Context, dir, name string, args ...string) (*exec.Cmd, error) {
	executable, err := exec.LookPath(name)
	if err != nil {
		return nil, fmt.Errorf("resolve provider executable %q: %w", name, err)
	}
	cmd := exec.CommandContext(ctx, executable, args...) //nolint:gosec // executable is resolved through PATH before the restricted environment is applied
	cmd.Dir = dir
	cmd.Env = ForRoot(dir)
	return cmd, nil
}

// ForRoot returns the minimal environment required for a local provider
// command. HOME/XDG directories are redirected below the runtime root so a
// provider cannot silently consult or modify the operator's global state.
func ForRoot(root string) []string {
	path := os.Getenv("PATH")
	if path == "" {
		path = "/usr/bin:/bin"
	}
	home := filepath.Join(root, ".provider-home")
	config := filepath.Join(root, ".provider-config")
	cache := filepath.Join(root, ".provider-cache")
	data := filepath.Join(root, ".provider-data")
	return []string{
		"PATH=" + path,
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_CONFIG_HOME=" + config,
		"XDG_CACHE_HOME=" + cache,
		"XDG_DATA_HOME=" + data,
	}
}
