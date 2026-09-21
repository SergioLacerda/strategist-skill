package install

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
)

// payloadSource returns the runtime payload compiled into this binary. It is
// a variable so tests can inject one; ordinary builds have none.
var payloadSource = runtimepayload.Default

// rankedExecutable is how a Ranked provider command is launched: a bare name
// resolved from PATH (host/custom), or an absolute private launcher plus a
// leading script argument.
type rankedExecutable struct {
	name   string
	prefix []string
}

var hostOpenSpec = rankedExecutable{name: "openspec"}

func (e rankedExecutable) args(rest []string) []string {
	return append(append([]string{}, e.prefix...), rest...)
}

// resolveRankedExecutable materializes the embedded, digest-verified private
// runtime when this binary carries one, and never consults PATH in that case.
// Without an embedded payload it returns the host executable (stage (a):
// missing tools fail with an actionable diagnostic instead).
func resolveRankedExecutable(strategistDir, providerID string, contract domain.RankedRuntimeContract) (rankedExecutable, *rankedRuntimeStateRuntime, error) {
	payload, ok := payloadSource()
	if !ok {
		return hostOpenSpec, nil, nil
	}
	rel := filepath.Join("weapon-runtime", providerID)
	dest := filepath.Join(strategistDir, rel)
	evidence, err := runtimepayload.Materialize(payload.FS, payload.Manifest, runtime.GOOS, runtime.GOARCH, dest)
	if err != nil {
		return rankedExecutable{}, nil, fmt.Errorf("private runtime: %w", err)
	}
	node, script, err := payload.Manifest.LauncherPaths(dest, runtime.GOOS)
	if err != nil {
		return rankedExecutable{}, nil, fmt.Errorf("private runtime: %w", err)
	}
	state := &rankedRuntimeStateRuntime{Node: relSlash(strategistDir, node), Script: relSlash(strategistDir, script)}
	for _, c := range evidence.Components {
		state.Components = append(state.Components, rankedRuntimeStateComponent{Name: c.Name, Version: c.Version, SHA256: c.SHA256})
	}
	if err := checkPrivateRuntimePins(contract, state); err != nil {
		return rankedExecutable{}, nil, err
	}
	return rankedExecutable{name: node, prefix: []string{script}}, state, nil
}

// checkPrivateRuntimePins fails closed when the materialized components are not
// the versions the certified provider contract pins (openspec and node). An
// unpinned contract accepts any payload.
func checkPrivateRuntimePins(contract domain.RankedRuntimeContract, state *rankedRuntimeStateRuntime) error {
	want := map[string]string{"openspec": contract.Version, "node": contract.NodeVersion}
	for _, c := range state.Components {
		if pin := want[c.Name]; pin != "" && pin != c.Version {
			return fmt.Errorf("error=ranked_runtime_pin_mismatch: private runtime %s is %s but the provider contract pins %s", c.Name, c.Version, pin)
		}
	}
	return nil
}

// hostVersionSkewMessage probes a host executable's --version and returns a
// warning when it differs from the contract pin ("" when it matches or the
// contract does not pin). It never fails the install: certification already
// covers the pinned identity, and a host executable is the uncertified path.
func hostVersionSkewMessage(ctx context.Context, root string, exe rankedExecutable, contract domain.RankedRuntimeContract, providerID string) string {
	if contract.Version == "" {
		return ""
	}
	out, err := runRankedRuntimeCommand(ctx, root, exe.name, exe.args([]string{"--version"})...)
	observed := ""
	if err == nil {
		observed = domain.ParseReportedVersion(out)
	}
	if !domain.VersionSkew(contract.Version, observed) {
		return ""
	}
	return domain.RankedRuntimeVersionSkewMessage(providerID, contract.Version, observed)
}

func relSlash(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(rel)
}
