package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimeenv"
)

// rankedCertificationReadiness reports Trust and PermissionGrant as Ready
// with a reason citing the catalog's certification digest, or Blocked when
// the catalog cannot be read, provider is not (or no longer) certified, or
// its conformance evidence fails evaluation (ADR-0043 DEC-006) — never
// silently falling back to Custom's trust.Verify/policy.EvaluateGrant path.
func rankedCertificationReadiness(root, slot, provider string) (trustCheck, grantCheck domain.ReadinessCheck) {
	raw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // G304: fixed runtime path
	if err != nil {
		blocked := domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_catalog_unreadable", Detail: err.Error()}
		return blocked, blocked
	}
	stamp, ok, err := domain.FindCatalogRankedStamp(raw, provider)
	if err != nil {
		blocked := domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_catalog_invalid", Detail: err.Error()}
		return blocked, blocked
	}
	if !ok || !stamp.Certified() {
		blocked := domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_not_certified"}
		return blocked, blocked
	}
	if result := evaluateRankedConformance(root, slot, stamp); !result.Accepted {
		blocked := domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_conformance_rejected", Detail: strings.Join(result.ReasonCodes, ", ")}
		return blocked, blocked
	}
	if runtime := rankedRuntimeReadiness(root, slot, provider, stamp); !runtime.Ready() {
		return runtime, runtime
	}
	ready := domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "ready_by_certification", Detail: stamp.CertificationDigest}
	return ready, ready
}

// rankedRuntimeStatePrivate is the private runtime recorded at install time:
// slash paths relative to the Strategist root.
type rankedRuntimeStatePrivate struct {
	Node   string `json:"node"`
	Script string `json:"script"`
}

type rankedRuntimeStateEntryCheck struct {
	Slot           string                     `json:"slot"`
	Provider       string                     `json:"provider"`
	ContractDigest string                     `json:"contract_digest"`
	Runtime        *rankedRuntimeStatePrivate `json:"runtime"`
}

type rankedRuntimeStateCheck struct {
	Entries []rankedRuntimeStateEntryCheck `json:"entries"`
}

// privateRuntimeFor returns the private runtime recorded for slot/provider, if any.
func (s rankedRuntimeStateCheck) privateRuntimeFor(slot, provider string) *rankedRuntimeStatePrivate {
	for _, entry := range s.Entries {
		if entry.Slot == slot && entry.Provider == provider {
			return entry.Runtime
		}
	}
	return nil
}

func rankedRuntimeReadiness(root, slot, provider string, stamp domain.CatalogRankedStamp) domain.ReadinessCheck {
	runtime := domain.NormalizeRankedRuntime(stamp.Runtime)
	if result := validateRankedRuntimeContract(runtime, slot, provider); !result.Ready() || runtime.Kind == domain.RankedRuntimeNone {
		return result
	}
	state, result := readRankedRuntimeState(root, provider, runtime.Root)
	if !result.Ready() {
		return result
	}
	if result := matchRankedRuntimeState(state, slot, provider, stamp.CertificationDigest, runtime.Root); !result.Ready() {
		return result
	}
	runtimeRoot := rankedRuntimeRoot(root, runtime.Root)
	if result := validateRankedRuntimeRoot(runtimeRoot, provider); !result.Ready() {
		return result
	}
	return rankedRuntimeExecutableReadiness(root, runtimeRoot, slot, provider, runtime.Version, state)
}

func rankedRuntimeExecutableReadiness(root, runtimeRoot, slot, provider, version string, state rankedRuntimeStateCheck) domain.ReadinessCheck {
	if private := state.privateRuntimeFor(slot, provider); private != nil {
		return runPrivateRankedRuntimeHealthcheck(root, runtimeRoot, provider, *private)
	}
	return hostRankedRuntimeReadiness(runtimeRoot, provider, version)
}

func validateRankedRuntimeContract(runtime domain.RankedRuntimeContract, slot, provider string) domain.ReadinessCheck {
	if err := runtime.Validate(); err != nil {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_contract_invalid", Detail: fmt.Sprintf("role/provider=%s/%s: %v", slot, provider, err)}
	}
	if runtime.Kind == domain.RankedRuntimeNone {
		return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "ranked_runtime_not_required"}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessReady}
}

func readRankedRuntimeState(root, provider, runtimeRoot string) (rankedRuntimeStateCheck, domain.ReadinessCheck) {
	stateRaw, err := os.ReadFile(filepath.Join(root, "ranked-runtimes.yaml")) //nolint:gosec // fixed path under the selected Strategist root
	if err != nil {
		return rankedRuntimeStateCheck{}, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_state_missing", Detail: fmt.Sprintf("provider=%s root=%s remediation= reinstall Strategist", provider, runtimeRoot)}
	}
	var state rankedRuntimeStateCheck
	if err := json.Unmarshal(stateRaw, &state); err != nil {
		return rankedRuntimeStateCheck{}, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_state_invalid", Detail: err.Error()}
	}
	return state, domain.ReadinessCheck{Status: domain.ReadinessReady}
}

func matchRankedRuntimeState(state rankedRuntimeStateCheck, slot, provider, expectedDigest, runtimeRoot string) domain.ReadinessCheck {
	for _, entry := range state.Entries {
		if entry.Slot != slot || entry.Provider != provider {
			continue
		}
		if entry.ContractDigest != expectedDigest {
			return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_digest_mismatch", Detail: fmt.Sprintf("provider=%s expected=%s observed=%s remedy=run `strategist upgrade` (or `strategist install --wizard`) to re-record the runtime for this binary", provider, expectedDigest, entry.ContractDigest)}
		}
		return domain.ReadinessCheck{Status: domain.ReadinessReady}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_binding_missing", Detail: fmt.Sprintf("provider=%s slot=%s root=%s", provider, slot, runtimeRoot)}
}

func rankedRuntimeRoot(root, runtimeRoot string) string {
	rootRel := strings.TrimPrefix(filepath.ToSlash(runtimeRoot), ".strategist/")
	return filepath.Join(root, filepath.FromSlash(rootRel))
}

func validateRankedRuntimeRoot(runtimeRoot, provider string) domain.ReadinessCheck {
	if _, err := os.Stat(filepath.Join(runtimeRoot, "config.yaml")); err != nil {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_root_missing", Detail: fmt.Sprintf("provider=%s root=%s remediation= reinstall Strategist", provider, runtimeRoot)}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessReady}
}

func runRankedRuntimeHealthcheck(runtimeRoot, provider string) domain.ReadinessCheck {
	ctx, cancel := context.WithTimeout(context.Background(), rankedHealthcheckTimeout())
	defer cancel()
	cmd, err := runtimeenv.Command(ctx, runtimeRoot, "openspec", "context", "--json")
	if err != nil {
		var missing *runtimeenv.ExecutableNotFoundError
		if errors.As(err, &missing) {
			return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeExecutableMissing, Detail: domain.RankedRuntimeExecutableMissingMessage(provider, missing.Name)}
		}
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_healthcheck_failed", Detail: fmt.Sprintf("provider=%s root=%s error=%v", provider, runtimeRoot, err)}
	}
	return finishRankedRuntimeHealthcheck(ctx, cmd, runtimeRoot, provider)
}

func finishRankedRuntimeHealthcheck(ctx context.Context, cmd *exec.Cmd, runtimeRoot, provider string) domain.ReadinessCheck {
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeHealthcheckTimeout, Detail: fmt.Sprintf("provider=%s root=%s the runtime did not answer within %s; raise the limit with %s (for example %s=60s) and rerun", provider, runtimeRoot, rankedHealthcheckTimeout(), rankedHealthcheckTimeoutEnv, rankedHealthcheckTimeoutEnv)}
		}
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_healthcheck_failed", Detail: fmt.Sprintf("provider=%s root=%s error=%v output=%s", provider, runtimeRoot, err, strings.TrimSpace(string(output)))}
	}
	if err := domain.ValidateOpenSpecHealthcheck(output, runtimeRoot); err != nil {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_root_mismatch", Detail: fmt.Sprintf("provider=%s root=%s error=%v", provider, runtimeRoot, err)}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "ranked_runtime_healthy", Detail: runtimeRoot}
}

// runPrivateRankedRuntimeHealthcheck runs the healthcheck from the private
// runtime recorded at install time. It never consults PATH; recorded paths must
// stay under weapon-runtime/ so a tampered state file cannot point elsewhere.
func runPrivateRankedRuntimeHealthcheck(root, runtimeRoot, provider string, private rankedRuntimeStatePrivate) domain.ReadinessCheck {
	node, script, ok := privateRuntimePaths(root, private)
	if !ok {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_state_invalid", Detail: fmt.Sprintf("provider=%s private runtime paths must be relative and under weapon-runtime/", provider)}
	}
	ctx, cancel := context.WithTimeout(context.Background(), rankedHealthcheckTimeout())
	defer cancel()
	cmd, err := runtimeenv.PrivateCommand(ctx, runtimeRoot, node, script, "context", "--json")
	if err != nil {
		var missing *runtimeenv.ExecutableNotFoundError
		if errors.As(err, &missing) {
			return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeExecutableMissing, Detail: fmt.Sprintf("provider=%s private runtime executable %s is missing; reinstall Strategist to materialize it", provider, private.Node)}
		}
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_healthcheck_failed", Detail: fmt.Sprintf("provider=%s root=%s error=%v", provider, runtimeRoot, err)}
	}
	return finishRankedRuntimeHealthcheck(ctx, cmd, runtimeRoot, provider)
}
