// Package policy evaluates plugin permission grants and enforcement evidence.
package policy

import (
	"regexp"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

// GrantRequest is the immutable subject and permission set being activated.
type GrantRequest struct {
	PackageDigest string
	AdapterDigest string
	Requested     []domain.PluginPermission
}

// DecisionReason is a stable reason code with a short diagnostic detail.
type DecisionReason struct {
	Code   string
	Detail string
}

// GrantDecision reports whether an existing grant authorizes activation.
type GrantDecision struct {
	Allowed bool
	Reasons []DecisionReason
}

// EnforcementReport is connector-supplied evidence about technical enforcement.
type EnforcementReport struct {
	ConnectorID string
	Enforceable []domain.PluginPermission
	Observed    []domain.PluginPermission
	Limitations []string
}

// EnforcementDecision distinguishes granted permissions from enforceable ones.
type EnforcementDecision struct {
	ConnectorID   string
	Complete      bool
	Enforceable   []domain.PluginPermission
	Unenforceable []domain.PluginPermission
	Observed      []domain.PluginPermission
	Limitations   []string
}

// EvaluateGrant checks digest binding and blocks permission escalation.
func EvaluateGrant(req GrantRequest, grant domain.PermissionGrant) GrantDecision {
	var reasons []DecisionReason
	if !digestPattern.MatchString(req.PackageDigest) {
		reasons = append(reasons, DecisionReason{Code: "invalid_package_digest", Detail: req.PackageDigest})
	}
	if !digestPattern.MatchString(req.AdapterDigest) {
		reasons = append(reasons, DecisionReason{Code: "invalid_adapter_digest", Detail: req.AdapterDigest})
	}
	if req.PackageDigest != "" && grant.PackageDigest != "" && req.PackageDigest != grant.PackageDigest {
		reasons = append(reasons, DecisionReason{Code: "package_digest_mismatch", Detail: "grant is bound to a different package digest"})
	}
	if req.AdapterDigest != "" && grant.AdapterDigest != "" && req.AdapterDigest != grant.AdapterDigest {
		reasons = append(reasons, DecisionReason{Code: "adapter_digest_mismatch", Detail: "grant is bound to a different adapter digest"})
	}
	granted := permissionSet(grant.GrantedPermissions)
	for _, permission := range sortedPermissions(req.Requested) {
		if !IsKnownPermission(permission) {
			reasons = append(reasons, DecisionReason{Code: "unknown_requested_permission", Detail: string(permission)})
			continue
		}
		if !granted[permission] {
			reasons = append(reasons, DecisionReason{Code: "permission_escalation_requires_reconsent", Detail: string(permission)})
		}
	}
	return GrantDecision{Allowed: len(reasons) == 0, Reasons: reasons}
}

// EvaluateEnforcement reports which granted permissions the connector can constrain.
func EvaluateEnforcement(report EnforcementReport, granted []domain.PluginPermission) EnforcementDecision {
	enforceableSet := permissionSet(report.Enforceable)
	var unenforceable []domain.PluginPermission
	for _, permission := range sortedPermissions(granted) {
		if !enforceableSet[permission] {
			unenforceable = append(unenforceable, permission)
		}
	}
	return EnforcementDecision{
		ConnectorID:   report.ConnectorID,
		Complete:      len(unenforceable) == 0,
		Enforceable:   sortedPermissions(report.Enforceable),
		Unenforceable: unenforceable,
		Observed:      sortedPermissions(report.Observed),
		Limitations:   append([]string(nil), report.Limitations...),
	}
}

// IsKnownPermission reports whether permission belongs to the Strategist plugin vocabulary.
func IsKnownPermission(permission domain.PluginPermission) bool {
	switch permission {
	case domain.PluginPermissionReadWorkspace,
		domain.PluginPermissionWriteAnalysis,
		domain.PluginPermissionWriteDocs,
		domain.PluginPermissionWriteSource,
		domain.PluginPermissionNetworkAccess,
		domain.PluginPermissionSubprocess,
		domain.PluginPermissionExternalApp,
		domain.PluginPermissionSecretAccess:
		return true
	default:
		return false
	}
}

func permissionSet(permissions []domain.PluginPermission) map[domain.PluginPermission]bool {
	set := map[domain.PluginPermission]bool{}
	for _, permission := range permissions {
		set[permission] = true
	}
	return set
}

func sortedPermissions(permissions []domain.PluginPermission) []domain.PluginPermission {
	out := append([]domain.PluginPermission(nil), permissions...)
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}
