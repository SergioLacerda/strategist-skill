package check

import (
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/trust"
	"gopkg.in/yaml.v3"
)

// trustPolicyFileName mirrors internal/embed/defaults/plugins/schemas/trust-policy.schema.yaml.
// No installer writes this file today — readTrustPolicy returns the same
// zero-value domain.TrustPolicy{} internal/install/embedded_skill_prepare.go
// already passes to IngestExternalSkills when none is configured, so reading
// a missing file introduces no new default behavior, only a real,
// file-backed override path for whoever eventually adds one.
const trustPolicyFileName = "trust-policy.yaml"

func readTrustPolicy(root string) domain.TrustPolicy {
	raw, err := os.ReadFile(filepath.Join(root, trustPolicyFileName)) //nolint:gosec // G304: fixed filename under the runtime root
	if err != nil {
		return domain.TrustPolicy{}
	}
	var p domain.TrustPolicy
	if yaml.Unmarshal(raw, &p) != nil {
		return domain.TrustPolicy{}
	}
	return p
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
// internal/install/embedded_skill_ingestion_filters.go already uses for
// external-skill ingestion, against whatever trust policy is actually
// configured (none, today, for every provider — embedded or external; no
// installer path persists one yet). This is a genuinely computed result, not
// a hardcoded "unknown": an empty policy has no trusted-publisher/-source
// requirements to fail, so it truthfully evaluates as trusted, and this will
// correctly start reporting real rejections the day an operator adds
// trust-policy.yaml or a future ingestion path starts persisting
// per-instance verification evidence.
func skillProviderTrustReadiness(root, provider, digest string) domain.ReadinessCheck {
	trustPolicy := readTrustPolicy(root)
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
// and no persisted domain.PermissionGrant record (none is written anywhere
// today). An empty Requested permission set — no skill.yaml declares one
// today — evaluates honestly to Ready ("nothing requested, nothing to
// grant"), and this will correctly start blocking the day a skill manifest
// declares a requested permission with no matching grant.
func skillProviderPermissionGrantReadiness(digest string) domain.ReadinessCheck {
	if digest == "" {
		return domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "permission_grant_not_evaluated", Detail: "no adapter_contract digest in plugins.lock"}
	}
	decision := policy.EvaluateGrant(policy.GrantRequest{PackageDigest: digest, AdapterDigest: digest}, domain.PermissionGrant{})
	if decision.Allowed {
		return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "no_permissions_requested"}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: decision.Reasons[0].Code}
}
