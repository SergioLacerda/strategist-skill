package check

import (
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/governance"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/trust"
	"gopkg.in/yaml.v3"
)

func readTrustPolicy(root string) domain.TrustPolicy {
	policy, _, err := governance.Load(root)
	if err != nil {
		return domain.TrustPolicy{}
	}
	return policy
}

// readPluginsLockFile reads plugins.lock the same tolerant way
// checkPluginLockParity does: a missing or unreadable file is not an error,
// it just means no digest is available yet — never a reason to block a
// readiness dimension that has nothing to do with lock presence.
func readPluginsLockFile(root string) domain.PluginLockFile {
	raw, err := os.ReadFile(filepath.Join(root, "plugins.lock")) //nolint:gosec // G304: fixed path under the runtime root
	if err != nil {
		return domain.PluginLockFile{}
	}
	var lock domain.PluginLockFile
	if yaml.Unmarshal(raw, &lock) != nil {
		return domain.PluginLockFile{}
	}
	return lock
}

// skillProviderTrustReadiness runs the same trust.Verify pipeline
// internal/install/embedded_weapon_ingestion_filters.go already uses for
// external-skill ingestion, against whatever trust policy is actually
// configured (none, by default, for every provider — embedded or external).
// This is a genuinely computed result, not
// a hardcoded "unknown": an empty policy has no trusted-publisher/-source
// requirements to fail, so it truthfully evaluates as trusted, and this will
// correctly start reporting real rejections the day an operator adds
// trust-policy.yaml or a future ingestion path starts persisting
// per-instance verification evidence.
func skillProviderTrustReadiness(root, provider, digest string) domain.ReadinessCheck {
	trustPolicy, _, err := governance.Load(root)
	if err != nil {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "governance_state_invalid", Detail: err.Error()}
	}
	result := trust.Verify(trust.Subject{Package: domain.PluginPackage{ID: provider, Digest: digest}}, trustPolicy, time.Now())
	if result.Trusted {
		reason := "trust_verified"
		if trustPolicy.Revision == "" {
			reason = "no_trust_policy_configured"
		}
		return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: reason}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: trustBlockReasonCode(result)}
}

func trustBlockReasonCode(result trust.Result) string {
	if len(result.Reasons) == 0 {
		return "trust_verification_failed"
	}
	return result.Reasons[0].Code
}

// skillProviderPermissionGrantReadiness runs the same policy.EvaluateGrant
// digest/permission-escalation check the plugin lifecycle store's activation
// path uses, with today's real inputs: the provider's own adapter_contract
// digest from plugins.lock (no separate package-vs-adapter digest split
// exists in the current lock format, so the same digest stands in for both)
// and the persisted domain.PermissionGrant record when one exists. An empty
// Requested permission set — no skill.yaml declares one today — evaluates
// honestly to Ready ("nothing requested, nothing to grant"), and a skill
// manifest declaring a requested permission is blocked without a matching
// grant.
func skillProviderPermissionGrantReadiness(digest string) domain.ReadinessCheck {
	return skillProviderPermissionGrantReadinessFor("", digest, nil)
}

func skillProviderPermissionGrantReadinessFor(root, digest string, requested []domain.PluginPermission) domain.ReadinessCheck {
	if digest == "" {
		return domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "permission_grant_not_evaluated", Detail: "no adapter_contract digest in plugins.lock"}
	}
	var grant domain.PermissionGrant
	if root != "" {
		_, grantFile, err := governance.Load(root)
		if err != nil {
			return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "governance_state_invalid", Detail: err.Error()}
		}
		grant, _ = governance.FindGrant(grantFile, digest, digest)
	}
	if len(requested) > 0 && grant.ID == "" {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "permission_grant_missing"}
	}
	decision := policy.EvaluateGrant(policy.GrantRequest{PackageDigest: digest, AdapterDigest: digest, Requested: requested}, grant)
	if decision.Allowed {
		return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "no_permissions_requested"}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: decision.Reasons[0].Code}
}
