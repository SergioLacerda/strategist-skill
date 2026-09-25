package check

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimeenv"
	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
)

// runHostNodeRankedRuntimeHealthcheck uses the absolute Node path recorded at
// installation time. Only the bundled script can be relative, and it must be
// inside weapon-runtime/<provider>/openspec/; the bundle's tree digest is
// re-verified before Node runs it, so neither a tampered state nor an altered
// bundle is ever executed.
//
// A host Node that satisfies the embedded bundle's minimum but differs from the
// version the provider contract certified is reported as a non-blocking skew:
// the runtime exists to run on the Node the client already has, so the
// difference informs rather than blocks.
func runHostNodeRankedRuntimeHealthcheck(root, runtimeRoot, provider string, contract domain.WeaponRuntime, state domain.RankedRuntimeStateRuntime) domain.ReadinessCheck {
	script, result := verifiedRankedBundle(root, provider, state)
	if !result.Ready() {
		return result
	}
	ctx, cancel := context.WithTimeout(context.Background(), rankedHealthcheckTimeout())
	defer cancel()
	observed, versionCheck := hostNodeVersionCheck(ctx, runtimeRoot, provider, state.Node)
	if !versionCheck.Ready() {
		return versionCheck
	}
	cmd, err := runtimeenv.PrivateCommand(ctx, runtimeRoot, state.Node, script, "context", "--json")
	if err != nil {
		return hostNodeCommandFailure(provider, runtimeRoot, err)
	}
	health := finishRankedRuntimeHealthcheck(ctx, cmd, runtimeRoot, provider)
	if !health.Ready() || !domain.VersionSkew(contract.NodeVersion, observed) {
		return health
	}
	return domain.ReadinessCheck{
		Status:     domain.ReadinessReady,
		ReasonCode: domain.ReasonRankedRuntimeVersionSkew,
		Detail:     domain.RankedRuntimeNodeVersionSkewMessage(provider, contract.NodeVersion, observed),
	}
}

// verifiedRankedBundle validates the recorded paths and the materialized
// bundle, returning the absolute launcher script.
func verifiedRankedBundle(root, provider string, state domain.RankedRuntimeStateRuntime) (string, domain.ReadinessCheck) {
	if !filepath.IsAbs(state.Node) {
		return "", rankedStateInvalid(provider, "host Node path must be absolute")
	}
	script, ok := recordedScriptPath(root, provider, state.Script)
	if !ok {
		return "", rankedStateInvalid(provider, "embedded OpenSpec script must be relative and under weapon-runtime/<provider>/openspec/")
	}
	component, ok := state.Component(domain.RankedRuntimeComponentOpenSpec)
	if !ok || component.SHA256 == "" {
		return "", rankedStateInvalid(provider, "runtime state records no OpenSpec bundle digest")
	}
	return script, rankedBundleCheck(root, provider, script, component.SHA256)
}

func rankedBundleCheck(root, provider, script, digest string) domain.ReadinessCheck {
	remedy := "run `strategist upgrade` or `strategist install --wizard` to re-materialize it"
	if info, err := os.Stat(script); err != nil || info.IsDir() {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeBundleMissing, Detail: fmt.Sprintf("provider=%s OpenSpec launcher %s is missing; %s", provider, script, remedy)}
	}
	dir, _ := rankedRuntimeDir(provider)
	err := runtimepayload.VerifyMaterializedOpenSpec(filepath.Join(root, filepath.FromSlash(dir)), digest)
	switch {
	case err == nil:
		return domain.ReadinessCheck{Status: domain.ReadinessReady}
	case errors.Is(err, runtimepayload.ErrPayloadMissing):
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeBundleMissing, Detail: fmt.Sprintf("provider=%s %v; %s", provider, err, remedy)}
	default:
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeBundleAltered, Detail: fmt.Sprintf("provider=%s OpenSpec bundle does not match the digest recorded at install (%v); %s", provider, err, remedy)}
	}
}

func rankedStateInvalid(provider, reason string) domain.ReadinessCheck {
	return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeStateInvalid, Detail: fmt.Sprintf("provider=%s %s", provider, reason)}
}

// hostNodeVersionCheck separates the three ways a host Node can fail to be
// usable: it is absent, it runs but fails, or it reports a version below the
// minimum the embedded OpenSpec bundle requires. Collapsing the middle case
// into "unsupported version" would report a corrupt or wrong-architecture Node
// as an old one.
func hostNodeVersionCheck(ctx context.Context, runtimeRoot, provider, node string) (observed string, check domain.ReadinessCheck) {
	cmd, err := runtimeenv.PrivateCommand(ctx, runtimeRoot, node, "--version")
	if err != nil {
		return "", hostNodeCommandFailure(provider, runtimeRoot, err)
	}
	out, err := cmd.Output()
	if err != nil {
		return "", domain.ReadinessCheck{
			Status:     domain.ReadinessBlocked,
			ReasonCode: "ranked_runtime_healthcheck_failed",
			Detail:     fmt.Sprintf("provider=%s host Node %q could not be executed: %v", provider, node, err),
		}
	}
	observed = domain.ParseReportedVersion(out)
	if !domain.VersionAtLeast(observed, domain.MinimumOpenSpecNodeVersion) {
		return observed, domain.ReadinessCheck{
			Status:     domain.ReadinessBlocked,
			ReasonCode: "ranked_runtime_host_node_unsupported",
			Detail:     fmt.Sprintf("provider=%s Node %q does not satisfy >=%s", provider, observed, domain.MinimumOpenSpecNodeVersion),
		}
	}
	return observed, domain.ReadinessCheck{Status: domain.ReadinessReady}
}

func hostNodeCommandFailure(provider, runtimeRoot string, err error) domain.ReadinessCheck {
	var missing *runtimeenv.ExecutableNotFoundError
	if errors.As(err, &missing) {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeExecutableMissing, Detail: domain.RankedRuntimeExecutableMissingMessage(provider, "node")}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_healthcheck_failed", Detail: fmt.Sprintf("provider=%s root=%s error=%v", provider, runtimeRoot, err)}
}
