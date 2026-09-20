// Package runtimeenv builds deterministic environments for provider-owned
// subprocesses. Provider runtimes must not inherit arbitrary host state.
package runtimeenv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ExecutableNotFoundError reports that a provider executable could not be
// resolved. It is typed so callers can turn the opaque exec failure into an
// actionable, cataloged diagnostic instead of matching on error text.
type ExecutableNotFoundError struct {
	Name string
	Err  error
}

func (e *ExecutableNotFoundError) Error() string {
	return fmt.Sprintf("resolve provider executable %q: %v", e.Name, e.Err)
}

func (e *ExecutableNotFoundError) Unwrap() error { return e.Err }

// Command resolves name before applying the restricted environment so the
// executable lookup still honors the operator's PATH without passing the rest
// of the host environment into the provider.
func Command(ctx context.Context, dir, name string, args ...string) (*exec.Cmd, error) {
	executable, err := exec.LookPath(name)
	if err != nil {
		return nil, &ExecutableNotFoundError{Name: name, Err: err}
	}
	cmd := exec.CommandContext(ctx, executable, args...) //nolint:gosec // executable is resolved through PATH before the restricted environment is applied
	cmd.Dir = dir
	cmd.Env = ForRoot(dir)
	return cmd, nil
}

// PrivateCommand runs an executable that Strategist itself materialized. The
// path must be absolute and present: there is no PATH lookup, and the
// subprocess PATH is only the executable's own directory, so a Ranked runtime
// can never resolve a tool from the client's machine.
func PrivateCommand(ctx context.Context, dir, executable string, args ...string) (*exec.Cmd, error) {
	if !filepath.IsAbs(executable) {
		return nil, fmt.Errorf("private runtime executable must be absolute, got %q", executable)
	}
	if info, err := os.Stat(executable); err != nil || info.IsDir() {
		if err == nil {
			err = fmt.Errorf("%s is a directory", executable)
		}
		return nil, &ExecutableNotFoundError{Name: executable, Err: err}
	}
	cmd := exec.CommandContext(ctx, executable, args...) //nolint:gosec // executable is a Strategist-materialized private runtime path
	cmd.Dir = dir
	cmd.Env = withPath(ForRoot(dir), filepath.Dir(executable))
	return cmd, nil
}

func withPath(env []string, path string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if !strings.HasPrefix(kv, "PATH=") {
			out = append(out, kv)
		}
	}
	return append(out, "PATH="+path)
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
	return append([]string{
		"PATH=" + path,
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_CONFIG_HOME=" + config,
		"XDG_CACHE_HOME=" + cache,
		"XDG_DATA_HOME=" + data,
	}, platformEnv(runtime.GOOS, os.Getenv)...)
}

// windowsStartupVariables are the only host variables a Windows provider
// process (Node, cmd.exe shims) needs to start; user-profile, npm and
// OpenSpec configuration stay excluded.
var windowsStartupVariables = []string{"SystemRoot", "TEMP", "TMP", "ComSpec", "PATHEXT"}

// platformEnv returns the operating-system variables required to start a
// process on goos. It is empty off Windows and skips variables that are unset.
func platformEnv(goos string, getenv func(string) string) []string {
	if goos != "windows" {
		return nil
	}
	var env []string
	for _, name := range windowsStartupVariables {
		if value := getenv(name); value != "" {
			env = append(env, name+"="+value)
		}
	}
	return env
}
