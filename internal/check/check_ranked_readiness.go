package check

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// rankedCertificationReadiness reports Trust and PermissionGrant as Ready
// with a reason citing the catalog's certification digest, or Blocked when
// the catalog cannot be read, provider is not (or no longer) certified, or
// its conformance evidence fails evaluation (ADR-0043 DEC-006) — never
// silently falling back to Custom's trust.Verify/policy.EvaluateGrant path.
// The installed runtime is reported separately as the dependencies dimension,
// so a runtime fault is never attributed to trust or permission grants.
func rankedCertificationReadiness(root, slot, provider string) (trustCheck, grantCheck, runtimeCheck domain.ReadinessCheck) {
	stamp, certification := certifiedRankedStamp(root, slot, provider)
	if !certification.Ready() {
		notEvaluated := domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "ranked_runtime_not_evaluated", Detail: "certification is blocked"}
		return certification, certification, notEvaluated
	}
	return certification, certification, rankedRuntimeReadiness(root, slot, provider, stamp)
}

func certifiedRankedStamp(root, slot, provider string) (domain.CatalogRankedStamp, domain.ReadinessCheck) {
	raw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // G304: fixed runtime path
	if err != nil {
		return domain.CatalogRankedStamp{}, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_catalog_unreadable", Detail: err.Error()}
	}
	stamp, ok, err := domain.FindCatalogRankedStamp(raw, provider)
	if err != nil {
		return domain.CatalogRankedStamp{}, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_catalog_invalid", Detail: err.Error()}
	}
	if !ok || !stamp.Certified() {
		return domain.CatalogRankedStamp{}, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_not_certified"}
	}
	if result := evaluateRankedConformance(root, slot, stamp); !result.Accepted {
		return domain.CatalogRankedStamp{}, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_conformance_rejected", Detail: strings.Join(result.ReasonCodes, ", ")}
	}
	return stamp, domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "ready_by_certification", Detail: stamp.CertificationDigest}
}

func rankedRuntimeReadiness(root, slot, provider string, stamp domain.CatalogRankedStamp) domain.ReadinessCheck {
	runtime := domain.NormalizeRankedRuntime(stamp.Runtime)
	if runtime.Kind == domain.RankedRuntimeHost || runtime.Kind == domain.RankedRuntimeExecutable {
		return domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "ranked_runtime_requires_host_invocation", Detail: fmt.Sprintf("provider=%s runtime=%s requires the declared host connector", provider, runtime.Kind)}
	}
	if result := validateRankedRuntimeContract(runtime, slot, provider); !result.Ready() || runtime.Kind == domain.RankedRuntimeNone {
		return result
	}
	if runtime.Kind == domain.RankedRuntimeEmbedded {
		return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "ranked_embedded_runtime_ready", Detail: fmt.Sprintf("provider=%s is executed by the Strategist embedded runtime", provider)}
	}
	runtimeRoot, recorded, result := recordedRankedRuntime(root, slot, provider, stamp, runtime)
	if !result.Ready() {
		return result
	}
	return runHostNodeRankedRuntimeHealthcheck(root, runtimeRoot, provider, runtime, recorded)
}

// recordedRankedRuntime loads the runtime state and returns the installed
// runtime root and the runtime recorded for the slot/provider binding.
func recordedRankedRuntime(root, slot, provider string, stamp domain.CatalogRankedStamp, runtime domain.RankedRuntimeContract) (string, domain.RankedRuntimeStateRuntime, domain.ReadinessCheck) {
	state, result := readRankedRuntimeState(root, provider, runtime.Root)
	if !result.Ready() {
		return "", domain.RankedRuntimeStateRuntime{}, result
	}
	entry, result := matchRankedRuntimeState(state, slot, provider, stamp.CertificationDigest, expectedRankedRole(root, slot), runtime.Root)
	if !result.Ready() {
		return "", domain.RankedRuntimeStateRuntime{}, result
	}
	runtimeRoot := rankedRuntimeRoot(root, runtime.Root)
	if result := validateRankedRuntimeRoot(runtimeRoot, provider); !result.Ready() {
		return "", domain.RankedRuntimeStateRuntime{}, result
	}
	if entry.Runtime == nil {
		return "", domain.RankedRuntimeStateRuntime{}, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeStateInvalid, Detail: fmt.Sprintf("provider=%s runtime state records no runtime; run `strategist upgrade` or `strategist install --wizard`", provider)}
	}
	return runtimeRoot, *entry.Runtime, domain.ReadinessCheck{Status: domain.ReadinessReady}
}

// expectedRankedRole is the role roles/default.yaml maps slot to, or "" when
// the map is unreadable (role compatibility reports that separately).
func expectedRankedRole(root, slot string) string {
	roles, err := loadRoleSlotMap(root)
	if err != nil {
		return ""
	}
	return roles[slot]
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

func readRankedRuntimeState(root, provider, runtimeRoot string) (domain.RankedRuntimeState, domain.ReadinessCheck) {
	stateRaw, err := os.ReadFile(filepath.Join(root, domain.RankedRuntimeStatePath)) //nolint:gosec // fixed path under the selected Strategist root
	if err != nil {
		return domain.RankedRuntimeState{}, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_state_missing", Detail: fmt.Sprintf("provider=%s root=%s remediation= reinstall Strategist", provider, runtimeRoot)}
	}
	state, err := domain.ParseRankedRuntimeState(stateRaw)
	var legacy *domain.RankedRuntimeStateLegacyError
	switch {
	case errors.As(err, &legacy):
		return domain.RankedRuntimeState{}, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeStateLegacy, Detail: fmt.Sprintf("provider=%s %v; run `strategist upgrade` or `strategist install --wizard`", provider, err)}
	case err != nil:
		return domain.RankedRuntimeState{}, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: domain.ReasonRankedRuntimeStateInvalid, Detail: fmt.Sprintf("provider=%s %v", provider, err)}
	}
	return state, domain.ReadinessCheck{Status: domain.ReadinessReady}
}

// matchRankedRuntimeState finds the entry recorded for slot/provider and
// requires its certification digest and role to match the current binding.
// An empty expectedRole skips the role comparison.
func matchRankedRuntimeState(state domain.RankedRuntimeState, slot, provider, expectedDigest, expectedRole, runtimeRoot string) (domain.RankedRuntimeStateEntry, domain.ReadinessCheck) {
	entry, ok := state.Entry(slot, provider)
	if !ok {
		return entry, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_binding_missing", Detail: fmt.Sprintf("provider=%s slot=%s root=%s", provider, slot, runtimeRoot)}
	}
	if entry.ContractDigest != expectedDigest {
		return entry, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_digest_mismatch", Detail: fmt.Sprintf("provider=%s expected=%s observed=%s remedy=run `strategist upgrade` (or `strategist install --wizard`) to re-record the runtime for this binary", provider, expectedDigest, entry.ContractDigest)}
	}
	if expectedRole != "" && entry.Role != expectedRole {
		return entry, domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_binding_missing", Detail: fmt.Sprintf("provider=%s slot=%s expected role=%s observed role=%s remedy=run `strategist install --wizard`", provider, slot, expectedRole, entry.Role)}
	}
	return entry, domain.ReadinessCheck{Status: domain.ReadinessReady}
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
