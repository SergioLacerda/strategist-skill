package check

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/conformance"
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

type rankedRuntimeStateCheck struct {
	Entries []struct {
		Slot           string `json:"slot"`
		Provider       string `json:"provider"`
		ContractDigest string `json:"contract_digest"`
	} `json:"entries"`
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
	return runRankedRuntimeHealthcheck(runtimeRoot, provider)
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
			return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_digest_mismatch", Detail: fmt.Sprintf("provider=%s expected=%s observed=%s", provider, expectedDigest, entry.ContractDigest)}
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "openspec", "context", "--json")
	cmd.Dir = runtimeRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "ranked_runtime_healthcheck_failed", Detail: fmt.Sprintf("provider=%s root=%s error=%v output=%s", provider, runtimeRoot, err, strings.TrimSpace(string(output)))}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "ranked_runtime_healthy", Detail: runtimeRoot}
}

// evaluateRankedConformance builds a conformance.CertificationRecord from
// stamp's persisted digests (ADR-0043 DEC-006, generic over every Ranked
// (role, provider) pairing — no pairing-specific logic here) and evaluates
// it via conformance.EvaluateCertification. HostAPIDigest is additionally
// recomputed live from the target workspace's own roles/<role>.yaml +
// internal_skills/<role>/SKILL.md — these do materialize into
// .strategist/, unlike the connector/test-suite source, which are
// Strategist-tool internals compiled into the binary and never ship to a
// target workspace (see embedded_skill_conformance.go's own doc comments
// in the install package for why). A mismatch there means the role/skill
// file was edited locally since certification — genuine staleness.
// ConnectorDigest/TestSuiteDigest have no independently verifiable live
// counterpart outside a fresh Strategist build, so input intentionally
// mirrors record for those two dimensions: real, content-derived values,
// just not re-verifiable from inside a target workspace.
func evaluateRankedConformance(root, slot string, stamp domain.CatalogRankedStamp) conformance.CertificationResult {
	record := conformance.CertificationRecord{
		SchemaVersion:   "strategist-conformance-record/v1",
		Level:           conformance.Level(stamp.ConformanceLevel),
		PackageDigest:   stamp.CertificationDigest,
		AdapterDigest:   stamp.CertificationDigest,
		HostAPIDigest:   stamp.HostAPIDigest,
		ConnectorDigest: stamp.ConnectorDigest,
		TestSuiteDigest: stamp.TestSuiteDigest,
		CertifiedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	input := conformance.CertificationInputDigests{
		PackageDigest:   stamp.CertificationDigest,
		AdapterDigest:   stamp.CertificationDigest,
		HostAPIDigest:   liveHostAPIDigest(root, slot, stamp.HostAPIDigest),
		ConnectorDigest: stamp.ConnectorDigest,
		TestSuiteDigest: stamp.TestSuiteDigest,
	}
	return conformance.EvaluateCertification(record, conformance.LevelC1Contract, input, time.Now())
}

// liveHostAPIDigest recomputes the target workspace's current
// roles/<role>.yaml + internal_skills/<role>/SKILL.md digest, so a local
// edit since certification is detected as certification_stale by
// evaluateRankedConformance. Falls back to fallback (never flags stale)
// when the role for slot cannot be resolved or the files cannot be read —
// an unrelated lookup failure must not itself block an otherwise-valid
// Ranked binding.
func liveHostAPIDigest(root, slot, fallback string) string {
	roleMap, err := loadRoleSlotMap(root)
	if err != nil {
		return fallback
	}
	role := roleMap[slot]
	if role == "" {
		return fallback
	}
	roleRaw, err := os.ReadFile(filepath.Join(root, "roles", role+".yaml")) //nolint:gosec // G304: fixed runtime path
	if err != nil {
		return fallback
	}
	skillRaw, err := os.ReadFile(filepath.Join(root, "internal_skills", role, "SKILL.md")) //nolint:gosec // G304: fixed runtime path
	if err != nil {
		return fallback
	}
	h := sha256.New()
	h.Write(roleRaw)
	h.Write(skillRaw)
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}
