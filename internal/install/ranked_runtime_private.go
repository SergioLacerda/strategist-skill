package install

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimeenv"
	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
)

// rankedExecutable is how a Ranked provider command is launched: a bare name
// resolved from PATH (host/custom), or an absolute private launcher plus a
// leading script argument.
type rankedExecutable struct {
	name   string
	prefix []string
}

var findHostNode = exec.LookPath

var hostNodeVersion = func(ctx context.Context, root, node string) ([]byte, error) {
	cmd, err := runtimeenv.PrivateCommand(ctx, root, node, "--version")
	if err != nil {
		return nil, fmt.Errorf("prepare host Node version command: %w", err)
	}
	return cmd.Output()
}

func (e rankedExecutable) args(rest []string) []string {
	return append(append([]string{}, e.prefix...), rest...)
}

// resolveRankedExecutable materializes the embedded, digest-verified OpenSpec
// bundle and runs it with a validated host Node. OpenSpec is never resolved
// from PATH; only the host Node executable is discovered there.
func resolveRankedExecutable(ctx context.Context, strategistDir, providerID string, contract domain.WeaponRuntime) (rankedExecutable, *domain.RankedRuntimeStateRuntime, error) {
	return resolveHostNodeRuntime(ctx, strategistDir, providerID, contract)
}

func resolveHostNodeRuntime(ctx context.Context, strategistDir, providerID string, contract domain.WeaponRuntime) (rankedExecutable, *domain.RankedRuntimeStateRuntime, error) {
	node, err := resolveHostNode(ctx, strategistDir)
	if err != nil {
		return rankedExecutable{}, nil, err
	}
	dest := filepath.Join(strategistDir, "weapon-runtime", providerID)
	// Materialize from this binary's embedded defaults, never from the copy
	// already extracted on disk. The on-disk tree carries its own
	// runtime.build.yaml, and TreeDigest deliberately excludes that file from
	// the digest it publishes, so anyone able to write the tree could rewrite
	// its certificate to match. Installation also preserves user-modified files
	// unless --force, so a stale or tampered tree survives a reinstall. The
	// embedded copy is the authority; the extracted one is not.
	defaults, ok := runtimepayload.DefaultOpenSpec()
	if !ok {
		return rankedExecutable{}, nil, fmt.Errorf("embedded OpenSpec defaults are unavailable; rebuild Strategist with the embedded skill bundle")
	}
	script, evidence, err := runtimepayload.MaterializeOpenSpec(defaults, dest)
	if err != nil {
		return rankedExecutable{}, nil, fmt.Errorf("embedded OpenSpec runtime: %w", err)
	}
	state := &domain.RankedRuntimeStateRuntime{Node: node, Script: relSlash(strategistDir, script)}
	for _, c := range evidence.Components {
		state.Components = append(state.Components, domain.RankedRuntimeStateComponent{Name: c.Name, Version: c.Version, SHA256: c.SHA256})
	}
	if err := checkOpenSpecPin(contract, state); err != nil {
		return rankedExecutable{}, nil, err
	}
	return rankedExecutable{name: node, prefix: []string{script}}, state, nil
}

// resolveHostNode discovers Node on PATH and returns its absolute path once it
// reports at least the minimum version the embedded OpenSpec bundle needs.
func resolveHostNode(ctx context.Context, strategistDir string) (string, error) {
	node, err := findHostNode("node")
	if err != nil {
		return "", &runtimeenv.ExecutableNotFoundError{Name: "node", Err: err}
	}
	versionOut, err := hostNodeVersion(ctx, strategistDir, node)
	if err != nil {
		return "", fmt.Errorf("host Node %q version: %w", node, err)
	}
	version := domain.ParseReportedVersion(versionOut)
	if !domain.VersionAtLeast(version, domain.MinimumOpenSpecNodeVersion) {
		return "", fmt.Errorf("host Node %q reports %q; Node >=%s is required to run the embedded OpenSpec bundle", node, version, domain.MinimumOpenSpecNodeVersion)
	}
	abs, err := filepath.Abs(node)
	if err != nil {
		return "", fmt.Errorf("resolve host Node %q path: %w", node, err)
	}
	return abs, nil
}

// checkOpenSpecPin fails closed when the embedded bundle is not the version
// the certified provider contract pins. An unpinned contract accepts any
// embedded bundle.
func checkOpenSpecPin(contract domain.WeaponRuntime, state *domain.RankedRuntimeStateRuntime) error {
	for _, c := range state.Components {
		if c.Name == "openspec" && contract.Version != "" && contract.Version != c.Version {
			return fmt.Errorf("error=ranked_runtime_pin_mismatch: embedded OpenSpec is %s but the provider contract pins %s", c.Version, contract.Version)
		}
	}
	return nil
}

func relSlash(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(rel)
}
