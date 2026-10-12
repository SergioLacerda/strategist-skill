package rankedbinding

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtime/env"
	"github.com/SergioLacerda/strategist-skill/internal/runtime/payload"
)

// RankedExecutable is how a Ranked provider command is launched: a bare name
// resolved from PATH (host/custom), or an absolute private launcher plus a
// leading script argument.
type RankedExecutable struct {
	Name   string
	Prefix []string
}

// FindHostNode locates the host Node executable; tests replace it.
var FindHostNode = exec.LookPath

// HostNodeVersion reports the version output of node; tests replace it.
var HostNodeVersion = func(ctx context.Context, root, node string) ([]byte, error) {
	cmd, err := env.PrivateCommand(ctx, root, node, "--version")
	if err != nil {
		return nil, fmt.Errorf("prepare host Node version command: %w", err)
	}
	return cmd.Output()
}

// Args returns the launcher prefix followed by rest.
func (e RankedExecutable) Args(rest []string) []string {
	return append(append([]string{}, e.Prefix...), rest...)
}

// ResolveRankedExecutable materializes the embedded, digest-verified OpenSpec
// bundle and runs it with a validated host Node. OpenSpec is never resolved
// from PATH; only the host Node executable is discovered there.
func ResolveRankedExecutable(ctx context.Context, strategistDir, providerID, weaponVersion string, contract domain.WeaponRuntime) (RankedExecutable, *domain.RankedRuntimeStateRuntime, error) {
	return resolveHostNodeRuntime(ctx, strategistDir, providerID, weaponVersion, contract)
}

func resolveHostNodeRuntime(ctx context.Context, strategistDir, providerID, weaponVersion string, contract domain.WeaponRuntime) (RankedExecutable, *domain.RankedRuntimeStateRuntime, error) {
	node, err := ResolveHostNode(ctx, strategistDir)
	if err != nil {
		return RankedExecutable{}, nil, err
	}
	dest := filepath.Join(strategistDir, "weapon-runtime", providerID)
	// Materialize from this binary's embedded defaults, never from the copy
	// already extracted on disk. The on-disk tree carries its own
	// runtime.build.yaml, and TreeDigest deliberately excludes that file from
	// the digest it publishes, so anyone able to write the tree could rewrite
	// its certificate to match. Installation also preserves user-modified files
	// unless --force, so a stale or tampered tree survives a reinstall. The
	// embedded copy is the authority; the extracted one is not.
	defaults, ok := payload.DefaultOpenSpec()
	if !ok {
		return RankedExecutable{}, nil, fmt.Errorf("embedded OpenSpec defaults are unavailable; rebuild Strategist with the embedded skill bundle")
	}
	script, evidence, err := payload.MaterializeOpenSpec(defaults, dest, weaponVersion)
	if err != nil {
		return RankedExecutable{}, nil, fmt.Errorf("embedded OpenSpec runtime: %w", err)
	}
	state := &domain.RankedRuntimeStateRuntime{Node: node, Script: RelSlash(strategistDir, script)}
	for _, c := range evidence.Components {
		state.Components = append(state.Components, domain.RankedRuntimeStateComponent{Name: c.Name, Version: c.Version, SHA256: c.SHA256})
	}
	if err := CheckOpenSpecPin(contract, state); err != nil {
		return RankedExecutable{}, nil, err
	}
	return RankedExecutable{Name: node, Prefix: []string{script}}, state, nil
}

// ResolveHostNode discovers Node on PATH and returns its absolute path once it
// reports at least the minimum version the embedded OpenSpec bundle needs.
func ResolveHostNode(ctx context.Context, strategistDir string) (string, error) {
	node, err := FindHostNode("node")
	if err != nil {
		return "", &env.ExecutableNotFoundError{Name: "node", Err: err}
	}
	versionOut, err := HostNodeVersion(ctx, strategistDir, node)
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

// CheckOpenSpecPin fails closed when the embedded bundle is not the version
// the certified provider contract pins. An unpinned contract accepts any
// embedded bundle.
func CheckOpenSpecPin(contract domain.WeaponRuntime, state *domain.RankedRuntimeStateRuntime) error {
	for _, c := range state.Components {
		if c.Name == "openspec" && contract.Version != "" && contract.Version != c.Version {
			return fmt.Errorf("error=ranked_runtime_pin_mismatch: embedded OpenSpec is %s but the provider contract pins %s", c.Version, contract.Version)
		}
	}
	return nil
}

// RelSlash returns path relative to root with forward slashes.
func RelSlash(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(rel)
}
