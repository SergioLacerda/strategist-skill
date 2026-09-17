package authorization

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
	"github.com/SergioLacerda/strategist-skill/internal/rolevalidation"
	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"gopkg.in/yaml.v3"
)

func loadActive(root string) (domain.ActiveConfig, error) {
	raw, err := os.ReadFile(filepath.Join(root, "active.yaml")) //nolint:gosec // root is resolved by the CLI
	if err != nil {
		return domain.ActiveConfig{}, fmt.Errorf("read active config: %w", err)
	}
	var active domain.ActiveConfig
	if err := yaml.Unmarshal(raw, &active); err != nil {
		return domain.ActiveConfig{}, fmt.Errorf("parse active config: %w", err)
	}
	if err := active.Validate(); err != nil {
		return domain.ActiveConfig{}, fmt.Errorf("validate active config: %w", err)
	}
	return active, nil
}

func bindingEvidence(root string, active domain.ActiveConfig) Dimension {
	for _, slot := range []string{"discovery", "refinement"} {
		plan, err := rolevalidation.BuildRoleInvocationPlan(root, slot)
		if err != nil {
			return Dimension{Name: "binding", Status: "blocked", EvidenceState: "failed", ReasonCode: bindingReason(err), Detail: fmt.Sprintf("slot=%s provider=%s: %v", slot, active.Slots[slot], err), Required: true}
		}
		if plan.WeaponID != active.Slots[slot] {
			return Dimension{Name: "binding", Status: "blocked", EvidenceState: "failed", ReasonCode: "active_binding_mismatch", Detail: fmt.Sprintf("slot=%s active=%s plan=%s", slot, active.Slots[slot], plan.WeaponID), Required: true}
		}
	}
	return Dimension{Name: "binding", Status: "ready", EvidenceState: "static", ReasonCode: "role_provider_binding_verified", Detail: "all slots resolve to their persisted Role→Weapon bindings", Required: true}
}

func loadRoleMap(root string) (domain.RoleSlotMap, error) {
	raw, err := os.ReadFile(filepath.Join(root, "roles", "default.yaml")) //nolint:gosec // root is resolved by the CLI
	if err != nil {
		return nil, fmt.Errorf("read role map: %w", err)
	}
	var roles domain.RoleSlotMap
	if err := yaml.Unmarshal(raw, &roles); err != nil {
		return nil, fmt.Errorf("parse role map: %w", err)
	}
	if err := roles.Validate(); err != nil {
		return nil, fmt.Errorf("validate role map: %w", err)
	}
	return roles, nil
}

func compiledArtifactEvidence(root string) Dimension {
	artifact := filepath.Join(root, ".compiled", ".config.gz")
	if _, err := os.Stat(artifact); os.IsNotExist(err) {
		return Dimension{}
	}
	result, err := (stale.Checker{}).Check(artifact)
	if err != nil || result.Stale {
		detail := result.Detail
		if err != nil {
			detail = err.Error()
		}
		return Dimension{Name: "runtime_artifact", Status: "stale", EvidenceState: "failed", ReasonCode: "compiled_runtime_stale", Detail: detail, Required: true}
	}
	return Dimension{Name: "runtime_artifact", Status: "ready", EvidenceState: "static", ReasonCode: "compiled_runtime_fresh", Detail: artifact, Required: true}
}

func gateEvidence(approval, execution string) Dimension {
	if execution == "" {
		execution = "allowed"
	}
	if execution != "allowed" {
		return Dimension{Name: "gate", Status: "blocked", EvidenceState: "policy", ReasonCode: "local_execution_gate_blocked", Detail: execution, Required: true}
	}
	if approval != "accepted" {
		if approval == "" {
			approval = "pending"
		}
		return Dimension{Name: "gate", Status: "blocked", EvidenceState: "approval", ReasonCode: "approval_gate_not_accepted", Detail: approval, Required: true}
	}
	return Dimension{Name: "gate", Status: "allowed", EvidenceState: "user_approved", ReasonCode: "approval_gate_accepted", Required: true}
}

func targetReason(decision policy.WriteDecision) string {
	if decision.Allowed {
		return "write_target_enforceable"
	}
	if strings.Contains(decision.Reason, "forbidden") {
		return "forbidden_planning_path"
	}
	return "write_target_not_enforceable"
}

func bindingReason(err error) string {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "digest mismatch"):
		return "ranked_runtime_digest_mismatch"
	case strings.Contains(message, "runtime"):
		return "provider_runtime_unavailable"
	default:
		return "role_provider_binding_invalid"
	}
}
